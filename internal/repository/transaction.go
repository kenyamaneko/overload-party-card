package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// runInTransaction は fn を 1 つのトランザクションで実行します。
// fn が成功したらコミットし、失敗したらロールバックします。ロールバックの失敗は fn のエラーと合わせて返します。
func runInTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	isCommitAttempted := false
	defer func() {
		if isCommitAttempted {
			return
		}
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("rollback tx: %w", rollbackErr))
		}
	}()

	if err := fn(tx); err != nil {
		return err
	}
	isCommitAttempted = true
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
