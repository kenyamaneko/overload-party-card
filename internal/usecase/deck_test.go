package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/constants"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
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

func setupDeckInteractor() (*DeckInteractor, *mockDeckRepo, *inMemoryPlayerCardRepo, *cache.CardCache) {
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
		{"C-007", "Compute C", "Tenki", "Compute"},
		{"C-008", "Database B", "Tenki", "Database"},
		{"C-009", "Strategy B", "Tenki", "Strategy"},
		{"C-010", "Incident B", "Tenki", "Incident"},
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

	pcRepo := newInMemoryPlayerCardRepo()
	svc := NewDeckInteractor(repo, pcRepo, cc)
	return svc, repo, pcRepo, cc
}

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
			name:     "30 cards -> valid",
			deckName: "Full Deck",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantUnlimited(pcRepo, pid, allTenCards...)
			},
			entries:   full30Entries(),
			wantValid: true,
		},
		{
			name:     "29 cards -> invalid",
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
			name:      "0 cards -> invalid",
			deckName:  "Empty Deck",
			grant:     func(pcRepo *inMemoryPlayerCardRepo, pid string) {},
			entries:   []apicard.DeckCardEntry{},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, pcRepo, _ := setupDeckInteractor()
			pid := "p1"
			tt.grant(pcRepo, pid)

			deck, err := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
				DeckName: tt.deckName,
				Cards:    tt.entries,
			})

			require.NoError(t, err)
			assert.Equal(t, tt.wantValid, deck.IsValid)
			assert.Equal(t, tt.deckName, deck.DeckName)
		})
	}
}

func TestCreateDeck_ValidationErrors(t *testing.T) {
	tests := []struct {
		name       string
		grant      func(pcRepo *inMemoryPlayerCardRepo, pid string)
		entries    []apicard.DeckCardEntry
		wantErrMsg string
	}{
		{
			name: "31 cards exceeds max",
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
			name: "unowned card",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-001", ArtNo: 0, Count: 3})
			},
			entries:    makeEntries("C-001", 3, "C-002", 1),
			wantErrMsg: "not enough owned",
		},
		{
			name: "unlimited card 4 copies (max 3)",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-001", ArtNo: 0, Count: 4})
			},
			entries:    makeEntries("C-001", 4),
			wantErrMsg: "exceeds restriction limit (4/3)",
		},
		{
			name: "limited card 2 copies (max 1)",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-050", ArtNo: 0, Count: 2})
			},
			entries:    makeEntries("C-050", 2),
			wantErrMsg: "exceeds restriction limit (2/1)",
		},
		{
			name: "semi_limited card 3 copies (max 2)",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-060", ArtNo: 0, Count: 3})
			},
			entries:    makeEntries("C-060", 3),
			wantErrMsg: "exceeds restriction limit (3/2)",
		},
		{
			name: "count = 0",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-001", ArtNo: 0, Count: 3})
			},
			entries:    []apicard.DeckCardEntry{{CardID: "C-001", ArtNo: 0, Count: 0}},
			wantErrMsg: "count must be positive",
		},
		{
			name: "count = -1",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-001", ArtNo: 0, Count: 3})
			},
			entries:    []apicard.DeckCardEntry{{CardID: "C-001", ArtNo: 0, Count: -1}},
			wantErrMsg: "count must be positive",
		},
		{
			name: "card not in definitions",
			grant: func(pcRepo *inMemoryPlayerCardRepo, pid string) {
				grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-999", ArtNo: 0, Count: 3})
			},
			entries:    makeEntries("C-999", 1),
			wantErrMsg: "card C-999 not found in card definitions",
		},
		{
			name: "cross-variant exceeds restriction",
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
			svc, _, pcRepo, _ := setupDeckInteractor()
			pid := "p1"
			tt.grant(pcRepo, pid)

			_, err := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
				DeckName: "Test",
				Cards:    tt.entries,
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
		{"limited 1 copy (max 1)", "C-050", 1},
		{"semi_limited 2 copies (max 2)", "C-060", 2},
		{"unlimited 3 copies (max 3)", "C-001", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, pcRepo, _ := setupDeckInteractor()
			pid := "p1"
			grantCards(pcRepo, pid, &domain.PlayerCard{CardID: tt.cardID, ArtNo: 0, Count: tt.count})

			_, err := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
				DeckName: "OK Deck",
				Cards:    makeEntries(tt.cardID, tt.count),
			})

			require.NoError(t, err)
		})
	}
}

func TestUpdateDeck(t *testing.T) {
	svc, _, pcRepo, _ := setupDeckInteractor()
	pid := "p1"
	grantUnlimited(pcRepo, pid, allTenCards...)

	created, err := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
		DeckName: "Original", Cards: makeEntries("C-001", 3, "C-002", 3),
	})
	require.NoError(t, err)
	assert.False(t, created.IsValid)

	updated, err := svc.UpdateDeck(context.Background(), pid, created.DeckID, apicard.DeckUpdateRequest{
		DeckName: "Updated Full", Cards: full30Entries(),
	})

	require.NoError(t, err)
	assert.Equal(t, "Updated Full", updated.DeckName)
	assert.True(t, updated.IsValid)
	assert.Len(t, updated.DeckCards, 10)
}

func TestDeleteDeck(t *testing.T) {
	svc, _, pcRepo, _ := setupDeckInteractor()
	pid := "p1"
	grantCards(pcRepo, pid, &domain.PlayerCard{CardID: "C-001", ArtNo: 0, Count: 3})

	created, _ := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
		DeckName: "To Delete", Cards: makeEntries("C-001", 3),
	})

	err := svc.DeleteDeck(context.Background(), pid, created.DeckID)
	require.NoError(t, err)
	decks, _ := svc.GetDecks(context.Background(), pid)
	assert.Empty(t, decks)
}

func TestValidateDeckForBattle_Full30Cards(t *testing.T) {
	svc, _, pcRepo, _ := setupDeckInteractor()
	pid := "p1"
	grantUnlimited(pcRepo, pid, allTenCards...)

	deck, err := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
		DeckName: "Full Deck", Cards: full30Entries(),
	})
	require.NoError(t, err)

	err = svc.ValidateDeckForBattle(context.Background(), pid, deck.DeckID)
	assert.NoError(t, err)
}

func TestValidateDeckForBattle_PartialDeck(t *testing.T) {
	svc, _, pcRepo, _ := setupDeckInteractor()
	pid := "p1"
	grantUnlimited(pcRepo, pid, "C-001", "C-002")

	deck, err := svc.CreateDeck(context.Background(), pid, apicard.DeckCreateRequest{
		DeckName: "Partial", Cards: makeEntries("C-001", 3, "C-002", 3),
	})
	require.NoError(t, err)

	err = svc.ValidateDeckForBattle(context.Background(), pid, deck.DeckID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "need exactly 30")
}

func TestRestrictionCopyCount(t *testing.T) {
	tests := []struct {
		name        string
		restriction string
		want        int
		wantErr     bool
	}{
		{"unlimited", "unlimited", 3, false},
		{"semi_limited", "semi_limited", 2, false},
		{"limited", "limited", 1, false},
		{"forbidden", "forbidden", 0, false},
		{"empty string is rejected", "", 0, true},
		{"unknown value is rejected", "weirdly_limited", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := constants.RestrictionCopyCount(tt.restriction)
			assert.Equal(t, tt.wantErr, err != nil, "err=%v", err)
			assert.Equal(t, tt.want, got)
		})
	}
}
