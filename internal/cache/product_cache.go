package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// ProductCache はプロダクト定義のインメモリキャッシュです。
type ProductCache struct {
	products []domain.Product
}

// NewProductCache は空の ProductCache を生成します。
func NewProductCache() *ProductCache {
	return &ProductCache{}
}

// Load は ProductRepo から全プロダクト定義を読み込みキャッシュに格納します。
func (c *ProductCache) Load(ctx context.Context, repo port.ProductRepo) error {
	products, err := repo.FindAll(ctx)
	if err != nil {
		return fmt.Errorf("load product cache: %w", err)
	}
	if len(products) == 0 {
		return fmt.Errorf("product cache: 0 products loaded, check products table or YAML seed")
	}

	c.products = make([]domain.Product, len(products))
	for i, p := range products {
		c.products[i] = *p
	}

	slog.Info("product cache loaded", "products", len(c.products))
	return nil
}

// LoadFromBytes は JSON バイト列からプロダクト定義をロードします。
func (c *ProductCache) LoadFromBytes(data []byte) error {
	var products []domain.Product
	if err := json.Unmarshal(data, &products); err != nil {
		return fmt.Errorf("parse product data: %w", err)
	}
	if len(products) == 0 {
		return fmt.Errorf("product cache: 0 products loaded, check products_gen.json")
	}
	c.products = products
	return nil
}

// All は全プロダクト定義のスナップショットを返します。
func (c *ProductCache) All() []domain.Product {
	snapshot := make([]domain.Product, len(c.products))
	copy(snapshot, c.products)
	return snapshot
}

// FindByID は指定 ID のプロダクトを返します。
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
