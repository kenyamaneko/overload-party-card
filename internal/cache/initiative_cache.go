package cache

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.InitiativeCache = (*InitiativeCache)(nil)

// InitiativeCache は施策定義のインメモリキャッシュです。
type InitiativeCache struct {
	initiatives map[string]domain.Initiative
}

// NewInitiativeCache は空の InitiativeCache を生成します。
func NewInitiativeCache() *InitiativeCache {
	return &InitiativeCache{initiatives: make(map[string]domain.Initiative)}
}

// Load は InitiativeRepository から全施策定義を読み込みキャッシュに格納します。
func (c *InitiativeCache) Load(ctx context.Context, repository port.InitiativeRepository) error {
	initiatives, err := repository.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("load initiative cache: %w", err)
	}
	if len(initiatives) == 0 {
		return fmt.Errorf("load initiative cache: %w: 0 initiatives loaded, check initiatives table", ErrNoInitiativeDefinitions)
	}

	c.initiatives = make(map[string]domain.Initiative, len(initiatives))
	for _, initiative := range initiatives {
		c.initiatives[initiative.InitiativeID] = *initiative
	}

	slog.Info("initiative cache loaded", "initiatives", len(c.initiatives))
	return nil
}

// FindByID は指定施策 ID の施策定義を返します。存在しなければ nil を返します。
func (c *InitiativeCache) FindByID(initiativeID string) *domain.Initiative {
	initiative, ok := c.initiatives[initiativeID]
	if !ok {
		return nil
	}
	return &initiative
}

// FindAll は全施策定義を施策 ID の昇順で返します。
func (c *InitiativeCache) FindAll() []domain.Initiative {
	snapshot := make([]domain.Initiative, 0, len(c.initiatives))
	for _, initiative := range c.initiatives {
		snapshot = append(snapshot, initiative)
	}
	sort.Slice(snapshot, func(i, j int) bool {
		return snapshot[i].InitiativeID < snapshot[j].InitiativeID
	})
	return snapshot
}

// Count はロード済み施策数を返します。
func (c *InitiativeCache) Count() int {
	return len(c.initiatives)
}
