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

// 永続化形式 (JSONB) の selection.type 値。domain には流出させない。
const (
	selectionTypeByFactions = "by_factions"
	selectionTypeByCardIDs  = "by_card_ids"
)

// GetPack は pack_id に対応するパック定義を返します。
// 行が存在しない場合は port.ErrNotFound、selection JSONB が不正な場合は
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
	sel, err := parseSelection(selRaw)
	if err != nil {
		return nil, fmt.Errorf("%w: pack %q: %v", port.ErrInvalidPackSelection, packID, err)
	}
	p.Selection = sel
	return &p, nil
}

// parseSelection は selection JSONB を domain.Selection 実装に変換します。
// 永続化形式の type discriminator を解釈する責務は repository に閉じる。
func parseSelection(raw json.RawMessage) (domain.Selection, error) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, fmt.Errorf("parse selection head: %w", err)
	}
	switch head.Type {
	case selectionTypeByFactions:
		var body struct {
			Factions []string `json:"factions"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, fmt.Errorf("parse selection by_factions: %w", err)
		}
		if len(body.Factions) == 0 {
			return nil, fmt.Errorf("selection by_factions: empty factions list")
		}
		return domain.SelectionByFactions{Factions: body.Factions}, nil
	case selectionTypeByCardIDs:
		var body struct {
			CardIDs []string `json:"card_ids"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, fmt.Errorf("parse selection by_card_ids: %w", err)
		}
		if len(body.CardIDs) == 0 {
			return nil, fmt.Errorf("selection by_card_ids: empty card_ids list")
		}
		return domain.SelectionByCardIDs{CardIDs: body.CardIDs}, nil
	}
	return nil, fmt.Errorf("unknown selection type %q", head.Type)
}
