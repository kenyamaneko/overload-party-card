// Package repositorytest は PostgreSQL に対するテストの共通部品を提供する。
//
// テスト用の PostgreSQL コンテナを Testcontainers で起動して接続プールを返す。
// コンテナはパッケージごとに 1 回だけ起動し (RunMain)、テスト間の状態は Truncate で空に戻す。
package repositorytest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	postgresImage    = "postgres:16-alpine"
	databaseName     = "test"
	databaseUser     = "test"
	databasePassword = "test"

	readyLogMessage = "database system is ready to accept connections"
	// コンテナは初期化用の一時起動と本起動の 2 回、起動完了のログを出すため、2 回目まで待つ。
	readyLogOccurrences = 2
	startupTimeout      = 60 * time.Second
)

// Postgres は起動済みの PostgreSQL コンテナと接続プールです。
type Postgres struct {
	Container *postgres.PostgresContainer
	Pool      *pgxpool.Pool
	schemas   []string
}

type config struct {
	schemaFiles []string
	schemas     []string
}

// Option は Start の設定です。
type Option func(*config)

// WithSchemaFile はスキーマ SQL のファイルを、リポジトリのルートからの相対パスで登録します。
// 登録した順に適用します。
func WithSchemaFile(repoRelativePath string) Option {
	return func(c *config) {
		c.schemaFiles = append(c.schemaFiles, repoRelativePath)
	}
}

// WithSchema は Truncate の対象にするスキーマを登録します。複数指定できます。
func WithSchema(name string) Option {
	return func(c *config) {
		c.schemas = append(c.schemas, name)
	}
}

// Start は PostgreSQL コンテナを起動し、登録されたスキーマ SQL を順に適用して接続プールを返します。
// 終了時は Close を呼びます (RunMain が呼びます)。
func Start(ctx context.Context, opts ...Option) (*Postgres, error) {
	cfg := &config{}
	for _, option := range opts {
		option(cfg)
	}
	if len(cfg.schemaFiles) == 0 {
		return nil, errors.New("repositorytest: at least one WithSchemaFile is required")
	}
	if len(cfg.schemas) == 0 {
		return nil, errors.New("repositorytest: at least one WithSchema is required")
	}

	root, err := findModuleRoot()
	if err != nil {
		return nil, err
	}

	container, err := postgres.Run(ctx,
		postgresImage,
		postgres.WithDatabase(databaseName),
		postgres.WithUsername(databaseUser),
		postgres.WithPassword(databasePassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog(readyLogMessage).
				WithOccurrence(readyLogOccurrences).
				WithStartupTimeout(startupTimeout),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, errors.Join(fmt.Errorf("container connection string: %w", err), container.Terminate(ctx))
	}

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("parse pool config: %w", err), container.Terminate(ctx))
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("pgxpool new: %w", err), container.Terminate(ctx))
	}

	for _, schemaFile := range cfg.schemaFiles {
		path := filepath.Join(root, schemaFile)
		schemaSQL, err := os.ReadFile(path)
		if err != nil {
			pool.Close()
			return nil, errors.Join(fmt.Errorf("read schema %s: %w", path, err), container.Terminate(ctx))
		}
		if _, err := pool.Exec(ctx, string(schemaSQL)); err != nil {
			pool.Close()
			return nil, errors.Join(fmt.Errorf("apply schema %s: %w", path, err), container.Terminate(ctx))
		}
	}

	return &Postgres{Container: container, Pool: pool, schemas: cfg.schemas}, nil
}

// Close は接続プールを閉じてコンテナを終了します。
func (p *Postgres) Close(ctx context.Context) error {
	p.Pool.Close()
	if err := p.Container.Terminate(ctx); err != nil {
		return fmt.Errorf("terminate container: %w", err)
	}
	return nil
}

// Truncate は登録したスキーマの全テーブルを空にします。
// テーブルを追加してもこの関数を直さずに済むよう、information_schema から対象を列挙します。
func (p *Postgres) Truncate(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	rows, err := p.Pool.Query(ctx, `
		SELECT table_schema || '.' || table_name
		  FROM information_schema.tables
		 WHERE table_schema = ANY($1)
		   AND table_type = 'BASE TABLE'
	`, p.schemas)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate tables: %v", err)
	}
	if len(tables) == 0 {
		return
	}

	statement := "TRUNCATE " + strings.Join(tables, ", ") + " RESTART IDENTITY CASCADE"
	if _, err := p.Pool.Exec(ctx, statement); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// RunMain は TestMain の処理をまとめます。コンテナの起動、テストの実行、後始末の順に行い、終了コードを返します。
// 各テストパッケージは `os.Exit(repositorytest.RunMain(m, &sharedPg, opts...))` と書き、
// `sharedPg.Pool` と `sharedPg.Truncate(t)` を使います。後始末に失敗したときは、テストが成功していても失敗の終了コードを返します。
func RunMain(m *testing.M, out **Postgres, opts ...Option) int {
	ctx := context.Background()

	pg, err := Start(ctx, opts...)
	if err != nil {
		slog.Error("repositorytest start failed", "error", err)
		return 1
	}
	*out = pg

	exitCode := m.Run()
	if err := pg.Close(ctx); err != nil {
		slog.Error("repositorytest close failed", "error", err)
		if exitCode == 0 {
			exitCode = 1
		}
	}
	return exitCode
}

// findModuleRoot はこのファイルの位置から親ディレクトリをたどり、go.mod があるディレクトリを返します。
func findModuleRoot() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("runtime.Caller failed")
	}
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found from repositorytest")
		}
		dir = parent
	}
}
