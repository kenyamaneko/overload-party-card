package cache

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.ProductCache = (*ProductCache)(nil)

// ProductCache はプロダクト定義のインメモリキャッシュです。
type ProductCache struct {
	products []domain.Product
}

// NewProductCache は空の ProductCache を生成します。
func NewProductCache() *ProductCache {
	return &ProductCache{}
}

// Load は ProductRepository から全プロダクト定義を読み込みキャッシュに格納します。
func (c *ProductCache) Load(ctx context.Context, repository port.ProductRepository) error {
	products, err := repository.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("load product cache: %w", err)
	}
	if len(products) == 0 {
		return fmt.Errorf("load product cache: %w: 0 products loaded, check products table", ErrNoProductDefinitions)
	}

	c.products = make([]domain.Product, len(products))
	for i, product := range products {
		c.products[i] = *product
	}

	slog.Info("product cache loaded", "products", len(c.products))
	return nil
}

// FindAll は全プロダクト定義のスナップショットを返します。
func (c *ProductCache) FindAll() []domain.Product {
	snapshot := make([]domain.Product, len(c.products))
	copy(snapshot, c.products)
	return snapshot
}

// FindByID は指定プロダクト ID のプロダクト定義を返します。存在しなければ nil を返します。
func (c *ProductCache) FindByID(productID string) *domain.Product {
	for i := range c.products {
		if c.products[i].ProductID == productID {
			return &c.products[i]
		}
	}
	return nil
}

// Count はロード済みプロダクト数を返します。
func (c *ProductCache) Count() int {
	return len(c.products)
}
