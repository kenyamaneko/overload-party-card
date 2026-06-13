package cache

import (
	"encoding/json"
	"fmt"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

// ProductCache はプロダクト定義のインメモリ保持です。
// 起動時に embed 済み JSON から一度ロードし、以降は読み取り専用として扱います。
type ProductCache struct {
	products []domain.Product
}

// NewProductCache は空の ProductCache を生成します。
func NewProductCache() *ProductCache {
	return &ProductCache{}
}

// LoadFromBytes は JSON バイト列 (data/cache/products_gen.json) からプロダクト定義を
// ロードします。0 件はマスター欠落としてエラーにします。
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

// Count はロード済みプロダクト数を返します。
func (c *ProductCache) Count() int {
	return len(c.products)
}
