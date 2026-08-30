package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	gamedesign "github.com/kenyamaneko/overload-party-common/packages/game-design-constants"
)

const (
	fxPlayerID   = "PLR-0001"
	fxFaction    = gamedesign.FactionSHE
	fxFaction2   = gamedesign.FactionTenki
	fxProductID  = "PD-0001"
	fxProductID2 = "PD-0002"
	fxRoutineID  = "IN-0001-R"
	fxSpecialID  = "IN-0001-S"
)

const fxProductsJSON = `[
  {"product_id":"PD-0001","faction":"SHE","product_name":"Product One","is_active":true},
  {"product_id":"PD-0002","faction":"Tenki","product_name":"Product Two","is_active":true}
]`

const fxInitiativesJSON = `[
  {"initiative_id":"IN-0001-R","product_id":"PD-0001","kind":"routine","name":"R1","insight_cost":100,"effect_text":"","effect":{}},
  {"initiative_id":"IN-0001-S","product_id":"PD-0001","kind":"special","name":"S1","insight_cost":200,"effect_text":"","effect":{}},
  {"initiative_id":"IN-0002-R","product_id":"PD-0002","kind":"routine","name":"R2","insight_cost":100,"effect_text":"","effect":{}},
  {"initiative_id":"IN-0002-S","product_id":"PD-0002","kind":"special","name":"S2","insight_cost":200,"effect_text":"","effect":{}}
]`

// fakeCardRepo は port.CardRepo のメモリバック実装。
type fakeCardRepo struct {
	cards      []*domain.Card
	findAllErr error
}

// FindAll は注入されたエラーがあればそれを優先し、なければ保持しているカード定義を返す。
func (r *fakeCardRepo) FindAll(_ context.Context) ([]*domain.Card, error) {
	if r.findAllErr != nil {
		return nil, r.findAllErr
	}
	return r.cards, nil
}

// fakePlayerCardRepo は port.PlayerCardRepo のメモリバック実装。
type fakePlayerCardRepo struct {
	cards         map[string][]*domain.PlayerCard
	getErr        error
	addErr        error
	addResult     int
	addedPlayerID string
	addedCards    []domain.CardPackCard
}

// newFakePlayerCardRepo は空の fakePlayerCardRepo を生成する。
func newFakePlayerCardRepo() *fakePlayerCardRepo {
	return &fakePlayerCardRepo{cards: make(map[string][]*domain.PlayerCard)}
}

// GetPlayerCards は注入されたエラーがあればそれを優先し、なければ指定プレイヤーの所持カードを返す。
func (r *fakePlayerCardRepo) GetPlayerCards(_ context.Context, playerID string) ([]*domain.PlayerCard, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.cards[playerID], nil
}

// AddCards は呼び出し引数を記録し、注入されたエラーがあればそれを優先し、なければ注入された結果を返す。
func (r *fakePlayerCardRepo) AddCards(_ context.Context, playerID string, cards []domain.CardPackCard) (int, error) {
	r.addedPlayerID = playerID
	r.addedCards = cards
	if r.addErr != nil {
		return 0, r.addErr
	}
	return r.addResult, nil
}

// seed は事前所持カードを投入する。
func (r *fakePlayerCardRepo) seed(playerID string, cards ...*domain.PlayerCard) {
	for _, c := range cards {
		c.PlayerID = playerID
	}
	r.cards[playerID] = append(r.cards[playerID], cards...)
}

// fakeFactionClient は port.FactionClient のテスト用実装。
type fakeFactionClient struct {
	factions []string
	err      error
}

// ListPlayerFactions は注入されたエラーがあればそれを優先し、なければ設定済みの所持陣営を返す。
func (c *fakeFactionClient) ListPlayerFactions(_ context.Context, _ string) ([]string, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.factions, nil
}

// fakeCardPackRepo は port.CardPackRepo のテスト用実装。
type fakeCardPackRepo struct {
	pack   *domain.CardPack
	getErr error
}

// GetPack は注入されたエラーがあればそれを優先し、なければ設定済みのパック定義を返す。
func (r *fakeCardPackRepo) GetPack(_ context.Context, _ string) (*domain.CardPack, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.pack, nil
}

