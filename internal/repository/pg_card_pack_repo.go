package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
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

// GetPack は pack_id に対応するパック定義を返します。
// 行が存在しない場合 port.ErrNotFound、selection JSONB が壊れている場合は
// port.ErrInvalidPackSelection で wrap して返します。
func (r *PgCardPackRepository) GetPack(ctx context.Context, packID string) (*domain.CardPack, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT pack_id, description, selection, copies_per_card, is_active, created_at, updated_at
		 FROM card_pack WHERE pack_id = $1`,
		packID,
	)
	var p domain.CardPack
	var selRaw json.RawMessage
	err := row.Scan(&p.PackID, &p.Description, &selRaw, &p.CopiesPerCard, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: card_pack %q", port.ErrNotFound, packID)
		}
		return nil, fmt.Errorf("query card_pack: %w", err)
	}
	sel, err := domain.ParseSelection(selRaw)
	if err != nil {
		return nil, fmt.Errorf("%w: pack %q: %v", port.ErrInvalidPackSelection, packID, err)
	}
	p.Selection = sel
	return &p, nil
}
