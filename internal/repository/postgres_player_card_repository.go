package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.PlayerCardRepository = (*PostgresPlayerCardRepository)(nil)

// PostgresPlayerCardRepository は PostgreSQL を使用した PlayerCardRepository の実装です。
type PostgresPlayerCardRepository struct {
	db dbtx
}

// NewPostgresPlayerCardRepository は PostgresPlayerCardRepository を生成します。
func NewPostgresPlayerCardRepository(pool *pgxpool.Pool) *PostgresPlayerCardRepository {
	return &PostgresPlayerCardRepository{db: pool}
}

// GetPlayerCards は指定プレイヤーの所持カードをカード ID・アート番号の昇順で返します。
func (r *PostgresPlayerCardRepository) GetPlayerCards(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
	rows, err := r.db.Query(ctx,
		`SELECT player_id, card_id, art_no, count
		 FROM card.player_cards WHERE player_id = $1 ORDER BY card_id, art_no`,
		playerID,
	)
	if err != nil {
		return nil, fmt.Errorf("query player cards: %w", err)
	}
	defer rows.Close()

	cards := make([]*domain.PlayerCard, 0)
	for rows.Next() {
		var playerCard domain.PlayerCard
		if err := rows.Scan(&playerCard.PlayerID, &playerCard.CardID, &playerCard.ArtNo, &playerCard.Count); err != nil {
			return nil, fmt.Errorf("scan player card: %w", err)
		}
		cards = append(cards, &playerCard)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate player cards: %w", err)
	}
	return cards, nil
}

// AddCards は指定の所持カードを、プレイヤー・カード・アート番号ごとに所持枚数へ加算します。
// 所持していないカードは新たに所持カードに加えます。
func (r *PostgresPlayerCardRepository) AddCards(ctx context.Context, cards []domain.PlayerCard) error {
	rows := make([][]any, len(cards))
	for i, card := range cards {
		rows[i] = []any{card.PlayerID, card.CardID, card.ArtNo, card.Count}
	}

	err := insertRows(ctx, r.db,
		"card.player_cards",
		[]string{"player_id", "card_id", "art_no", "count"},
		rows,
		"ON CONFLICT (player_id, card_id, art_no) DO UPDATE SET count = player_cards.count + EXCLUDED.count",
	)
	if err != nil {
		return fmt.Errorf("bulk upsert player_cards: %w", err)
	}
	return nil
}
