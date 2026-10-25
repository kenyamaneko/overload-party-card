package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.ProductRepository = (*PostgresProductRepository)(nil)

// PostgresProductRepository は PostgreSQL を使用した ProductRepository の実装です。
type PostgresProductRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresProductRepository は PostgresProductRepository を生成します。
func NewPostgresProductRepository(pool *pgxpool.Pool) *PostgresProductRepository {
	return &PostgresProductRepository{pool: pool}
}

// FindAll は有効な全プロダクトをプロダクト ID の昇順で返します。
func (r *PostgresProductRepository) FindAll(ctx context.Context) ([]*domain.Product, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT product_id, faction, product_name, is_active
		 FROM card.products WHERE is_active = true ORDER BY product_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	products := make([]*domain.Product, 0)
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
