package repository

import (
	"context"
	"encoding/json"
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

// FindAll は全プロダクト定義を施策込みで返します。
func (r *PgProductRepository) FindAll(ctx context.Context) ([]*domain.Product, error) {
	products, index, err := r.queryProducts(ctx)
	if err != nil {
		return nil, err
	}
	if err := r.attachInitiatives(ctx, index); err != nil {
		return nil, err
	}
	return products, nil
}

// queryProducts はプロダクト本体を product_id 昇順で読み、ID→*Product の索引も返します。
func (r *PgProductRepository) queryProducts(ctx context.Context) ([]*domain.Product, map[string]*domain.Product, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT product_id, faction, product_name
		 FROM products ORDER BY product_id`,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	var products []*domain.Product
	index := make(map[string]*domain.Product)
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ProductID, &p.Faction, &p.ProductName); err != nil {
			return nil, nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, &p)
		index[p.ProductID] = products[len(products)-1]
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate products: %w", err)
	}
	return products, index, nil
}

// attachInitiatives は全施策を読み、親プロダクトの Initiatives に紐づけます。
func (r *PgProductRepository) attachInitiatives(ctx context.Context, index map[string]*domain.Product) error {
	rows, err := r.pool.Query(ctx,
		`SELECT initiative_id, product_id, kind, name, insight_cost, effect_text, effect
		 FROM initiatives ORDER BY product_id, initiative_id`,
	)
	if err != nil {
		return fmt.Errorf("query initiatives: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			initiative domain.Initiative
			productID  string
			effect     json.RawMessage
		)
		if err := rows.Scan(&initiative.InitiativeID, &productID, &initiative.Kind,
			&initiative.Name, &initiative.InsightCost, &initiative.EffectText, &effect); err != nil {
			return fmt.Errorf("scan initiative: %w", err)
		}
		initiative.Effect = effect
		product, ok := index[productID]
		if !ok {
			return fmt.Errorf("initiative %s references unknown product %s", initiative.InitiativeID, productID)
		}
		product.Initiatives = append(product.Initiatives, initiative)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate initiatives: %w", err)
	}
	return nil
}
