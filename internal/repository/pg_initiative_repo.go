package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.InitiativeRepo = (*PgInitiativeRepository)(nil)

// PgInitiativeRepository は PostgreSQL を使用した InitiativeRepo の実装です。
type PgInitiativeRepository struct {
	pool *pgxpool.Pool
}

// NewPgInitiativeRepository は PgInitiativeRepository を生成します。
func NewPgInitiativeRepository(pool *pgxpool.Pool) *PgInitiativeRepository {
	return &PgInitiativeRepository{pool: pool}
}

// FindAll は全施策定義を返します。
func (r *PgInitiativeRepository) FindAll(ctx context.Context) ([]*domain.Initiative, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT initiative_id, product_id, kind, name, insight_cost, effect_text, effect
		 FROM initiatives ORDER BY initiative_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("query initiatives: %w", err)
	}
	defer rows.Close()

	var initiatives []*domain.Initiative
	for rows.Next() {
		var (
			i      domain.Initiative
			effect json.RawMessage
		)
		if err := rows.Scan(&i.InitiativeID, &i.ProductID, &i.Kind, &i.Name,
			&i.InsightCost, &i.EffectText, &effect); err != nil {
			return nil, fmt.Errorf("scan initiative: %w", err)
		}
		i.Effect = effect
		initiatives = append(initiatives, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate initiatives: %w", err)
	}
	return initiatives, nil
}
