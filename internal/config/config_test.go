package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/config"
	"github.com/kenyamaneko/overload-party-card/internal/config/configtest"
)

func TestConfig(t *testing.T) {
	t.Run("[起動設定]起動設定の環境変数からの読み込み", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("必須の環境変数がすべて設定されていて、DATABASE_IAM_AUTH_ENABLEDがfalseのとき", func(t *testing.T) {
				t.Run("PORTが9090のとき、待受ポートは9090になる", func(t *testing.T) {
					configtest.SetEnvironment(t, map[string]string{"PORT": "9090"})

					cfg, err := config.LoadFromEnv()

					require.NoError(t, err)
					assert.Equal(t, 9090, cfg.Port)
				})

				t.Run("ENVの値ごとの動作環境", func(t *testing.T) {
					tests := []struct {
						name string
						env  string
					}{
						{name: "ENVがlocalのとき、動作環境はlocalになる", env: "local"},
						{name: "ENVがdevのとき、動作環境はdevになる", env: "dev"},
						{name: "ENVがstgのとき、動作環境はstgになる", env: "stg"},
						{name: "ENVがprodのとき、動作環境はprodになる", env: "prod"},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							configtest.SetEnvironment(t, map[string]string{"ENV": tt.env})

							cfg, err := config.LoadFromEnv()

							require.NoError(t, err)
							assert.Equal(t, tt.env, string(cfg.Env))
						})
					}
				})

				t.Run("CloudSQLのIAM認証は無効になる", func(t *testing.T) {
					configtest.SetEnvironment(t, nil)

					cfg, err := config.LoadFromEnv()

					require.NoError(t, err)
					assert.False(t, cfg.IsDatabaseIAMAuthEnabled)
				})
			})

			t.Run("DATABASE_IAM_AUTH_ENABLEDがtrueで、CLOUDSQL_CONNECTION_NAMEが設定されているとき、CloudSQLのIAM認証は有効になる", func(t *testing.T) {
				configtest.SetEnvironment(t, map[string]string{
					"DATABASE_IAM_AUTH_ENABLED": "true",
					"CLOUDSQL_CONNECTION_NAME":  "TST-project:TST-region:TST-instance",
				})

				cfg, err := config.LoadFromEnv()

				require.NoError(t, err)
				assert.True(t, cfg.IsDatabaseIAMAuthEnabled)
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("確かめる環境変数以外のすべての環境変数が、受け付けられる値で設定されているとき", func(t *testing.T) {
				t.Run("PORTが未設定のとき、PORTが必須であることを示すエラーになる", func(t *testing.T) {
					configtest.SetEnvironment(t, map[string]string{"PORT": ""})

					_, err := config.LoadFromEnv()

					assert.ErrorContains(t, err, "PORT is required")
				})

				t.Run("PORTがabcのとき、PORTの値が数値でないことを示すエラーになる", func(t *testing.T) {
					configtest.SetEnvironment(t, map[string]string{"PORT": "abc"})

					_, err := config.LoadFromEnv()

					assert.ErrorContains(t, err, `PORT "abc"`)
				})

				t.Run("ENVが未設定のとき、ENVが必須であることを示すエラーになる", func(t *testing.T) {
					configtest.SetEnvironment(t, map[string]string{"ENV": ""})

					_, err := config.LoadFromEnv()

					assert.ErrorContains(t, err, "ENV is required")
				})

				t.Run("ENVがtestのとき、ENVの値がlocal・dev・stg・prodのいずれでもないことを示すエラーになる", func(t *testing.T) {
					configtest.SetEnvironment(t, map[string]string{"ENV": "test"})

					_, err := config.LoadFromEnv()

					assert.ErrorContains(t, err, `ENV must be one of "local", "dev", "stg", "prod"`)
				})

				t.Run("DATABASE_CONNが未設定のとき、DATABASE_CONNが必須であることを示すエラーになる", func(t *testing.T) {
					configtest.SetEnvironment(t, map[string]string{"DATABASE_CONN": ""})

					_, err := config.LoadFromEnv()

					assert.ErrorContains(t, err, "DATABASE_CONN is required")
				})

				t.Run("DATABASE_IAM_AUTH_ENABLEDが大文字のTRUEのとき、trueかfalseのいずれかが必要であることを示すエラーになる", func(t *testing.T) {
					configtest.SetEnvironment(t, map[string]string{"DATABASE_IAM_AUTH_ENABLED": "TRUE"})

					_, err := config.LoadFromEnv()

					assert.ErrorContains(t, err, `DATABASE_IAM_AUTH_ENABLED must be "true" or "false"`)
				})

				t.Run("DATABASE_IAM_AUTH_ENABLEDがtrueで、CLOUDSQL_CONNECTION_NAMEが未設定のとき、CLOUDSQL_CONNECTION_NAMEが必須であることを示すエラーになる", func(t *testing.T) {
					configtest.SetEnvironment(t, map[string]string{
						"DATABASE_IAM_AUTH_ENABLED": "true",
						"CLOUDSQL_CONNECTION_NAME":  "",
					})

					_, err := config.LoadFromEnv()

					assert.ErrorContains(t, err, "CLOUDSQL_CONNECTION_NAME is required")
				})

				t.Run("INTERNAL_AUTH_PUBLIC_KEYが未設定のとき、INTERNAL_AUTH_PUBLIC_KEYが必須であることを示すエラーになる", func(t *testing.T) {
					configtest.SetEnvironment(t, map[string]string{"INTERNAL_AUTH_PUBLIC_KEY": ""})

					_, err := config.LoadFromEnv()

					assert.ErrorContains(t, err, "INTERNAL_AUTH_PUBLIC_KEY is required")
				})

				t.Run("ACCOUNT_SERVICE_URLが未設定のとき、ACCOUNT_SERVICE_URLが必須であることを示すエラーになる", func(t *testing.T) {
					configtest.SetEnvironment(t, map[string]string{"ACCOUNT_SERVICE_URL": ""})

					_, err := config.LoadFromEnv()

					assert.ErrorContains(t, err, "ACCOUNT_SERVICE_URL is required")
				})
			})
		})
	})
}
