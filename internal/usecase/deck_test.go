package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

type mockDeckRepo struct {
	decks      map[string][]*domain.Deck
	deckCards  map[int64][]domain.DeckCard
	nextDeckID int64
}

func newMockDeckRepo() *mockDeckRepo {
	return &mockDeckRepo{
		decks:      make(map[string][]*domain.Deck),
		deckCards:  make(map[int64][]domain.DeckCard),
		nextDeckID: 1,
	}
}

func (m *mockDeckRepo) Create(_ context.Context, deck domain.Deck, deckCardEntries []domain.DeckCardEntry) (int64, error) {
	id := m.nextDeckID
	m.nextDeckID++
	deck.DeckID = id
	stored := deck
	m.decks[stored.PlayerID] = append(m.decks[stored.PlayerID], &stored)
	cards := make([]domain.DeckCard, len(deckCardEntries))
	for i, e := range deckCardEntries {
		cards[i] = domain.DeckCard{
			PlayerID: stored.PlayerID,
			DeckID:   id,
			CardID:   e.CardID,
			ArtNo:    e.ArtNo,
			Count:    e.Count,
		}
	}
	m.deckCards[id] = cards
	return id, nil
}

func (m *mockDeckRepo) FindByPlayerID(_ context.Context, playerID string) ([]*domain.Deck, error) {
	return m.decks[playerID], nil
}

func (m *mockDeckRepo) FindByID(_ context.Context, playerID string, deckID int64) (*domain.Deck, error) {
	for _, d := range m.decks[playerID] {
		if d.DeckID == deckID {
			return d, nil
		}
	}
	return nil, fmt.Errorf("deck not found")
}

func (m *mockDeckRepo) GetDeckCards(_ context.Context, _ string, deckID int64) ([]domain.DeckCard, error) {
	return m.deckCards[deckID], nil
}

func (m *mockDeckRepo) Update(_ context.Context, deck domain.Deck, deckCardEntries []domain.DeckCardEntry) error {
	stored := deck
	for i, d := range m.decks[deck.PlayerID] {
		if d.DeckID == deck.DeckID {
			m.decks[deck.PlayerID][i] = &stored
			break
		}
	}
	cards := make([]domain.DeckCard, len(deckCardEntries))
	for i, e := range deckCardEntries {
		cards[i] = domain.DeckCard{
			PlayerID: deck.PlayerID,
			DeckID:   deck.DeckID,
			CardID:   e.CardID,
			ArtNo:    e.ArtNo,
			Count:    e.Count,
		}
	}
	m.deckCards[deck.DeckID] = cards
	return nil
}

func (m *mockDeckRepo) Delete(_ context.Context, playerID string, deckID int64) error {
	for i, d := range m.decks[playerID] {
		if d.DeckID == deckID {
			m.decks[playerID] = append(m.decks[playerID][:i], m.decks[playerID][i+1:]...)
			break
		}
	}
	delete(m.deckCards, deckID)
	return nil
}

func setupDeckInteractor(t *testing.T) (*DeckInteractor, *mockDeckRepo, *inMemoryPlayerCardRepo, *cache.CardCache) {
	t.Helper()
	repo := newMockDeckRepo()
	cc := cache.NewCardCache()

	unlimitedCards := []struct {
		id      string
		name    string
		faction string
		typ     string
	}{
		{"C-001", "Compute A", "SHE", "Compute"},
		{"C-002", "Compute B", "SHE", "Compute"},
		{"C-003", "Database A", "SHE", "Database"},
		{"C-004", "Strategy A", "SHE", "Strategy"},
		{"C-005", "Incident A", "SHE", "Incident"},
		{"C-006", "Platform A", "SHE", "Platform"},
		{"C-007", "Compute C", "Neutral", "Compute"},
		{"C-008", "Database B", "Neutral", "Database"},
		{"C-009", "Strategy B", "Neutral", "Strategy"},
		{"C-010", "Incident B", "Neutral", "Incident"},
		{"C-011", "Compute D", "Tenki", "Compute"},
	}
	for _, c := range unlimitedCards {
		cc.InjectForTest(c.id, &domain.Card{
			CardID: c.id, CardName: c.name, Faction: c.faction, CardType: c.typ,
			Restriction: "unlimited", IsActive: true,
			Stats: json.RawMessage(`{}`),
		})
	}

	cc.InjectForTest("C-050", &domain.Card{
		CardID: "C-050", CardName: "Limited Spell", Faction: "SHE", CardType: "Strategy",
		Restriction: "limited", IsActive: true,
		Stats: json.RawMessage(`{}`),
	})

	cc.InjectForTest("C-060", &domain.Card{
		CardID: "C-060", CardName: "SemiLimited Trap", Faction: "SHE", CardType: "Incident",
		Restriction: "semi_limited", IsActive: true,
		Stats: json.RawMessage(`{}`),
	})

	pc := cache.NewProductCache()
	require.NoError(t, pc.LoadFromBytes([]byte(testProductsJSON)))
	ic := cache.NewInitiativeCache()
	require.NoError(t, ic.LoadFromBytes([]byte(testInitiativesJSON)))

	pcRepo := newInMemoryPlayerCardRepo()
	svc := NewDeckInteractor(repo, pcRepo, cc, pc, ic)
	return svc, repo, pcRepo, cc
}

