package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.CardPackRepo = (*PgCardPackRepository)(nil)

// PgCardPackRepository は PostgreSQL を使用した CardPackRepo の実装です。
type PgCardPackRepository struct {
	pool *pgxpool.Pool
}

// NewPgCardPackRepository は PgCardPackRepository を生成します。
func NewPgCardPackRepository(pool *pgxpool.Pool) *PgCardPackRepository {
	return &PgCardPackRepository{pool: pool}
}

// GetPack は pack_id に対応するパック定義と内包カードを返します。
// 行が存在しない場合は port.ErrNotFound を返します。
func (r *PgCardPackRepository) GetPack(ctx context.Context, packID string) (*domain.CardPack, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT p.pack_id, p.description, p.is_active, p.created_at, p.updated_at,
		        c.card_id, c.copies
		 FROM card.card_pack p
		 LEFT JOIN card.card_pack_cards c ON c.pack_id = p.pack_id
		 WHERE p.pack_id = $1
		 ORDER BY c.card_id`,
		packID,
	)
	if err != nil {
		return nil, fmt.Errorf("query card_pack: %w", err)
	}
	defer rows.Close()

	var p *domain.CardPack
	for rows.Next() {
		var (
			row    domain.CardPack
			cardID *string
			copies *int
		)
		if err := rows.Scan(&row.PackID, &row.Description, &row.IsActive,
			&row.CreatedAt, &row.UpdatedAt, &cardID, &copies); err != nil {
			return nil, fmt.Errorf("scan card_pack row: %w", err)
		}
		if p == nil {
			row.Cards = make([]domain.CardPackCard, 0)
			p = &row
		}
		if cardID != nil && copies != nil {
			p.Cards = append(p.Cards, domain.CardPackCard{CardID: *cardID, Copies: *copies})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate card_pack rows: %w", err)
	}
	if p == nil {
		return nil, fmt.Errorf("%w: card_pack %q", port.ErrNotFound, packID)
	}
	return p, nil
}

