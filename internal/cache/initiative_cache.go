package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// InitiativeCache は施策定義のインメモリキャッシュです。
type InitiativeCache struct {
	initiatives map[string]domain.Initiative
}

// NewInitiativeCache は空の InitiativeCache を生成します。
func NewInitiativeCache() *InitiativeCache {
	return &InitiativeCache{initiatives: make(map[string]domain.Initiative)}
}

// Load は InitiativeRepo から全施策定義を読み込みキャッシュに格納します。
func (c *InitiativeCache) Load(ctx context.Context, repo port.InitiativeRepo) error {
	initiatives, err := repo.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("load initiative cache: %w", err)
	}
	if len(initiatives) == 0 {
		return fmt.Errorf("initiative cache: 0 initiatives loaded, check initiatives table or YAML seed")
	}

	c.initiatives = make(map[string]domain.Initiative, len(initiatives))
	for _, i := range initiatives {
		c.initiatives[i.InitiativeID] = *i
	}

	slog.Info("initiative cache loaded", "initiatives", len(c.initiatives))
	return nil
}

// LoadFromBytes は JSON バイト列から施策定義をロードします。
func (c *InitiativeCache) LoadFromBytes(data []byte) error {
	var initiatives []domain.Initiative
	if err := json.Unmarshal(data, &initiatives); err != nil {
		return fmt.Errorf("parse initiative data: %w", err)
	}
	if len(initiatives) == 0 {
		return fmt.Errorf("initiative cache: 0 initiatives loaded, check initiatives_gen.json")
	}

	c.initiatives = make(map[string]domain.Initiative, len(initiatives))
	for _, i := range initiatives {
		c.initiatives[i.InitiativeID] = i
	}
	return nil
}

// FindByID は指定 ID の施策を返します。存在しなければ nil を返します。
func (c *InitiativeCache) FindByID(initiativeID string) *domain.Initiative {
	i, ok := c.initiatives[initiativeID]
	if !ok {
		return nil
	}
	return &i
}

// All は全施策定義を initiative_id 昇順で返します。
func (c *InitiativeCache) All() []domain.Initiative {
	snapshot := make([]domain.Initiative, 0, len(c.initiatives))
	for _, i := range c.initiatives {
		snapshot = append(snapshot, i)
	}
	sort.Slice(snapshot, func(a, b int) bool {
		return snapshot[a].InitiativeID < snapshot[b].InitiativeID
	})
	return snapshot
}

// Count はロード済み施策数を返します。
func (c *InitiativeCache) Count() int {
	return len(c.initiatives)
}
