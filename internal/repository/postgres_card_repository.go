package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.CardRepository = (*PostgresCardRepository)(nil)

// PostgresCardRepository は PostgreSQL を使用した CardRepository の実装です。
type PostgresCardRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresCardRepository は PostgresCardRepository を生成します。
func NewPostgresCardRepository(pool *pgxpool.Pool) *PostgresCardRepository {
	return &PostgresCardRepository{pool: pool}
}

// FindAll は有効な全カード定義をカード ID の昇順で返します。
func (r *PostgresCardRepository) FindAll(ctx context.Context) ([]*domain.Card, error) {
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
	var statsDocument []byte
	err := rows.Scan(
		&c.CardID,
		&c.CardName,
		&c.ResourceLabel,
		&c.Faction,
		&c.CardType,
		&c.Subtype,
		&c.IsResizable,
		&c.IsElastic,
		&statsDocument,
		&c.EffectText,
		&c.Effects,
		&c.Restriction,
		&c.IsActive,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.Stats, err = decodeCardStats(statsDocument)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// cardStatsDocument は保存されたステータスの JSON の形式です。
type cardStatsDocument struct {
	Availability    *int64 `json:"availability"`
	MaintenanceCost *int64 `json:"maintenance_cost"`
	SLAPenalty      *int64 `json:"sla_penalty"`
	Throughput      *int64 `json:"throughput"`
	ThroughputMax   *int64 `json:"throughput_max"`
	Yield           *int64 `json:"yield"`
	YieldMax        *int64 `json:"yield_max"`
}

// decodeCardStats は保存されたステータスを解釈します。未知の項目を含むときはエラーにします。
func decodeCardStats(document []byte) (domain.CardStats, error) {
	var stats cardStatsDocument
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&stats); err != nil {
		return domain.CardStats{}, fmt.Errorf("decode card stats: %w", err)
	}
	return domain.CardStats(stats), nil
}
