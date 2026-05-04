package usecase

import (
	"context"
	"sort"
	"sync"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

// inMemoryCardRepo は port.CardRepo のメモリバック実装。
// usecase 層を仕様ベースで検証する各テストで共有する。
type inMemoryCardRepo struct {
	cards map[string]*domain.Card
}

func newInMemoryCardRepo(cards map[string]*domain.Card) *inMemoryCardRepo {
	return &inMemoryCardRepo{cards: cards}
}

func (r *inMemoryCardRepo) FindAll(_ context.Context) ([]*domain.Card, error) {
	result := make([]*domain.Card, 0, len(r.cards))
	for _, c := range r.cards {
		result = append(result, c)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CardID < result[j].CardID })
	return result, nil
}

func (r *inMemoryCardRepo) FindCardIDsByFactions(_ context.Context, factions []string) ([]string, error) {
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

// inMemoryPlayerCardRepo は port.PlayerCardRepo のメモリバック実装。
// 並行アクセスに耐えるため mutex で保護し、Seed で事前データ投入を許す。
type inMemoryPlayerCardRepo struct {
	mu          sync.Mutex
	playerCards map[string][]*domain.PlayerCard
}

func newInMemoryPlayerCardRepo() *inMemoryPlayerCardRepo {
	return &inMemoryPlayerCardRepo{playerCards: make(map[string][]*domain.PlayerCard)}
}

func (r *inMemoryPlayerCardRepo) GetPlayerCards(_ context.Context, playerID string) ([]*domain.PlayerCard, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.playerCards[playerID], nil
}

func (r *inMemoryPlayerCardRepo) AddCards(_ context.Context, playerID string, cardIDs []string, countPerCard int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing := make(map[string]*domain.PlayerCard, len(r.playerCards[playerID]))
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
		pc := &domain.PlayerCard{PlayerID: playerID, CardID: id, ArtNo: 0, Count: countPerCard}
		r.playerCards[playerID] = append(r.playerCards[playerID], pc)
		existing[id] = pc
	}
	return len(cardIDs) * countPerCard, nil
}

// Seed は事前データを投入するヘルパ。テストの初期状態を構築するために使う。
func (r *inMemoryPlayerCardRepo) Seed(playerID string, cards []*domain.PlayerCard) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.playerCards[playerID] = append(r.playerCards[playerID], cards...)
}