// testProductsJSON は SHE プロダクト PD-TST と別陣営 (Tenki) の PD-TST2 を持つテスト用定義。
const testProductID = "PD-TST"
const testRoutineID = "IN-TST-R"
const testSpecialID = "IN-TST-S"
const testProductsJSON = `[
  {"product_id":"PD-TST","faction":"SHE","product_name":"Test"},
  {"product_id":"PD-TST2","faction":"Tenki","product_name":"Other"}
]`

// testInitiativesJSON は PD-TST / PD-TST2 それぞれのルーチン・スペシャルを持つ。
// effect は検証に使わないため最小限。
const testInitiativesJSON = `[
  {"initiative_id":"IN-TST-R","product_id":"PD-TST","kind":"routine","name":"R","insight_cost":100,"effect_text":"","effect":{"ops":[]}},
  {"initiative_id":"IN-TST-S","product_id":"PD-TST","kind":"special","name":"S","insight_cost":200,"effect_text":"","effect":{"ops":[]}},
  {"initiative_id":"IN-TST-R2","product_id":"PD-TST2","kind":"routine","name":"R2","insight_cost":100,"effect_text":"","effect":{"ops":[]}},
  {"initiative_id":"IN-TST-S2","product_id":"PD-TST2","kind":"special","name":"S2","insight_cost":200,"effect_text":"","effect":{"ops":[]}}
]`

func grantCards(repo *inMemoryPlayerCardRepo, playerID string, cards ...*domain.PlayerCard) {
	for _, c := range cards {
		c.PlayerID = playerID
	}
	repo.Seed(playerID, cards)
}

func makeEntries(pairs ...interface{}) []apicard.DeckCardEntry {
	var entries []apicard.DeckCardEntry
	for i := 0; i < len(pairs); i += 2 {
		entries = append(entries, apicard.DeckCardEntry{
			CardID: pairs[i].(string),
			ArtNo:  0,
			Count:  pairs[i+1].(int),
		})
	}
	return entries
}

func grantUnlimited(repo *inMemoryPlayerCardRepo, playerID string, cardIDs ...string) {
	for _, id := range cardIDs {
		grantCards(repo, playerID, &domain.PlayerCard{CardID: id, ArtNo: 0, Count: 3})
	}
}

var allTenCards = []string{"C-001", "C-002", "C-003", "C-004", "C-005", "C-006", "C-007", "C-008", "C-009", "C-010"}

func full30Entries() []apicard.DeckCardEntry {
	return makeEntries(
		"C-001", 3, "C-002", 3, "C-003", 3, "C-004", 3, "C-005", 3,
		"C-006", 3, "C-007", 3, "C-008", 3, "C-009", 3, "C-010", 3,
	)
}

