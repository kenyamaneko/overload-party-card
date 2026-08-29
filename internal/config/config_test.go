package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/config"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("PORT", "8080")
	t.Setenv("ENV", "local")
	t.Setenv("DATABASE_CONN", "postgres://test")
	t.Setenv("DATABASE_IAM_AUTH_ENABLED", "false")
	t.Setenv("INTERNAL_AUTH_PUBLIC_KEY", "dummy-public-key")
	t.Setenv("ACCOUNT_SERVICE_URL", "http://account.internal.test")
}

func TestConfigFromEnv(t *testing.T) {
	t.Run("[config] 起動設定の読み込み", func(t *testing.T) {
		t.Run("環境変数PORTが未設定のとき、起動設定の読み込みはエラーになる", func(t *testing.T) {
			setValidEnv(t)
			require.NoError(t, os.Unsetenv("PORT"))

			_, err := config.FromEnv()

			assert.ErrorContains(t, err, "PORT is required")
		})

		t.Run("環境変数PORTが整数として解釈できない値のとき、起動設定の読み込みはエラーになる", func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("PORT", "not-a-number")

			_, err := config.FromEnv()

			assert.ErrorContains(t, err, `PORT "not-a-number"`)
		})

		t.Run("環境変数PORTが整数として解釈できる値のとき、起動設定のポート番号としてその値が読み取られる", func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("PORT", "9090")

			cfg, err := config.FromEnv()

			require.NoError(t, err)
			assert.Equal(t, 9090, cfg.Port)
		})

		t.Run("環境変数ENVが未設定のとき、起動設定の読み込みはエラーになる", func(t *testing.T) {
			setValidEnv(t)
			require.NoError(t, os.Unsetenv("ENV"))

			_, err := config.FromEnv()

			assert.ErrorContains(t, err, "ENV is required")
		})

		t.Run("環境変数ENVがlocal/dev/stg/prodのいずれでもない値のとき、起動設定の読み込みはエラーになる", func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("ENV", "invalid-env")

			_, err := config.FromEnv()

			assert.ErrorContains(t, err, "ENV must be one of")
		})

		envCases := []string{"local", "dev", "stg", "prod"}
		for _, envValue := range envCases {
			t.Run("環境変数ENVが"+envValue+"のとき、起動設定の動作環境としてその値が読み取られる", func(t *testing.T) {
				setValidEnv(t)
				t.Setenv("ENV", envValue)

				cfg, err := config.FromEnv()

				require.NoError(t, err)
				assert.Equal(t, config.Env(envValue), cfg.Env)
			})
		}

		t.Run("環境変数DATABASE_CONNが未設定(空文字含む)のとき、起動設定の読み込みはエラーになる", func(t *testing.T) {
			setValidEnv(t)
			require.NoError(t, os.Unsetenv("DATABASE_CONN"))

			_, err := config.FromEnv()

			assert.ErrorContains(t, err, "DATABASE_CONN")
		})

		t.Run("環境変数DATABASE_CONNが設定されているとき、起動設定のデータベース接続文字列としてその値が読み取られる", func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("DATABASE_CONN", "postgres://custom")

			cfg, err := config.FromEnv()

			require.NoError(t, err)
			assert.Equal(t, "postgres://custom", cfg.DatabaseConn)
		})

		t.Run("環境変数DATABASE_IAM_AUTH_ENABLEDがtrueでもfalseでもない値(未設定含む)のとき、起動設定の読み込みはエラーになる", func(t *testing.T) {
			setValidEnv(t)
			require.NoError(t, os.Unsetenv("DATABASE_IAM_AUTH_ENABLED"))

			_, err := config.FromEnv()

			assert.ErrorContains(t, err, "DATABASE_IAM_AUTH_ENABLED must be")
		})

		t.Run("環境変数DATABASE_IAM_AUTH_ENABLEDがtrueで、環境変数CLOUDSQL_CONNECTION_NAMEが未設定(空文字含む)のとき、起動設定の読み込みはエラーになる", func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("DATABASE_IAM_AUTH_ENABLED", "true")
			t.Setenv("CLOUDSQL_CONNECTION_NAME", "placeholder")
			require.NoError(t, os.Unsetenv("CLOUDSQL_CONNECTION_NAME"))

			_, err := config.FromEnv()

			assert.ErrorContains(t, err, "CLOUDSQL_CONNECTION_NAME")
		})

		t.Run("環境変数DATABASE_IAM_AUTH_ENABLEDがtrueで、環境変数CLOUDSQL_CONNECTION_NAMEが設定されているとき、起動設定はCloud SQL IAM認証が有効な状態として読み込まれる", func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("DATABASE_IAM_AUTH_ENABLED", "true")
			t.Setenv("CLOUDSQL_CONNECTION_NAME", "proj:region:instance")

			cfg, err := config.FromEnv()

			require.NoError(t, err)
			assert.True(t, cfg.DatabaseIAMAuthEnabled)
		})

		t.Run("環境変数DATABASE_IAM_AUTH_ENABLEDがtrueで、環境変数CLOUDSQL_CONNECTION_NAMEが設定されているとき、起動設定のCloud SQL接続名としてその値が読み取られる", func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("DATABASE_IAM_AUTH_ENABLED", "true")
			t.Setenv("CLOUDSQL_CONNECTION_NAME", "proj:region:instance")

			cfg, err := config.FromEnv()

			require.NoError(t, err)
			assert.Equal(t, "proj:region:instance", cfg.CloudSQLConnectionName)
		})

		t.Run("環境変数DATABASE_IAM_AUTH_ENABLEDがfalseのとき、環境変数CLOUDSQL_CONNECTION_NAMEが設定されていても、起動設定のCloud SQL接続名は空文字のままになる", func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("DATABASE_IAM_AUTH_ENABLED", "false")
			t.Setenv("CLOUDSQL_CONNECTION_NAME", "proj:region:instance")

			cfg, err := config.FromEnv()

			require.NoError(t, err)
			assert.Empty(t, cfg.CloudSQLConnectionName)
		})

		t.Run("環境変数INTERNAL_AUTH_PUBLIC_KEYが未設定(空文字含む)のとき、起動設定の読み込みはエラーになる", func(t *testing.T) {
			setValidEnv(t)
			require.NoError(t, os.Unsetenv("INTERNAL_AUTH_PUBLIC_KEY"))

			_, err := config.FromEnv()

			assert.ErrorContains(t, err, "INTERNAL_AUTH_PUBLIC_KEY")
		})

		t.Run("環境変数INTERNAL_AUTH_PUBLIC_KEYが設定されているとき、起動設定の内部認証公開鍵としてその値が読み取られる", func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("INTERNAL_AUTH_PUBLIC_KEY", "custom-public-key")

			cfg, err := config.FromEnv()

			require.NoError(t, err)
			assert.Equal(t, "custom-public-key", cfg.InternalAuthPublicKey)
		})

		t.Run("環境変数ACCOUNT_SERVICE_URLが未設定(空文字含む)のとき、起動設定の読み込みはエラーになる", func(t *testing.T) {
			setValidEnv(t)
			require.NoError(t, os.Unsetenv("ACCOUNT_SERVICE_URL"))

			_, err := config.FromEnv()

			assert.ErrorContains(t, err, "ACCOUNT_SERVICE_URL")
		})

		t.Run("環境変数ACCOUNT_SERVICE_URLが設定されているとき、起動設定のaccountサービスURLとしてその値が読み取られる", func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("ACCOUNT_SERVICE_URL", "http://custom.account.test")

			cfg, err := config.FromEnv()

			require.NoError(t, err)
			assert.Equal(t, "http://custom.account.test", cfg.AccountServiceURL)
		})
	})
}
