package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.CardRepo = (*PgCardRepository)(nil)

// PgCardRepository は PostgreSQL を使用した CardRepo の実装です。
type PgCardRepository struct {
	pool *pgxpool.Pool
}

// NewPgCardRepository は PgCardRepository を生成します。
func NewPgCardRepository(pool *pgxpool.Pool) *PgCardRepository {
	return &PgCardRepository{pool: pool}
}

// FindAll は有効な全カード定義を返します。
func (r *PgCardRepository) FindAll(ctx context.Context) ([]*domain.Card, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT card_id, card_name, resource_label, faction, card_type, subtype, resizable, elastic, stats, effect_text, effects, restriction, is_active, created_at, updated_at
		 FROM card.card_definitions WHERE is_active = true ORDER BY card_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("query cards: %w", err)
	}
	defer rows.Close()

	cards := make([]*domain.Card, 0)
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, fmt.Errorf("scan card: %w", err)
		}
		cards = append(cards, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cards: %w", err)
	}
	return cards, nil
}

func scanCard(rows pgx.Rows) (*domain.Card, error) {
	var c domain.Card
	var stats, effects json.RawMessage
	err := rows.Scan(
		&c.CardID,
		&c.CardName,
		&c.ResourceLabel,
		&c.Faction,
		&c.CardType,
		&c.Subtype,
		&c.Resizable,
		&c.Elastic,
		&stats,
		&c.EffectText,
		&effects,
		&c.Restriction,
		&c.IsActive,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.Stats = stats
	c.Effects = effects
	return &c, nil
}
