package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/model"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.PlayerCardRepo = (*PgPlayerCardRepository)(nil)

// PgPlayerCardRepository は PostgreSQL を使用した PlayerCardRepo の実装です。
type PgPlayerCardRepository struct {
	pool *pgxpool.Pool
}

// NewPgPlayerCardRepository は PgPlayerCardRepository を生成します。
func NewPgPlayerCardRepository(pool *pgxpool.Pool) *PgPlayerCardRepository {
	return &PgPlayerCardRepository{pool: pool}
}

// GetPlayerCards は指定プレイヤーの所持カード一覧を返します。
func (r *PgPlayerCardRepository) GetPlayerCards(ctx context.Context, playerID string) ([]*model.PlayerCard, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT player_id, card_id, art_no, count
		 FROM player_cards WHERE player_id = $1 ORDER BY card_id, art_no`,
		playerID,
	)
	if err != nil {
		return nil, fmt.Errorf("query player cards: %w", err)
	}
	defer rows.Close()

	cards := make([]*model.PlayerCard, 0, 32)
	for rows.Next() {
		var pc model.PlayerCard
		if err := rows.Scan(&pc.PlayerID, &pc.CardID, &pc.ArtNo, &pc.Count); err != nil {
			return nil, fmt.Errorf("scan player card: %w", err)
		}
		cards = append(cards, &pc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate player cards: %w", err)
	}
	return cards, nil
}

// AddCards は player_cards を UPSERT し、conflict 時は count を加算します。
// 単一トランザクションで実行し、追加されたコピー総数を返します。
func (r *PgPlayerCardRepository) AddCards(ctx context.Context, playerID string, cardIDs []string, countPerCard int) (int, error) {
	if countPerCard <= 0 {
		return 0, fmt.Errorf("countPerCard must be positive, got %d", countPerCard)
	}
	if len(cardIDs) == 0 {
		return 0, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO player_cards (player_id, card_id, art_no, count)
		VALUES ($1, $2, 0, $3)
		ON CONFLICT (player_id, card_id, art_no)
		DO UPDATE SET count = player_cards.count + EXCLUDED.count
	`
	for _, cid := range cardIDs {
		if _, err := tx.Exec(ctx, q, playerID, cid, countPerCard); err != nil {
			return 0, fmt.Errorf("upsert player_card %s: %w", cid, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}
	return len(cardIDs) * countPerCard, nil
}
