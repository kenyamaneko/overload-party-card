package usecasetest

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"
)

// DeckSetup はデッキのテストの前提となるマスターデータ・所持カード・所持陣営です。
type DeckSetup struct {
	Cards         []*domain.Card
	Products      []*domain.Product
	Initiatives   []*domain.Initiative
	OwnedCards    []domain.PlayerCard
	OwnedFactions []string
}

// NewDefaultDeckSetup は SHE 陣営のカード、3 つのプロダクトと施策、所持陣営 SHE を持つ標準の前提を返します。
func NewDefaultDeckSetup() DeckSetup {
	return DeckSetup{
		Cards: NewUnlimitedSheCards(CatalogCardCount),
		Products: []*domain.Product{
			NewTestProduct("TST-0001", FactionSHE),
			NewTestProduct("TST-0002", FactionTenki),
			NewTestProduct("TST-0003", FactionSHE),
		},
		Initiatives: []*domain.Initiative{
			NewTestInitiative("TST-1001", "TST-0001", "routine"),
			NewTestInitiative("TST-1002", "TST-0001", "special"),
			NewTestInitiative("TST-1003", "TST-0003", "routine"),
			NewTestInitiative("TST-1004", "TST-0003", "special"),
			NewTestInitiative("TST-1005", "TST-0002", "routine"),
			NewTestInitiative("TST-1006", "TST-0002", "special"),
		},
		OwnedFactions: []string{FactionSHE},
	}
}

// NewDeckSetupOwning は標準の前提に、指定のカードの指定と同じ枚数の所持カードを加えて返します。
func NewDeckSetupOwning(entries []domain.DeckCardEntry) DeckSetup {
	setup := NewDefaultDeckSetup()
	setup.OwnedCards = ToPlayerCards(entries)
	return setup
}

// NewDeckSetupWithReducedOwnership は合計 30 枚のデッキに必要な所持枚数のうち、先頭のカードだけ 1 枚足りない前提を返します。
func NewDeckSetupWithReducedOwnership() DeckSetup {
	setup := NewDeckSetupOwning(BuildDeckCardEntries(ValidDeckTotal))
	setup.OwnedCards[0].Count = DeckCopiesPerCard - 1
	return setup
}

// NewDeckSetupWithoutProduct は合計 30 枚のデッキに必要な所持枚数があり、プロダクト TST-0001 が存在しない前提を返します。
func NewDeckSetupWithoutProduct() DeckSetup {
	setup := NewDeckSetupOwning(BuildDeckCardEntries(ValidDeckTotal))
	setup.Products = []*domain.Product{
		NewTestProduct("TST-0002", FactionTenki),
		NewTestProduct("TST-0003", FactionSHE),
	}
	return setup
}

// DeckHarness はデッキの usecase と、それが使う偽物の組です。
type DeckHarness struct {
	Interactor  *usecase.DeckInteractor
	DeckRepo    *FakeDeckRepository
	PlayerCards *FakePlayerCardRepository
	Factions    *FakeFactionClient
	Clock       *FakeClock
}

// NewDeckHarness は指定の前提で DeckHarness を組み立てます。
func NewDeckHarness(t *testing.T, setup DeckSetup) *DeckHarness {
	t.Helper()
	deckRepo := NewFakeDeckRepository()
	playerCards := NewFakePlayerCardRepository()
	for _, c := range setup.OwnedCards {
		playerCards.SetCount(c.PlayerID, c.CardID, c.ArtNo, c.Count)
	}
	factions := &FakeFactionClient{Factions: map[string][]string{PlayerA: setup.OwnedFactions}}
	clock := &FakeClock{CurrentTime: time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)}
	interactor := usecase.NewDeckInteractor(
		deckRepo,
		playerCards,
		LoadCardCache(t, setup.Cards),
		LoadProductCache(t, setup.Products),
		LoadInitiativeCache(t, setup.Initiatives),
		factions,
		clock,
	)
	return &DeckHarness{
		Interactor:  interactor,
		DeckRepo:    deckRepo,
		PlayerCards: playerCards,
		Factions:    factions,
		Clock:       clock,
	}
}

// NewDraft は宣言陣営 SHE、プロダクト TST-0001 の標準の作成・更新の内容を、指定のカードの指定つきで返します。
func NewDraft(entries ...domain.DeckCardEntry) domain.DeckDraft {
	return domain.DeckDraft{
		DeckName:  "テストデッキ",
		Faction:   FactionSHE,
		ProductID: "TST-0001",
		RoutineID: "TST-1001",
		SpecialID: "TST-1002",
		Cards:     entries,
	}
}

// NewSavedDeck はプレイヤーAの保存済みのデッキを、カード構成なしで返します。
func NewSavedDeck() domain.Deck {
	return domain.Deck{
		PlayerID:  PlayerA,
		DeckName:  "保存済みデッキ",
		Faction:   FactionSHE,
		ProductID: "TST-0001",
		RoutineID: "TST-1001",
		SpecialID: "TST-1002",
	}
}

// SeedDeck は偽のデッキのデータベースへデッキを直接保存し、採番したデッキ ID を返します。
func (h *DeckHarness) SeedDeck(t *testing.T, deck domain.Deck, entries []domain.DeckCardEntry) int64 {
	t.Helper()
	deckID, err := h.DeckRepo.Create(t.Context(), deck, entries)
	require.NoError(t, err)
	return deckID
}

// CreateDeck はプレイヤーAのデッキを作成し、作成の応答を返します。
func (h *DeckHarness) CreateDeck(t *testing.T, draft domain.DeckDraft) *domain.DeckDetail {
	t.Helper()
	created, err := h.Interactor.CreateDeck(t.Context(), PlayerA, draft)
	require.NoError(t, err)
	return created
}

// FetchDeck はプレイヤーAの指定デッキを取得します。
func (h *DeckHarness) FetchDeck(t *testing.T, deckID int64) *domain.DeckDetail {
	t.Helper()
	fetched, err := h.Interactor.GetDeck(t.Context(), PlayerA, deckID)
	require.NoError(t, err)
	return fetched
}

// FindListedDeck はプレイヤーAのデッキ一覧から、指定の deck_id のデッキを特定して返します。
func (h *DeckHarness) FindListedDeck(t *testing.T, deckID int64) domain.DeckDetail {
	t.Helper()
	details, err := h.Interactor.GetDecks(t.Context(), PlayerA)
	require.NoError(t, err)
	idx := slices.IndexFunc(details, func(d domain.DeckDetail) bool { return d.Deck.DeckID == deckID })
	require.GreaterOrEqual(t, idx, 0, "デッキ一覧に deck_id %d のデッキが無い", deckID)
	return details[idx]
}
