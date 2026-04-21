package service

import (
	"context"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	"github.com/kenyamaneko/overload-party-card/internal/model"
)

type mockCardRepo struct {
	cards map[string]*apicard.CardDefinition
}

func newMockCardRepo(cards map[string]*apicard.CardDefinition) *mockCardRepo {
	return &mockCardRepo{cards: cards}
}

func (r *mockCardRepo) FindAll(_ context.Context) ([]*apicard.CardDefinition, error) {
	result := make([]*apicard.CardDefinition, 0, len(r.cards))
	for _, c := range r.cards {
		result = append(result, c)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CardID < result[j].CardID })
	return result, nil
}

func (r *mockCardRepo) FindCardIDsByFactions(_ context.Context, factions []string) ([]string, error) {
	set := make(map[string]struct{}, len(factions))
	for _, f := range factions {
		set[f] = struct{}{}
	}
	ids := make([]string, 0, len(r.cards))
	for _, c := range r.cards {
		_, ok := set[c.Faction]
		if ok && c.IsActive {
			ids = append(ids, c.CardID)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

type mockPlayerCardRepoShared struct {
	mu          sync.Mutex
	playerCards map[string][]*model.PlayerCard
}

func newMockPlayerCardRepoShared() *mockPlayerCardRepoShared {
	return &mockPlayerCardRepoShared{playerCards: make(map[string][]*model.PlayerCard)}
}

func (r *mockPlayerCardRepoShared) GetPlayerCards(_ context.Context, playerID string) ([]*model.PlayerCard, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.playerCards[playerID], nil
}

func (r *mockPlayerCardRepoShared) AddCards(_ context.Context, playerID string, cardIDs []string, countPerCard int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing := make(map[string]*model.PlayerCard, len(r.playerCards[playerID]))
	for _, pc := range r.playerCards[playerID] {
		if pc.ArtNo == 0 {
			existing[pc.CardID] = pc
		}
	}
	for _, id := range cardIDs {
		if pc, ok := existing[id]; ok {
			pc.Count += countPerCard
			continue
		}
		pc := &model.PlayerCard{PlayerID: playerID, CardID: id, ArtNo: 0, Count: countPerCard}
		r.playerCards[playerID] = append(r.playerCards[playerID], pc)
		existing[id] = pc
	}
	return len(cardIDs) * countPerCard, nil
}

func (r *mockPlayerCardRepoShared) Seed(playerID string, cards []*model.PlayerCard) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.playerCards[playerID] = append(r.playerCards[playerID], cards...)
}

type ownershipExpectation struct {
	cardID  string
	isOwned bool
}

func TestGetAllCards(t *testing.T) {
	tests := []struct {
		name  string
		cards map[string]*apicard.CardDefinition
		seed  []*model.PlayerCard
		want  []ownershipExpectation
	}{
		{
			name: "some cards owned",
			cards: map[string]*apicard.CardDefinition{
				"C-001": {CardID: "C-001", CardName: "Fireball"},
				"C-002": {CardID: "C-002", CardName: "Shield"},
				"C-003": {CardID: "C-003", CardName: "Heal"},
			},
			seed: []*model.PlayerCard{
				{PlayerID: "p1", CardID: "C-001", ArtNo: 1, Count: 1},
				{PlayerID: "p1", CardID: "C-003", ArtNo: 1, Count: 2},
			},
			want: []ownershipExpectation{
				{"C-001", true},
				{"C-002", false},
				{"C-003", true},
			},
		},
		{
			name:  "no cards in master",
			cards: map[string]*apicard.CardDefinition{},
			seed:  nil,
			want:  nil,
		},
		{
			name: "no owned cards",
			cards: map[string]*apicard.CardDefinition{
				"C-001": {CardID: "C-001", CardName: "Fireball"},
				"C-002": {CardID: "C-002", CardName: "Shield"},
			},
			seed: nil,
			want: []ownershipExpectation{
				{"C-001", false},
				{"C-002", false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pcRepo := newMockPlayerCardRepoShared()
			pcRepo.Seed("p1", tt.seed)
			svc := NewCardService(newMockCardRepo(tt.cards), pcRepo)

			got, err := svc.GetAllCards(context.Background(), "p1")
			require.NoError(t, err)
			require.Len(t, got, len(tt.want))
			for i, w := range tt.want {
				assert.Equal(t, w.cardID, got[i].CardID)
				assert.Equal(t, w.isOwned, got[i].IsOwned)
			}
		})
	}
}

func TestFindAllRaw(t *testing.T) {
	cards := map[string]*apicard.CardDefinition{
		"C-001": {CardID: "C-001", CardName: "Fireball"},
		"C-002": {CardID: "C-002", CardName: "Shield"},
	}
	svc := NewCardService(newMockCardRepo(cards), newMockPlayerCardRepoShared())

	result, err := svc.FindAllRaw(context.Background())
	require.NoError(t, err)
	require.Len(t, result, 2)
}