func TestCreateDeck_Validity(t *testing.T) {
	tests := []struct {
		name      string
		deckName  string
		grant     func(pcRepo *inMemoryPlayerCardRepo, pid string)
		entries   []apicard.DeckCardEntry
		wantValid bool
	}{
		{
			name:     "30枚ちょうどは有効",
			deckName: "Full Deck",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantUnlimited(pcRepo, pid, allTenCards...)
			},
			entries:   full30Entries(),
			wantValid: true,
		},
		{
			name:     "29枚は無効",
			deckName: "Almost Full",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantUnlimited(pcRepo, pid, "C-001", "C-002", "C-003", "C-004", "C-005", "C-006", "C-007", "C-008", "C-009")
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-010", ArtNo: 0, Count: 2})
			},
			entries: makeEntries(
				"C-001", 3, "C-002", 3, "C-003", 3, "C-004", 3, "C-005", 3,
				"C-006", 3, "C-007", 3, "C-008", 3, "C-009", 3, "C-010", 2,
			),
			wantValid: false,
		},
		{
			name:      "0枚は無効",
			deckName:  "Empty Deck",
			grant:     func(pcRepo *inMemoryPlayerCardRepo, pid string) {},
			entries:   []apicard.DeckCardEntry{},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, pcRepo, _ := setupDeckInteractor(t)
			pid := "p1"
			tt.grant(pcRepo, pid)

			deck, err := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
				DeckName:  tt.deckName,
				Faction:   "SHE",
				ProductID: testProductID,
				RoutineID: testRoutineID,
				SpecialID: testSpecialID,
				Cards:     tt.entries,
			})

			require.NoError(t, err)
			assert.Equal(t, tt.wantValid, deck.IsValid)
			assert.Equal(t, tt.deckName, deck.DeckName)
			assert.Equal(t, "SHE", deck.Faction)
		})
	}
}

func TestCreateDeck_ValidationErrors(t *testing.T) {
	tests := []struct {
		name       string
		faction    string
		grant      func(pcRepo *inMemoryPlayerCardRepo, pid string)
		entries    []apicard.DeckCardEntry
		wantErrMsg string
	}{
		{
			name:    "陣営が空",
			faction: "",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-001", ArtNo: 0, Count: 3})
			},
			entries:    makeEntries("C-001", 3),
			wantErrMsg: `faction "" is not selectable`,
		},
		{
			name:    "Neutral は選択不可",
			faction: "Neutral",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-007", ArtNo: 0, Count: 3})
			},
			entries:    makeEntries("C-007", 3),
			wantErrMsg: `faction "Neutral" is not selectable`,
		},
		{
			name:    "未知の陣営",
			faction: "Atlantis",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-001", ArtNo: 0, Count: 3})
			},
			entries:    makeEntries("C-001", 3),
			wantErrMsg: `faction "Atlantis" is not selectable`,
		},
		{
			name:    "31枚は上限超過",
			faction: "SHE",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantUnlimited(pcRepo, pid, allTenCards...)
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-050", ArtNo: 0, Count: 1})
			},
			entries: makeEntries(
				"C-001", 3, "C-002", 3, "C-003", 3, "C-004", 3, "C-005", 3,
				"C-006", 3, "C-007", 3, "C-008", 3, "C-009", 3, "C-010", 3,
				"C-050", 1,
			),
			wantErrMsg: "deck cannot exceed 30 cards",
		},
		{
			name:    "未所持カード",
			faction: "SHE",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-001", ArtNo: 0, Count: 3})
			},
			entries:    makeEntries("C-001", 3, "C-002", 1),
			wantErrMsg: "not enough owned",
		},
		{
			name:    "無制限カード4枚は上限3超過",
			faction: "SHE",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-001", ArtNo: 0, Count: 4})
			},
			entries:    makeEntries("C-001", 4),
			wantErrMsg: "exceeds restriction limit (4/3)",
		},
		{
			name:    "制限カード2枚は上限1超過",
			faction: "SHE",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-050", ArtNo: 0, Count: 2})
			},
			entries:    makeEntries("C-050", 2),
			wantErrMsg: "exceeds restriction limit (2/1)",
		},
		{
			name:    "準制限カード3枚は上限2超過",
			faction: "SHE",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-060", ArtNo: 0, Count: 3})
			},
			entries:    makeEntries("C-060", 3),
			wantErrMsg: "exceeds restriction limit (3/2)",
		},
		{
			name:    "枚数0は無効",
			faction: "SHE",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-001", ArtNo: 0, Count: 3})
			},
			entries:    []apicard.DeckCardEntry{{CardID: "C-001", ArtNo: 0, Count: 0}},
			wantErrMsg: "count must be positive",
		},
		{
			name:    "枚数-1は無効",
			faction: "SHE",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-001", ArtNo: 0, Count: 3})
			},
			entries:    []apicard.DeckCardEntry{{CardID: "C-001", ArtNo: 0, Count: -1}},
			wantErrMsg: "count must be positive",
		},
		{
			name:    "カード定義に存在しない",
			faction: "SHE",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-999", ArtNo: 0, Count: 3})
			},
			entries:    makeEntries("C-999", 1),
			wantErrMsg: "card C-999 not found in card definitions",
		},
		{
			name:    "アート違い合算で制限超過",
			faction: "SHE",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid,
					&domain.PlayerCard{CardID: "C-001", ArtNo: 0, Count: 3},
					&domain.PlayerCard{CardID: "C-001", ArtNo: 1, Count: 3},
				)
			},
			entries: []apicard.DeckCardEntry{
				{CardID: "C-001", ArtNo: 0, Count: 3},
				{CardID: "C-001", ArtNo: 1, Count: 1},
			},
			wantErrMsg: "exceeds restriction limit (4/3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, pcRepo, _ := setupDeckInteractor(t)
			pid := "p1"
			tt.grant(pcRepo, pid)

			_, err := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
				DeckName:  "Test",
				Faction:   tt.faction,
				ProductID: testProductID,
				RoutineID: testRoutineID,
				SpecialID: testSpecialID,
				Cards:     tt.entries,
			})

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