// fakeDeckRepo は port.DeckRepo のメモリバック実装。
type fakeDeckRepo struct {
	decks      map[int64]domain.Deck
	deckCards  map[int64][]domain.DeckCard
	nextDeckID int64

	createErr       error
	findByPlayerErr error
	findByIDErr     error
	getDeckCardsErr error
	updateErr       error
	deleteErr       error

	// getDeckCardsOverride が設定されている場合、GetDeckCards は保存済みの内容ではなく
	// この内容を返す (再取得結果がリクエスト内容そのままの echo でないことを確かめる用途)。
	getDeckCardsOverride []domain.DeckCard
}

// newFakeDeckRepo は空の fakeDeckRepo を生成する。
func newFakeDeckRepo() *fakeDeckRepo {
	return &fakeDeckRepo{
		decks:      make(map[int64]domain.Deck),
		deckCards:  make(map[int64][]domain.DeckCard),
		nextDeckID: 1,
	}
}

// Create は注入されたエラーがあればそれを優先し、なければ採番した deck_id でデッキと構成カードを保存する。
func (r *fakeDeckRepo) Create(_ context.Context, deck domain.Deck, entries []domain.DeckCardEntry) (int64, error) {
	if r.createErr != nil {
		return 0, r.createErr
	}
	id := r.nextDeckID
	r.nextDeckID++
	deck.DeckID = id
	r.decks[id] = deck
	r.deckCards[id] = entriesToDeckCards(deck.PlayerID, id, entries)
	return id, nil
}

// FindByPlayerID は注入されたエラーがあればそれを優先し、なければ指定プレイヤーのデッキを返す。
func (r *fakeDeckRepo) FindByPlayerID(_ context.Context, playerID string) ([]*domain.Deck, error) {
	if r.findByPlayerErr != nil {
		return nil, r.findByPlayerErr
	}
	var out []*domain.Deck
	for id := range r.decks {
		if d := r.decks[id]; d.PlayerID == playerID {
			stored := d
			out = append(out, &stored)
		}
	}
	return out, nil
}

// FindByID は注入されたエラーがあればそれを優先し、なければ該当デッキを返す。
func (r *fakeDeckRepo) FindByID(_ context.Context, _ string, deckID int64) (*domain.Deck, error) {
	if r.findByIDErr != nil {
		return nil, r.findByIDErr
	}
	d, ok := r.decks[deckID]
	if !ok {
		return nil, port.ErrNotFound
	}
	return &d, nil
}

// GetDeckCards は注入されたエラーがあればそれを優先し、getDeckCardsOverride が設定されて
// いればそれを、なければ保存済みの指定デッキの構成カードを返す。
func (r *fakeDeckRepo) GetDeckCards(_ context.Context, _ string, deckID int64) ([]domain.DeckCard, error) {
	if r.getDeckCardsErr != nil {
		return nil, r.getDeckCardsErr
	}
	if r.getDeckCardsOverride != nil {
		return r.getDeckCardsOverride, nil
	}
	return r.deckCards[deckID], nil
}

// Update は注入されたエラーがあればそれを優先し、なければデッキと構成カードを差し替える。
func (r *fakeDeckRepo) Update(_ context.Context, deck domain.Deck, entries []domain.DeckCardEntry) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.decks[deck.DeckID] = deck
	r.deckCards[deck.DeckID] = entriesToDeckCards(deck.PlayerID, deck.DeckID, entries)
	return nil
}

// Delete は注入されたエラーがあればそれを優先し、なければ指定デッキを削除する。
func (r *fakeDeckRepo) Delete(_ context.Context, _ string, deckID int64) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.decks, deckID)
	delete(r.deckCards, deckID)
	return nil
}

// seed は事前にデッキと構成カードを投入する。
func (r *fakeDeckRepo) seed(deck domain.Deck, cards []domain.DeckCard) {
	r.decks[deck.DeckID] = deck
	r.deckCards[deck.DeckID] = cards
}

