package cache

import (
	"encoding/json"
	"fmt"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

// ProductCache はプロダクト定義のインメモリキャッシュです。
type ProductCache struct {
	products []domain.Product
}

// NewProductCache は空の ProductCache を生成します。
func NewProductCache() *ProductCache {
	return &ProductCache{}
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
