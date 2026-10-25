package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.InitiativeRepository = (*PostgresInitiativeRepository)(nil)

// PostgresInitiativeRepository は PostgreSQL を使用した InitiativeRepository の実装です。
type PostgresInitiativeRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresInitiativeRepository は PostgresInitiativeRepository を生成します。
func NewPostgresInitiativeRepository(pool *pgxpool.Pool) *PostgresInitiativeRepository {
	return &PostgresInitiativeRepository{pool: pool}
}

// FindAll は有効な全施策を施策 ID の昇順で返します。
func (r *PostgresInitiativeRepository) FindAll(ctx context.Context) ([]*domain.Initiative, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT initiative_id, product_id, kind, name, insight_cost, effect_text, effect, is_active
		 FROM card.initiatives WHERE is_active = true ORDER BY initiative_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("query initiatives: %w", err)
	}
	defer rows.Close()

	initiatives := make([]*domain.Initiative, 0)
	for rows.Next() {
		var initiative domain.Initiative
		if err := rows.Scan(&initiative.InitiativeID, &initiative.ProductID, &initiative.Kind, &initiative.Name,
			&initiative.InsightCost, &initiative.EffectText, &initiative.Effect, &initiative.IsActive); err != nil {
			return nil, fmt.Errorf("scan initiative: %w", err)
		}
		initiatives = append(initiatives, &initiative)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate initiatives: %w", err)
	}
	return initiatives, nil
}
