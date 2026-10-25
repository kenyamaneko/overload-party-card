package cache

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.CardCache = (*CardCache)(nil)

// CardCache はカード定義のインメモリキャッシュです。
type CardCache struct {
	mu    sync.RWMutex
	cards map[string]*domain.Card
}

// NewCardCache は空の CardCache を生成します。
func NewCardCache() *CardCache {
	return &CardCache{cards: make(map[string]*domain.Card)}
}

// FindByID は指定カード ID のカード定義を返します。存在しなければ nil を返します。
func (c *CardCache) FindByID(cardID string) *domain.Card {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cards[cardID]
}

// Load は CardRepository から全カード定義を読み込みキャッシュに格納します。
func (c *CardCache) Load(ctx context.Context, repository port.CardRepository) error {
	cards, err := repository.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("load card cache: %w", err)
	}

	if len(cards) == 0 {
		return fmt.Errorf("load card cache: %w: 0 cards loaded, check card_definitions table", ErrNoCardDefinitions)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cards = make(map[string]*domain.Card, len(cards))
	for _, card := range cards {
		c.cards[card.CardID] = card
	}

	slog.Info("card cache loaded", "cards", len(c.cards))
	return nil
}

// Count はキャッシュ内のカード数を返します。
func (c *CardCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cards)
}