func factionEntries(cardIDs []string) []apicard.DeckCardEntry {
	entries := make([]apicard.DeckCardEntry, 0, len(cardIDs))
	for _, id := range cardIDs {
		entries = append(entries, apicard.DeckCardEntry{CardID: id, ArtNo: 0, Count: 1})
	}
	return entries
}

// TestCreateDeck_FactionComposition_Valid は、宣言陣営 (SHE) と Neutral のみで構成された
// デッキが陣営検証を通過することを検証します (枚数不足は IsValid=false でエラーにはならない)。
func TestCreateDeck_FactionComposition_Valid(t *testing.T) {
	tests := []struct {
		name    string
		cardIDs []string
	}{
		{"選択陣営のみ", []string{"C-001"}},
		{"選択陣営＋ニュートラル", []string{"C-001", "C-007"}},
		{"ニュートラルのみ", []string{"C-007"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, pcRepo, _ := setupDeckInteractor(t)
			pid := "p1"
			grantUnlimited(pcRepo, pid, tt.cardIDs...)

			_, err := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
				DeckName: "Test", Faction: "SHE", ProductID: testProductID,
				RoutineID: testRoutineID, SpecialID: testSpecialID,
				Cards: factionEntries(tt.cardIDs),
			})

			require.NoError(t, err)
		})
	}
}

// TestCreateDeck_FactionComposition_Invalid は、他陣営カードを含むデッキが
// 陣営検証で弾かれることを検証します。
func TestCreateDeck_FactionComposition_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		cardIDs []string
	}{
		{"他陣営のみ", []string{"C-011"}},
		{"選択陣営＋他陣営", []string{"C-001", "C-011"}},
		{"選択陣営＋他陣営＋ニュートラル", []string{"C-001", "C-011", "C-007"}},
		{"他陣営＋ニュートラル", []string{"C-011", "C-007"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, pcRepo, _ := setupDeckInteractor(t)
			pid := "p1"
			grantUnlimited(pcRepo, pid, tt.cardIDs...)

			_, err := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
				DeckName: "Test", Faction: "SHE", ProductID: testProductID,
				RoutineID: testRoutineID, SpecialID: testSpecialID,
				Cards: factionEntries(tt.cardIDs),
			})

			require.Error(t, err)
			assert.Contains(t, err.Error(), "only SHE and Neutral cards are allowed")
		})
	}
}

