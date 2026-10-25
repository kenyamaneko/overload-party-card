package usecasetest

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// FakeClock は固定の時刻を返す偽の時計です。
type FakeClock struct {
	CurrentTime time.Time
}

// Now は設定された固定の時刻を返します。
func (c *FakeClock) Now() time.Time {
	return c.CurrentTime
}

type deckKey struct {
	playerID string
	deckID   int64
}

// FakeDeckRepository は状態を保持する偽のデッキのデータベースです。
type FakeDeckRepository struct {
	decks  []domain.Deck
	cards  map[deckKey][]domain.DeckCard
	nextID int64

	CreateErr         error
	FindByPlayerIDErr error
	FindByIDErr       error
	GetDeckCardsErr   error
	UpdateErr         error
}

// NewFakeDeckRepository は空の FakeDeckRepository を生成します。
func NewFakeDeckRepository() *FakeDeckRepository {
	return &FakeDeckRepository{cards: map[deckKey][]domain.DeckCard{}}
}

func (r *FakeDeckRepository) findDeckIndex(playerID string, deckID int64) int {
	return slices.IndexFunc(r.decks, func(d domain.Deck) bool {
		return d.PlayerID == playerID && d.DeckID == deckID
	})
}

func convertToDeckCards(playerID string, deckID int64, entries []domain.DeckCardEntry) []domain.DeckCard {
	var cards []domain.DeckCard
	for _, e := range entries {
		cards = append(cards, domain.DeckCard{PlayerID: playerID, DeckID: deckID, CardID: e.CardID, ArtNo: e.ArtNo, Count: e.Count})
	}
	return cards
}

// Create はデッキとカード構成を保存し、採番したデッキ ID を返します。
func (r *FakeDeckRepository) Create(_ context.Context, deck domain.Deck, entries []domain.DeckCardEntry) (int64, error) {
	if r.CreateErr != nil {
		return 0, r.CreateErr
	}
	r.nextID++
	deck.DeckID = r.nextID
	r.decks = append(r.decks, deck)
	r.cards[deckKey{deck.PlayerID, deck.DeckID}] = convertToDeckCards(deck.PlayerID, deck.DeckID, entries)
	return deck.DeckID, nil
}

// FindByPlayerID は指定プレイヤーの全デッキを返します。
func (r *FakeDeckRepository) FindByPlayerID(_ context.Context, playerID string) ([]*domain.Deck, error) {
	if r.FindByPlayerIDErr != nil {
		return nil, r.FindByPlayerIDErr
	}
	var decks []*domain.Deck
	for _, d := range r.decks {
		if d.PlayerID == playerID {
			deck := d
			decks = append(decks, &deck)
		}
	}
	return decks, nil
}

// FindByID は指定プレイヤーの指定デッキを返します。
func (r *FakeDeckRepository) FindByID(_ context.Context, playerID string, deckID int64) (*domain.Deck, error) {
	if r.FindByIDErr != nil {
		return nil, r.FindByIDErr
	}
	idx := r.findDeckIndex(playerID, deckID)
	if idx < 0 {
		return nil, port.ErrNotFound
	}
	deck := r.decks[idx]
	return &deck, nil
}

// GetDeckCards は指定デッキのカード構成を返します。
func (r *FakeDeckRepository) GetDeckCards(_ context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
	if r.GetDeckCardsErr != nil {
		return nil, r.GetDeckCardsErr
	}
	return slices.Clone(r.cards[deckKey{playerID, deckID}]), nil
}

// Update はデッキを更新し、カード構成を指定の内容に差し替えます。作成日時は変えません。
func (r *FakeDeckRepository) Update(_ context.Context, deck domain.Deck, entries []domain.DeckCardEntry) error {
	if r.UpdateErr != nil {
		return r.UpdateErr
	}
	idx := r.findDeckIndex(deck.PlayerID, deck.DeckID)
	if idx < 0 {
		return port.ErrNotFound
	}
	deck.CreatedAt = r.decks[idx].CreatedAt
	r.decks[idx] = deck
	r.cards[deckKey{deck.PlayerID, deck.DeckID}] = convertToDeckCards(deck.PlayerID, deck.DeckID, entries)
	return nil
}

// Delete は指定デッキとそのカード構成を削除します。
func (r *FakeDeckRepository) Delete(_ context.Context, playerID string, deckID int64) error {
	idx := r.findDeckIndex(playerID, deckID)
	if idx < 0 {
		return port.ErrNotFound
	}
	r.decks = slices.Delete(r.decks, idx, idx+1)
	delete(r.cards, deckKey{playerID, deckID})
	return nil
}

type playerCardKey struct {
	playerID string
	cardID   string
	artNo    int64
}

// PlayerCardsSnapshot は FakePlayerCardRepository の所持枚数を退避したものです。
type PlayerCardsSnapshot struct {
	counts map[playerCardKey]int
}

