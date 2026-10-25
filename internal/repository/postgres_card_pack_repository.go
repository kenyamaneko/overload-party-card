package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.CardPackRepository = (*PostgresCardPackRepository)(nil)

// PostgresCardPackRepository は PostgreSQL を使用した CardPackRepository の実装です。
type PostgresCardPackRepository struct {
	db dbtx
}

// NewPostgresCardPackRepository は PostgresCardPackRepository を生成します。
func NewPostgresCardPackRepository(pool *pgxpool.Pool) *PostgresCardPackRepository {
	return &PostgresCardPackRepository{db: pool}
}

// GetPack はパック ID に対応する配布パックを、内包カードをカード ID の昇順にして返します。
// パックが存在しない場合は port.ErrNotFound を返します。
func (r *PostgresCardPackRepository) GetPack(ctx context.Context, packID string) (*domain.CardPack, error) {
	rows, err := r.db.Query(ctx,
		`SELECT p.pack_id, p.description, p.is_active,
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

	var pack *domain.CardPack
	for rows.Next() {
		var (
			row    domain.CardPack
			cardID *string
			copies *int
		)
		if err := rows.Scan(&row.PackID, &row.Description, &row.IsActive, &cardID, &copies); err != nil {
			return nil, fmt.Errorf("scan card_pack row: %w", err)
		}
		if pack == nil {
			row.Cards = make([]domain.CardPackCard, 0)
			pack = &row
		}
		if cardID != nil && copies != nil {
			pack.Cards = append(pack.Cards, domain.CardPackCard{CardID: *cardID, Copies: *copies})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate card_pack rows: %w", err)
	}
	if pack == nil {
		return nil, fmt.Errorf("%w: card_pack %q", port.ErrNotFound, packID)
	}
	return pack, nil
}