// entriesToDeckCards はリクエストの entry を永続化済み DeckCard 形式に詰め替える。
func entriesToDeckCards(playerID string, deckID int64, entries []domain.DeckCardEntry) []domain.DeckCard {
	cards := make([]domain.DeckCard, len(entries))
	for i, e := range entries {
		cards[i] = domain.DeckCard{PlayerID: playerID, DeckID: deckID, CardID: e.CardID, ArtNo: e.ArtNo, Count: e.Count}
	}
	return cards
}

// seedCard は cache.CardCache に 1 件のカード定義を投入する。
func seedCard(cc *cache.CardCache, cardID, faction, restriction string) {
	cc.InjectForTest(cardID, &domain.Card{
		CardID:        cardID,
		CardName:      "Test Card " + cardID,
		ResourceLabel: "vCPU",
		Faction:       faction,
		CardType:      gamedesign.CardTypeCompute,
		Restriction:   restriction,
		IsActive:      true,
		Stats:         json.RawMessage(`{}`),
	})
}

// baselineCardIDs は制限区分 unlimited (投入上限 3 枚) のカード 10 種類の ID を返す。
// 10 種 × 3 枚でデッキ上限枚数 (constants.DeckSize=30) ちょうどのデッキを組み立てるために使う。
func baselineCardIDs() []string {
	ids := make([]string, 10)
	for i := range ids {
		ids[i] = fmt.Sprintf("TST-%04d", i+1)
	}
	return ids
}

// seedBaselineCards は baselineCardIDs の全カードを SHE 陣営・unlimited 制限として
// cache.CardCache に投入する。
func seedBaselineCards(cc *cache.CardCache) {
	for _, id := range baselineCardIDs() {
		seedCard(cc, id, fxFaction, gamedesign.RestrictionUnlimited)
	}
}

// baselineCards は baselineCardIDs の各カードを 3 枚ずつ指定するカードエントリ (合計 30 枚) を返す。
func baselineCards() []apicard.DeckCardEntry {
	ids := baselineCardIDs()
	entries := make([]apicard.DeckCardEntry, len(ids))
	for i, id := range ids {
		entries[i] = apicard.DeckCardEntry{CardID: id, ArtNo: 1, Count: 3}
	}
	return entries
}

// baselineOwnedCards は baselineCards が要求する枚数をちょうど所持しているプレイヤー所持カードを返す。
func baselineOwnedCards() []*domain.PlayerCard {
	ids := baselineCardIDs()
	owned := make([]*domain.PlayerCard, len(ids))
	for i, id := range ids {
		owned[i] = &domain.PlayerCard{CardID: id, ArtNo: 1, Count: 3}
	}
	return owned
}

// newDeckFixture は正常系デッキ操作に必要な依存 (カード・プロダクト・施策) を投入済みの
// DeckInteractor と、振る舞いを差し替えるための fake を返す。factionClient は既定で
// fxFaction (SHE) を所持している状態にする。
func newDeckFixture(t *testing.T) (*DeckInteractor, *fakeDeckRepo, *fakePlayerCardRepo, *fakeFactionClient) {
	t.Helper()

	cc := cache.NewCardCache()
	seedBaselineCards(cc)
	seedCard(cc, "TST-NEUTRAL", gamedesign.FactionNeutral, gamedesign.RestrictionUnlimited)
	seedCard(cc, "TST-OTHERFACTION", fxFaction2, gamedesign.RestrictionUnlimited)
	seedCard(cc, "TST-LIMITED", fxFaction, gamedesign.RestrictionLimited)
	seedCard(cc, "TST-FORBIDDEN", fxFaction, gamedesign.RestrictionForbidden)
	seedCard(cc, "TST-UNKNOWNRESTRICTION", fxFaction, "premium")

	pc := cache.NewProductCache()
	require.NoError(t, pc.LoadFromBytes([]byte(fxProductsJSON)))
	ic := cache.NewInitiativeCache()
	require.NoError(t, ic.LoadFromBytes([]byte(fxInitiativesJSON)))

	deckRepo := newFakeDeckRepo()
	playerCardRepo := newFakePlayerCardRepo()
	factionClient := &fakeFactionClient{factions: []string{fxFaction}}

	interactor := NewDeckInteractor(deckRepo, playerCardRepo, cc, pc, ic, factionClient)
	return interactor, deckRepo, playerCardRepo, factionClient
}
