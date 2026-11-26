package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.ProcessedEventRepository = (*PostgresProcessedEventRepository)(nil)

// PostgresProcessedEventRepository は PostgreSQL を使用した ProcessedEventRepository の実装です。
type PostgresProcessedEventRepository struct {
	db dbtx
}

// NewPostgresProcessedEventRepository は PostgresProcessedEventRepository を生成します。
func NewPostgresProcessedEventRepository(pool *pgxpool.Pool) *PostgresProcessedEventRepository {
	return &PostgresProcessedEventRepository{db: pool}
}

// Insert はイベントを処理済みとして記録します。
// 初めて記録したときは isInserted が true、記録済みのイベントのときは false になります。
func (r *PostgresProcessedEventRepository) Insert(ctx context.Context, eventID, eventType string) (isInserted bool, err error) {
	var insertedEventID string
	err = r.db.QueryRow(ctx,
		`INSERT INTO card.processed_events (event_id, event_type)
		 VALUES ($1, $2)
		 ON CONFLICT (event_id) DO NOTHING
		 RETURNING event_id`,
		eventID, eventType,
	).Scan(&insertedEventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("insert card.processed_events: %w", err)
	}
	return true, nil
}
