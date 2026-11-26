//go:build integration

package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/config"
)

func TestDatabasePoolCreation(t *testing.T) {
	t.Run("[データベース接続]接続プールの作成", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("CloudSQLのIAM認証を使わない設定で、起動中のPostgreSQLの接続文字列から作成した接続プールで1を返す問い合わせを実行すると、結果は1になる", func(t *testing.T) {
				ctx := context.Background()
				connectionString, err := sharedPg.Container.ConnectionString(ctx, "sslmode=disable")
				require.NoError(t, err)
				pool, release, err := newDatabasePool(ctx, &config.Config{DatabaseConn: connectionString})
				require.NoError(t, err)
				t.Cleanup(func() {
					pool.Close()
					require.NoError(t, release())
				})

				var result int
				err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)

				require.NoError(t, err)
				assert.Equal(t, 1, result)
			})
		})
	})
}
