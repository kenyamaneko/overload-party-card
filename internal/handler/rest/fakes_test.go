package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"
)

const (
	fxPlayerID  = "p1"
	fxCardID    = "TST-0001"
	fxProductID = "PD-TST"
	fxRoutineID = "IN-TST-R"
	fxSpecialID = "IN-TST-S"
	fxFaction   = "SHE"
)

const fxProductsJSON = `[{"product_id":"PD-TST","faction":"SHE","product_name":"Test"}]`

const fxInitiativesJSON = `[
  {"initiative_id":"IN-TST-R","product_id":"PD-TST","kind":"routine","name":"R","insight_cost":100,"effect_text":"","effect":{"ops":[]}},
  {"initiative_id":"IN-TST-S","product_id":"PD-TST","kind":"special","name":"S","insight_cost":200,"effect_text":"","effect":{"ops":[]}}
]`

// fakeDeckRepo は port.DeckRepo のメモリバック実装。
type fakeDeckRepo struct {
	decks       map[int64]domain.Deck
	deckCards   map[int64][]domain.DeckCard
	nextDeckID  int64
	findByIDErr error
}

// newFakeDeckRepo は空の fakeDeckRepo を生成します。
func newFakeDeckRepo() *fakeDeckRepo {
	return &fakeDeckRepo{
		decks:      make(map[int64]domain.Deck),
		deckCards:  make(map[int64][]domain.DeckCard),
		nextDeckID: 1,
	}
}

// Create は採番した deck_id でデッキと構成カードを保存します。
func (r *fakeDeckRepo) Create(_ context.Context, deck domain.Deck, entries []domain.DeckCardEntry) (int64, error) {
	id := r.nextDeckID
	r.nextDeckID++
	deck.DeckID = id
	r.decks[id] = deck
	r.deckCards[id] = toDeckCards(deck.PlayerID, id, entries)
	return id, nil
}

// FindByPlayerID は指定プレイヤーのデッキを返します。
func (r *fakeDeckRepo) FindByPlayerID(_ context.Context, playerID string) ([]*domain.Deck, error) {
	var out []*domain.Deck
	for id := range r.decks {
		if d := r.decks[id]; d.PlayerID == playerID {
			stored := d
			out = append(out, &stored)
		}
	}
	return out, nil
}

// FindByID は注入されたエラーがあればそれを優先し、なければ該当デッキを返します。
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

// GetDeckCards は指定デッキの構成カードを返します。
func (r *fakeDeckRepo) GetDeckCards(_ context.Context, _ string, deckID int64) ([]domain.DeckCard, error) {
	return r.deckCards[deckID], nil
}

// Update はデッキと構成カードを差し替えます。
func (r *fakeDeckRepo) Update(_ context.Context, deck domain.Deck, entries []domain.DeckCardEntry) error {
	r.decks[deck.DeckID] = deck
	r.deckCards[deck.DeckID] = toDeckCards(deck.PlayerID, deck.DeckID, entries)
	return nil
}

// Delete は指定デッキを削除します。
func (r *fakeDeckRepo) Delete(_ context.Context, _ string, deckID int64) error {
	delete(r.decks, deckID)
	delete(r.deckCards, deckID)
	return nil
}

// toDeckCards はリクエストの entry を永続化済み DeckCard 形式に詰め替える。
func toDeckCards(playerID string, deckID int64, entries []domain.DeckCardEntry) []domain.DeckCard {
	cards := make([]domain.DeckCard, len(entries))
	for i, e := range entries {
		cards[i] = domain.DeckCard{
			PlayerID: playerID,
			DeckID:   deckID,
			CardID:   e.CardID,
			ArtNo:    e.ArtNo,
			Count:    e.Count,
		}
	}
	return cards
}

// fakePlayerCardRepo は port.PlayerCardRepo のメモリバック実装。
type fakePlayerCardRepo struct {
	cards  map[string][]*domain.PlayerCard
	getErr error
}

// newFakePlayerCardRepo は空の fakePlayerCardRepo を生成します。
func newFakePlayerCardRepo() *fakePlayerCardRepo {
	return &fakePlayerCardRepo{cards: make(map[string][]*domain.PlayerCard)}
}

// GetPlayerCards は注入されたエラーがあればそれを優先し、なければ指定プレイヤーの所持カードを返します。
func (r *fakePlayerCardRepo) GetPlayerCards(_ context.Context, playerID string) ([]*domain.PlayerCard, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.cards[playerID], nil
}

