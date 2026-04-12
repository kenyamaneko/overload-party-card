package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// CardCache はカード定義のインメモリキャッシュです。
type CardCache struct {
	mu    sync.RWMutex
	cards map[string]*apicard.CardDefinition
}

// NewCardCache は空の CardCache を生成します。
func NewCardCache() *CardCache {
	return &CardCache{cards: make(map[string]*apicard.CardDefinition)}
}

// Get は指定 cardID のカード定義を返します。存在しなければ nil を返します。
func (c *CardCache) Get(cardID string) *apicard.CardDefinition {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cards[cardID]
}

// Load は CardRepo から全カード定義を読み込みキャッシュに格納します。
func (c *CardCache) Load(ctx context.Context, repo port.CardRepo) error {
	cards, err := repo.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("load card cache: %w", err)
	}

	if len(cards) == 0 {
		return fmt.Errorf("cache.Load: 0 cards loaded, check card_definitions table or YAML seed")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cards = make(map[string]*apicard.CardDefinition, len(cards))
	for _, card := range cards {
		c.cards[card.CardID] = card
	}

	log.Printf("card cache loaded: %d cards", len(c.cards))
	return nil
}

// All はキャッシュ内の全カード定義のスナップショットを返します。
func (c *CardCache) All() map[string]*apicard.CardDefinition {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := make(map[string]*apicard.CardDefinition, len(c.cards))
	for k, v := range c.cards {
		snapshot[k] = v
	}
	return snapshot
}

// Count はキャッシュ内のカード数を返します。
func (c *CardCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cards)
}

// LoadFromBytes は JSON バイト列からキャッシュを構築します。
// テストで DB を使わずに data/cache/cards_gen.json からシードするために使用します。
func (c *CardCache) LoadFromBytes(data []byte) error {
	var cards []*apicard.CardDefinition
	if err := json.Unmarshal(data, &cards); err != nil {
		return fmt.Errorf("parse card data: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cards = make(map[string]*apicard.CardDefinition, len(cards))
	for _, card := range cards {
		c.cards[card.CardID] = card
	}
	return nil
}

// InjectForTest はテスト用にカード定義を直接キャッシュに挿入します。
func (c *CardCache) InjectForTest(cardID string, card *apicard.CardDefinition) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cards[cardID] = card
}
