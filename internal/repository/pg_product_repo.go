package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.ProductRepo = (*PgProductRepository)(nil)

// PgProductRepository は PostgreSQL を使用した ProductRepo の実装です。
type PgProductRepository struct {
	pool *pgxpool.Pool
}

// NewPgProductRepository は PgProductRepository を生成します。
func NewPgProductRepository(pool *pgxpool.Pool) *PgProductRepository {
	return &PgProductRepository{pool: pool}
}

// FindAll は有効な全プロダクト定義を返します。
func (r *PgProductRepository) FindAll(ctx context.Context) ([]*domain.Product, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT product_id, faction, product_name, is_active
		 FROM products WHERE is_active = true ORDER BY product_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ProductID, &p.Faction, &p.ProductName, &p.IsActive); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}
	return products, nil
}