func TestCreateDeck_InvalidInitiatives(t *testing.T) {
	tests := []struct {
		name       string
		productID  string
		routineID  string
		specialID  string
		wantErrMsg string
	}{
		{
			name:       "不明なプロダクト",
			productID:  "PD-NOPE",
			routineID:  testRoutineID,
			specialID:  testSpecialID,
			wantErrMsg: "not found",
		},
		{
			name:       "他陣営のプロダクト",
			productID:  "PD-TST2",
			routineID:  testRoutineID,
			specialID:  testSpecialID,
			wantErrMsg: "not deck faction",
		},
		{
			name:       "不明なルーチン",
			productID:  testProductID,
			routineID:  "IN-NOPE",
			specialID:  testSpecialID,
			wantErrMsg: "is not a routine",
		},
		{
			name:       "不明なスペシャル",
			productID:  testProductID,
			routineID:  testRoutineID,
			specialID:  "IN-NOPE",
			wantErrMsg: "is not a special",
		},
		{
			name:       "ルーチンIDをスペシャルに指定",
			productID:  testProductID,
			routineID:  testRoutineID,
			specialID:  testRoutineID,
			wantErrMsg: "is not a special",
		},
		{
			name:       "スペシャルIDをルーチンに指定",
			productID:  testProductID,
			routineID:  testSpecialID,
			specialID:  testSpecialID,
			wantErrMsg: "is not a routine",
		},
		{
			name:       "別プロダクトのルーチン",
			productID:  testProductID,
			routineID:  "IN-TST-R2",
			specialID:  testSpecialID,
			wantErrMsg: "is not a routine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, pcRepo, _ := setupDeckInteractor(t)
			pid := "p1"
			grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-001", ArtNo: 0, Count: 3})

			_, err := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
				DeckName:  "Test",
				Faction:   "SHE",
				ProductID: tt.productID,
				RoutineID: tt.routineID,
				SpecialID: tt.specialID,
				Cards:     makeEntries("C-001", 3),
			})

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

func TestCreateDeck_RestrictionAtLimit(t *testing.T) {
	tests := []struct {
		name   string
		cardID string
		count  int
	}{
		{"制限カード1枚は上限ちょうど", "C-050", 1},
		{"準制限カード2枚は上限ちょうど", "C-060", 2},
		{"無制限カード3枚は上限ちょうど", "C-001", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, pcRepo, _ := setupDeckInteractor(t)
			pid := "p1"
			grantCards(pcRepo, pid, &domain.PlayerCard{CardID: tt.cardID, ArtNo: 0, Count: tt.count})

			_, err := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
				DeckName:  "OK Deck",
				Faction:   "SHE",
				ProductID: testProductID,
				RoutineID: testRoutineID,
				SpecialID: testSpecialID,
				Cards:     makeEntries(tt.cardID, tt.count),
			})

			require.NoError(t, err)
		})
	}
}

func TestUpdateDeck(t *testing.T) {
	svc, _, pcRepo, _ := setupDeckInteractor(t)
	pid := "p1"
	grantUnlimited(pcRepo, pid, allTenCards...)

	created, err := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
		DeckName: "Original", Faction: "SHE", ProductID: testProductID,
		RoutineID: testRoutineID, SpecialID: testSpecialID, Cards: makeEntries("C-001", 3, "C-002", 3),
	})
	require.NoError(t, err)
	assert.False(t, created.IsValid)

	updated, err := svc.UpdateDeck(context.Background(), pid, created.DeckID, apicard.DeckUpdateRequest{
		DeckName: "Updated Full", Faction: "SHE", ProductID: testProductID,
		RoutineID: testRoutineID, SpecialID: testSpecialID, Cards: full30Entries(),
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated Full", updated.DeckName)
	assert.True(t, updated.IsValid)
	require.NotNil(t, updated.DeckCards)
	assert.Len(t, *updated.DeckCards, 10)
}

func TestDeleteDeck(t *testing.T) {
	svc, _, pcRepo, _ := setupDeckInteractor(t)
	pid := "p1"
	grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-001", ArtNo: 0, Count: 3})

	created, _ := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
		DeckName: "To Delete", Faction: "SHE", ProductID: testProductID,
		RoutineID: testRoutineID, SpecialID: testSpecialID, Cards: makeEntries("C-001", 3),
	})

	err := svc.DeleteDeck(context.Background(), pid, created.DeckID)
	require.NoError(t, err)
	decks, _ := svc.GetDecks(context.Background(), pid)
	assert.Empty(t, decks)
}

func TestValidateDeckForBattle_Full30Cards(t *testing.T) {
	svc, _, pcRepo, _ := setupDeckInteractor(t)
	pid := "p1"
	grantUnlimited(pcRepo, pid, allTenCards...)

	deck, err := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
		DeckName: "Full Deck", Faction: "SHE", ProductID: testProductID,
		RoutineID: testRoutineID, SpecialID: testSpecialID, Cards: full30Entries(),
	})
	require.NoError(t, err)

	err = svc.ValidateDeckForBattle(context.Background(), pid, deck.DeckID)
	assert.NoError(t, err)
}

func TestValidateDeckForBattle_PartialDeck(t *testing.T) {
	svc, _, pcRepo, _ := setupDeckInteractor(t)
	pid := "p1"
	grantUnlimited(pcRepo, pid, "C-001", "C-002")

	deck, err := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
		DeckName: "Partial", Faction: "SHE", ProductID: testProductID,
		RoutineID: testRoutineID, SpecialID: testSpecialID, Cards: makeEntries("C-001", 3, "C-002", 3),
	})
	require.NoError(t, err)

	err = svc.ValidateDeckForBattle(context.Background(), pid, deck.DeckID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "need exactly 30")
}
