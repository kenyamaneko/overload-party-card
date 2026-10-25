package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.GrantTransactionRunner = (*PostgresGrantTransactionRunner)(nil)

// PostgresGrantTransactionRunner は PostgreSQL のトランザクションを使用した GrantTransactionRunner の実装です。
type PostgresGrantTransactionRunner struct {
	pool *pgxpool.Pool
}

// NewPostgresGrantTransactionRunner は PostgresGrantTransactionRunner を生成します。
func NewPostgresGrantTransactionRunner(pool *pgxpool.Pool) *PostgresGrantTransactionRunner {
	return &PostgresGrantTransactionRunner{pool: pool}
}

// RunInTransaction は fn を 1 つのトランザクションで実行し、fn に渡したリポジトリの変更をまとめて確定します。
// fn がエラーを返したときは、リポジトリの変更をすべて取り消してそのエラーを返します。
func (r *PostgresGrantTransactionRunner) RunInTransaction(ctx context.Context, fn func(repositories port.GrantRepositories) error) error {
	return runInTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		return fn(port.GrantRepositories{
			ProcessedEvents: &PostgresProcessedEventRepository{db: tx},
			CardPacks:       &PostgresCardPackRepository{db: tx},
			PlayerCards:     &PostgresPlayerCardRepository{db: tx},
		})
	})
}
