package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
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
func (r *PgPlayerCardRepository) GetPlayerCards(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT player_id, card_id, art_no, count
		 FROM card.player_cards WHERE player_id = $1 ORDER BY card_id, art_no`,
		playerID,
	)
	if err != nil {
		return nil, fmt.Errorf("query player cards: %w", err)
	}
	defer rows.Close()

	cards := make([]*domain.PlayerCard, 0, 32)
	for rows.Next() {
		var pc domain.PlayerCard
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
// 各 CardPackCard.Copies が個別に加算されるため、カードごとに異なる枚数を一括で加算できます。
// 単一文の bulk UPSERT で実行し、加算したコピー総数を返します。
func (r *PgPlayerCardRepository) AddCards(ctx context.Context, playerID string, cards []domain.CardPackCard) (int, error) {
	if len(cards) == 0 {
		return 0, nil
	}

	var sb strings.Builder
	sb.WriteString(`INSERT INTO card.player_cards (player_id, card_id, art_no, count) VALUES `)

	args := make([]any, 0, len(cards)*4)
	total := 0
	for i, c := range cards {
		if i > 0 {
			sb.WriteString(",")
		}
		base := i*4 + 1
		fmt.Fprintf(&sb, "($%d,$%d,$%d,$%d)", base, base+1, base+2, base+3)
		args = append(args, playerID, c.CardID, 0, c.Copies)
		total += c.Copies
	}
	sb.WriteString(` ON CONFLICT (player_id, card_id, art_no) DO UPDATE SET count = player_cards.count + EXCLUDED.count`)

	if _, err := r.pool.Exec(ctx, sb.String(), args...); err != nil {
		return 0, fmt.Errorf("bulk upsert player_cards: %w", err)
	}
	return total, nil
}
