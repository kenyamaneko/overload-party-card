package repositorytest

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

const (
	// PlayerOneID はテスト用の 1 人目のプレイヤー ID です。
	PlayerOneID = "00000000-0000-0000-0000-000000000001"
	// PlayerTwoID はテスト用の 2 人目のプレイヤー ID です。
	PlayerTwoID = "00000000-0000-0000-0000-000000000002"
	// AbsentDeckID は保存されていないデッキの ID として指定するテスト用の値です。
	AbsentDeckID int64 = 999999
	// EventTypeForTest は処理済みイベントの記録に渡すテスト用のイベント種別です。
	EventTypeForTest = "test_event"

	// SQLStateUniqueViolation は PostgreSQL の一意制約違反の SQLSTATE です。
	SQLStateUniqueViolation = "23505"
	// SQLStateInvalidTextRepresentation は PostgreSQL の不正な入力構文エラーの SQLSTATE です。
	SQLStateInvalidTextRepresentation = "22P02"

	unreachableDatabaseURL = "postgres://test:test@127.0.0.1:1/test?sslmode=disable"
)

// NewUnreachablePool は接続できないデータベースを指す接続プールを返します。
func NewUnreachablePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), unreachableDatabaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

// RequireConnectionError は err の原因がデータベースに接続できないことであると確かめます。
func RequireConnectionError(t *testing.T, err error) {
	t.Helper()
	var connectErr *pgconn.ConnectError
	require.ErrorAs(t, err, &connectErr)
}

// RequireSQLState は err の原因が PostgreSQL のエラーで、その SQLSTATE が wantCode であると確かめます。
func RequireSQLState(t *testing.T, err error, wantCode string) {
	t.Helper()
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, wantCode, pgErr.Code)
}

// ToPointer は値へのポインタを返します。
func ToPointer[T any](value T) *T {
	return &value
}

// ParseTime は RFC 3339 形式の文字列を時刻にします。
func ParseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return parsed
}

// RequireTimeEqual は PostgreSQL の保存精度に合わせて want をマイクロ秒に丸めたとき、got と同じ時刻であると確かめます。
func RequireTimeEqual(t *testing.T, want, got time.Time) {
	t.Helper()
	require.True(t, want.Round(time.Microsecond).Equal(got), "want %s, got %s", want, got)
}

// CountProcessedEvents は DB に保存されている処理済みイベントの件数を数えます。
func CountProcessedEvents(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var count int
	err := pool.QueryRow(t.Context(), `SELECT count(*) FROM card.processed_events`).Scan(&count)
	require.NoError(t, err)
	return count
}

// ToPlayerCards は所持カードのポインタの一覧を値の一覧にします。
func ToPlayerCards(cards []*domain.PlayerCard) []domain.PlayerCard {
	values := make([]domain.PlayerCard, 0, len(cards))
	for _, card := range cards {
		values = append(values, *card)
	}
	return values
}

// ProductRow はプロダクトの表へ直接挿入する行です。
type ProductRow struct {
	ProductID   string
	Faction     string
	ProductName string
	IsActive    bool
}

// NewProductRow は有効なテスト用のプロダクトの行を返します。
func NewProductRow(productID string) ProductRow {
	return ProductRow{
		ProductID:   productID,
		Faction:     "Tenki",
		ProductName: "テストプロダクト",
		IsActive:    true,
	}
}

// InsertProduct はプロダクトの行を表へ直接挿入します。
func InsertProduct(t *testing.T, pool *pgxpool.Pool, row ProductRow) {
	t.Helper()
	_, err := pool.Exec(t.Context(), `
		INSERT INTO card.products (product_id, faction, product_name, is_active)
		VALUES ($1, $2, $3, $4)`,
		row.ProductID, row.Faction, row.ProductName, row.IsActive,
	)
	require.NoError(t, err)
}