// AddCards は所持カードを追加し、加算したコピー総数を返します。
func (r *fakePlayerCardRepo) AddCards(_ context.Context, playerID string, cards []domain.CardPackCard) (int, error) {
	total := 0
	for _, c := range cards {
		r.cards[playerID] = append(r.cards[playerID], &domain.PlayerCard{PlayerID: playerID, CardID: c.CardID, Count: c.Copies})
		total += c.Copies
	}
	return total, nil
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
}

// ListPlayerFactions は設定済みの所持ファクションを返します。
func (c *fakeFactionClient) ListPlayerFactions(_ context.Context, _ string) ([]string, error) {
	return c.factions, nil
}

// fakeCardRepo は port.CardRepo のメモリバック実装。
type fakeCardRepo struct {
	cards      []*domain.Card
	findAllErr error
}

// newFakeCardRepo は空の fakeCardRepo を生成します。
func newFakeCardRepo() *fakeCardRepo {
	return &fakeCardRepo{}
}

// FindAll は注入されたエラーがあればそれを優先し、なければ保持しているカード定義を返します。
func (r *fakeCardRepo) FindAll(_ context.Context) ([]*domain.Card, error) {
	if r.findAllErr != nil {
		return nil, r.findAllErr
	}
	return r.cards, nil
}

// newDeckHandlerFixture は実 DeckInteractor を載せた DeckHandler と、
// 振る舞いを差し替えるための repo を返す。陣営 SHE のカード・プロダクト・施策を投入済み。
func newDeckHandlerFixture(t *testing.T) (*DeckHandler, *fakeDeckRepo, *fakePlayerCardRepo) {
	t.Helper()

	cc := cache.NewCardCache()
	cc.InjectForTest(fxCardID, &domain.Card{
		CardID: fxCardID, CardName: "Test Compute", Faction: fxFaction, CardType: "Compute",
		Restriction: "unlimited", IsActive: true, Stats: json.RawMessage(`{}`),
	})

	pc := cache.NewProductCache()
	require.NoError(t, pc.LoadFromBytes([]byte(fxProductsJSON)))
	ic := cache.NewInitiativeCache()
	require.NoError(t, ic.LoadFromBytes([]byte(fxInitiativesJSON)))

	deckRepo := newFakeDeckRepo()
	pcRepo := newFakePlayerCardRepo()
	fc := &fakeFactionClient{factions: []string{fxFaction}}

	interactor := usecase.NewDeckInteractor(deckRepo, pcRepo, cc, pc, ic, fc)
	return NewDeckHandler(interactor), deckRepo, pcRepo
}

// newCardHandlerFixture は実 CardInteractor を載せた CardHandler と、
// 振る舞いを差し替えるための repo を返す。
func newCardHandlerFixture() (*CardHandler, *fakeCardRepo, *fakePlayerCardRepo) {
	cardRepo := newFakeCardRepo()
	pcRepo := newFakePlayerCardRepo()
	interactor := usecase.NewCardInteractor(cardRepo, pcRepo)
	return NewCardHandler(interactor), cardRepo, pcRepo
}

// newPlayerCardHandlerFixture は実 PlayerCardInteractor を載せた PlayerCardHandler と、
// 振る舞いを差し替えるための repo を返す。
func newPlayerCardHandlerFixture() (*PlayerCardHandler, *fakePlayerCardRepo) {
	pcRepo := newFakePlayerCardRepo()
	interactor := usecase.NewPlayerCardInteractor(pcRepo, cache.NewCardCache())
	return NewPlayerCardHandler(interactor), pcRepo
}

// decodeBody は応答本文を JSON として復号する。原因を示す "error" フィールドを
// 完全一致で確かめるために使う (sentinel error はラップ元の接頭辞を含んだ
// 文言になり、部分一致の Contains では別の原因と取り違えるため)。
func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

// captureSlog は既定の slog logger を JSON 出力のテスト用 logger に差し替え、
// テスト終了後に元へ戻す。ログ出力内容を検証するために使う。
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(original) })
	return &buf
}

// decodeLogRecords は captureSlog で集めた JSON Lines ログを 1 行 1 レコードとして復号する。
func decodeLogRecords(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var records []map[string]any
	dec := json.NewDecoder(buf)
	for dec.More() {
		var rec map[string]any
		require.NoError(t, dec.Decode(&rec))
		records = append(records, rec)
	}
	return records
}
