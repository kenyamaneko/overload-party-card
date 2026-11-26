package main

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/config"
)

func TestDatabasePoolCreationFailure(t *testing.T) {
	t.Run("[データベース接続]接続プールの作成", func(t *testing.T) {
		t.Run("異常系", func(t *testing.T) {
			t.Run("接続文字列がnot-a-dsnのとき", func(t *testing.T) {
				tests := []struct {
					name string
					cfg  *config.Config
				}{
					{
						name: "CloudSQLのIAM認証を使わない設定のとき、接続文字列を読めないことを示すエラーになる",
						cfg:  &config.Config{DatabaseConn: "not-a-dsn"},
					},
					{
						name: "CloudSQLのIAM認証を使う設定のとき、接続文字列を読めないことを示すエラーになる",
						cfg: &config.Config{
							DatabaseConn:             "not-a-dsn",
							IsDatabaseIAMAuthEnabled: true,
							CloudSQLConnectionName:   "TST-project:TST-region:TST-instance",
						},
					},
				}
				for _, tt := range tests {
					t.Run(tt.name, func(t *testing.T) {
						_, _, err := newDatabasePool(context.Background(), tt.cfg)

						var parseConfigErr *pgconn.ParseConfigError
						assert.ErrorAs(t, err, &parseConfigErr)
					})
				}
			})
		})
	})
}