// FakePlayerCardRepository は状態を保持する偽の所持カードのデータベースです。
type FakePlayerCardRepository struct {
	counts map[playerCardKey]int

	GetErr               error
	failingCardID        string
	failingAddErr        error
	remainingAddFailures int
}

// NewFakePlayerCardRepository は所持カードが無い FakePlayerCardRepository を生成します。
func NewFakePlayerCardRepository() *FakePlayerCardRepository {
	return &FakePlayerCardRepository{counts: map[playerCardKey]int{}}
}

// SetCount は指定の所持カードの枚数を直接設定します。
func (r *FakePlayerCardRepository) SetCount(playerID, cardID string, artNo int64, count int) {
	r.counts[playerCardKey{playerID, cardID, artNo}] = count
}

// TakeSnapshot は現在の所持枚数を退避します。
func (r *FakePlayerCardRepository) TakeSnapshot() PlayerCardsSnapshot {
	return PlayerCardsSnapshot{counts: maps.Clone(r.counts)}
}

// RestoreSnapshot は退避した所持枚数に戻します。
func (r *FakePlayerCardRepository) RestoreSnapshot(snapshot PlayerCardsSnapshot) {
	r.counts = snapshot.counts
}

// GetPlayerCards は指定プレイヤーの所持カードをカード ID・アート番号の昇順で返します。
func (r *FakePlayerCardRepository) GetPlayerCards(_ context.Context, playerID string) ([]*domain.PlayerCard, error) {
	if r.GetErr != nil {
		return nil, r.GetErr
	}
	var cards []*domain.PlayerCard
	for key, count := range r.counts {
		if key.playerID == playerID {
			cards = append(cards, &domain.PlayerCard{PlayerID: key.playerID, CardID: key.cardID, ArtNo: key.artNo, Count: count})
		}
	}
	slices.SortFunc(cards, func(a, b *domain.PlayerCard) int {
		return cmp.Or(cmp.Compare(a.CardID, b.CardID), cmp.Compare(a.ArtNo, b.ArtNo))
	})
	return cards, nil
}

// AddCards は指定の所持カードを所持枚数へ加算します。FailAddingOnce で設定した付与は失敗します。
func (r *FakePlayerCardRepository) AddCards(_ context.Context, cards []domain.PlayerCard) error {
	isFailing := r.remainingAddFailures > 0 && slices.ContainsFunc(cards, func(c domain.PlayerCard) bool {
		return c.CardID == r.failingCardID
	})
	if isFailing {
		r.remainingAddFailures--
		return r.failingAddErr
	}
	for _, c := range cards {
		r.counts[playerCardKey{c.PlayerID, c.CardID, c.ArtNo}] += c.Count
	}
	return nil
}

// FailAddingOnce は指定のカードを含む次の付与を、一度だけ指定のエラーで失敗させます。
func (r *FakePlayerCardRepository) FailAddingOnce(cardID string, err error) {
	r.failingCardID = cardID
	r.failingAddErr = err
	r.remainingAddFailures = 1
}

// FetchOwnedCards は指定プレイヤーの所持カードを取得し、値の一覧で返します。
func (r *FakePlayerCardRepository) FetchOwnedCards(t *testing.T, playerID string) []domain.PlayerCard {
	t.Helper()
	cards, err := r.GetPlayerCards(t.Context(), playerID)
	require.NoError(t, err)
	owned := make([]domain.PlayerCard, 0, len(cards))
	for _, c := range cards {
		owned = append(owned, *c)
	}
	return owned
}

// FakeFactionClient はプレイヤーごとの所持陣営を返す偽のaccountサービスです。
type FakeFactionClient struct {
	Factions map[string][]string
	Err      error
}

// ListPlayerFactions は指定プレイヤーの所持陣営を返します。
func (c *FakeFactionClient) ListPlayerFactions(_ context.Context, playerID string) ([]string, error) {
	if c.Err != nil {
		return nil, c.Err
	}
	return slices.Clone(c.Factions[playerID]), nil
}

// FakeCardRepository は偽のカードマスターのデータベースです。
type FakeCardRepository struct {
	Cards []*domain.Card
	Err   error
}

// FindAll は保持しているカード定義を返します。
func (r *FakeCardRepository) FindAll(_ context.Context) ([]*domain.Card, error) {
	return slices.Clone(r.Cards), r.Err
}

// FakeProductRepository は偽のプロダクトのデータベースです。
type FakeProductRepository struct {
	Products []*domain.Product
}

// FindAll は保持しているプロダクトを返します。
func (r *FakeProductRepository) FindAll(_ context.Context) ([]*domain.Product, error) {
	return slices.Clone(r.Products), nil
}

// FakeInitiativeRepository は偽の施策のデータベースです。
type FakeInitiativeRepository struct {
	Initiatives []*domain.Initiative
}

// FindAll は保持している施策を返します。
func (r *FakeInitiativeRepository) FindAll(_ context.Context) ([]*domain.Initiative, error) {
	return slices.Clone(r.Initiatives), nil
}
