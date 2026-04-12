package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	"github.com/kenyamaneko/overload-party-card/internal/model"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

type mockCardRepo struct {
	cards map[string]*apicard.CardDefinition
}

func newMockCardRepo(cards map[string]*apicard.CardDefinition) *mockCardRepo {
	return &mockCardRepo{cards: cards}
}

func (r *mockCardRepo) FindAll(_ context.Context) ([]*apicard.CardDefinition, error) {
	var result []*apicard.CardDefinition
	for _, c := range r.cards {
		result = append(result, c)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CardID < result[j].CardID })
	return result, nil
}

func (r *mockCardRepo) FindByCardID(_ context.Context, cardID string) (*apicard.CardDefinition, error) {
	c, ok := r.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("card %s: %w", cardID, port.ErrNotFound)
	}
	return c, nil
}

func (r *mockCardRepo) FindCardIDsByFactions(_ context.Context, factions []string) ([]string, error) {
	set := make(map[string]struct{}, len(factions))
	for _, f := range factions {
		set[f] = struct{}{}
	}
	var ids []string
	for _, c := range r.cards {
		if _, ok := set[c.Faction]; ok && c.IsActive {
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

func TestGetAllCards_WithOwnership(t *testing.T) {
	cards := map[string]*apicard.CardDefinition{
		"C-001": {CardID: "C-001", CardName: "Fireball"},
		"C-002": {CardID: "C-002", CardName: "Shield"},
		"C-003": {CardID: "C-003", CardName: "Heal"},
	}
	cardRepo := newMockCardRepo(cards)
	pcRepo := newMockPlayerCardRepoShared()
	pcRepo.Seed("player1", []*model.PlayerCard{
		{PlayerID: "player1", CardID: "C-001", ArtNo: 1, Count: 1},
		{PlayerID: "player1", CardID: "C-003", ArtNo: 1, Count: 2},
	})

	svc := NewCardService(cardRepo, pcRepo)

	result, err := svc.GetAllCards(context.Background(), "player1")
	require.NoError(t, err)
	require.Len(t, result, 3)

	assert.Equal(t, "C-001", result[0].CardID)
	assert.True(t, result[0].IsOwned)

	assert.Equal(t, "C-002", result[1].CardID)
	assert.False(t, result[1].IsOwned)

	assert.Equal(t, "C-003", result[2].CardID)
	assert.True(t, result[2].IsOwned)
}

func TestGetAllCards_NoCards(t *testing.T) {
	svc := NewCardService(newMockCardRepo(map[string]*apicard.CardDefinition{}), newMockPlayerCardRepoShared())

	result, err := svc.GetAllCards(context.Background(), "player1")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetAllCards_NoOwnedCards(t *testing.T) {
	cards := map[string]*apicard.CardDefinition{
		"C-001": {CardID: "C-001", CardName: "Fireball"},
		"C-002": {CardID: "C-002", CardName: "Shield"},
	}
	svc := NewCardService(newMockCardRepo(cards), newMockPlayerCardRepoShared())

	result, err := svc.GetAllCards(context.Background(), "player1")
	require.NoError(t, err)
	require.Len(t, result, 2)

	for _, c := range result {
		assert.False(t, c.IsOwned, "card %s should not be owned", c.CardID)
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
