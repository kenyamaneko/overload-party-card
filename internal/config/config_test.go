package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPublicKeyPEM は config が値をそのまま保持することの確認にだけ使うダミー。
// 鍵としての妥当性は検証しないため、PEM の体裁だけ揃えている。
const testPublicKeyPEM = "-----BEGIN PUBLIC KEY-----\ndummy-not-a-real-key\n-----END PUBLIC KEY-----\n"

var allEnvKeys = []string{
	"PORT",
	"ENV",
	"DATABASE_CONN",
	"DATABASE_IAM_AUTH_ENABLED",
	"CLOUDSQL_CONNECTION_NAME",
	"INTERNAL_AUTH_PUBLIC_KEY",
	"ACCOUNT_SERVICE_URL",
}

// setEnv は os.Getenv が "" と unset を区別しない性質を使い、未指定キーに "" を渡して未設定を再現する。
func setEnv(t *testing.T, envs map[string]string) {
	t.Helper()
	for _, k := range allEnvKeys {
		t.Setenv(k, envs[k])
	}
}

func mergeEnv(maps ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

var validEnv = map[string]string{
	"PORT":                      "9003",
	"ENV":                       "dev",
	"DATABASE_CONN":             "host=localhost port=5432 dbname=card user=card password=card sslmode=disable",
	"DATABASE_IAM_AUTH_ENABLED": "false",
	"INTERNAL_AUTH_PUBLIC_KEY":  testPublicKeyPEM,
	"ACCOUNT_SERVICE_URL":       "http://localhost:9005",
}

func TestFromEnv(t *testing.T) {
	t.Run("[config]環境変数からのConfig構築", func(t *testing.T) {
		t.Run("必須envが揃うとき、全フィールドがConfigに伝搬する", func(t *testing.T) {
			setEnv(t, validEnv)

			cfg, err := FromEnv()

			require.NoError(t, err)
			assert.Equal(t, 9003, cfg.Port)
			assert.Equal(t, Env("dev"), cfg.Env)
			assert.Equal(t, "host=localhost port=5432 dbname=card user=card password=card sslmode=disable", cfg.DatabaseConn)
			assert.Equal(t, testPublicKeyPEM, cfg.InternalAuthPublicKey)
			assert.Equal(t, "http://localhost:9005", cfg.AccountServiceURL)
		})

		t.Run("DATABASE_IAM_AUTH_ENABLEDがfalseのとき、CLOUDSQL_CONNECTION_NAMEが未設定でも成功する", func(t *testing.T) {
			setEnv(t, validEnv)

			cfg, err := FromEnv()

			require.NoError(t, err)
			assert.False(t, cfg.DatabaseIAMAuthEnabled)
			assert.Empty(t, cfg.CloudSQLConnectionName)
		})

		t.Run("DATABASE_IAM_AUTH_ENABLEDがtrueかつCLOUDSQL_CONNECTION_NAMEが指定されるとき、両方の値がConfigに反映される", func(t *testing.T) {
			setEnv(t, mergeEnv(validEnv, map[string]string{
				"DATABASE_IAM_AUTH_ENABLED": "true",
				"CLOUDSQL_CONNECTION_NAME":  "overload-party-dev:asia-northeast1:overload-party-db",
			}))

			cfg, err := FromEnv()

			require.NoError(t, err)
			assert.True(t, cfg.DatabaseIAMAuthEnabled)
			assert.Equal(t, "overload-party-dev:asia-northeast1:overload-party-db", cfg.CloudSQLConnectionName)
		})

		acceptedEnvCases := []struct {
			name string
			env  string
			want Env
		}{
			{name: `ENVが "local" のとき、Configの環境がlocalになる`, env: "local", want: Env("local")},
			{name: `ENVが "dev" のとき、Configの環境がdevになる`, env: "dev", want: Env("dev")},
			{name: `ENVが "stg" のとき、Configの環境がstgになる`, env: "stg", want: Env("stg")},
			{name: `ENVが "prod" のとき、Configの環境がprodになる`, env: "prod", want: Env("prod")},
		}
		for _, tc := range acceptedEnvCases {
			t.Run(tc.name, func(t *testing.T) {
				setEnv(t, mergeEnv(validEnv, map[string]string{"ENV": tc.env}))

				cfg, err := FromEnv()

				require.NoError(t, err)
				assert.Equal(t, tc.want, cfg.Env)
			})
		}

		invalidCases := []struct {
			name    string
			envs    map[string]string
			wantErr string
		}{
			{
				name:    "PORTが未設定のとき、エラーになる",
				envs:    mergeEnv(validEnv, map[string]string{"PORT": ""}),
				wantErr: "PORT is required",
			},
			{
				name:    "PORTが数値でないとき、エラーになる",
				envs:    mergeEnv(validEnv, map[string]string{"PORT": "not-a-number"}),
				wantErr: "PORT",
			},
			{
				name:    "ENVが未設定のとき、エラーになる",
				envs:    mergeEnv(validEnv, map[string]string{"ENV": ""}),
				wantErr: "ENV is required",
			},
			{
				name:    "ENVが未定義値のとき、エラーになる",
				envs:    mergeEnv(validEnv, map[string]string{"ENV": "staging"}),
				wantErr: "ENV must be",
			},
			{
				name:    "DATABASE_CONNが未設定のとき、エラーになる",
				envs:    mergeEnv(validEnv, map[string]string{"DATABASE_CONN": ""}),
				wantErr: "DATABASE_CONN is required",
			},
			{
				name:    "DATABASE_IAM_AUTH_ENABLEDが未設定のとき、変数名を含むエラーになる",
				envs:    mergeEnv(validEnv, map[string]string{"DATABASE_IAM_AUTH_ENABLED": ""}),
				wantErr: "DATABASE_IAM_AUTH_ENABLED must be",
			},
			{
				name:    `DATABASE_IAM_AUTH_ENABLEDが "true"/"false" 以外の "yes" のとき、変数名を含むエラーになる`,
				envs:    mergeEnv(validEnv, map[string]string{"DATABASE_IAM_AUTH_ENABLED": "yes"}),
				wantErr: "DATABASE_IAM_AUTH_ENABLED must be",
			},
			{
				name: "DATABASE_IAM_AUTH_ENABLEDがtrueかつCLOUDSQL_CONNECTION_NAMEが未設定のとき、CLOUDSQL_CONNECTION_NAMEを含むエラーになる",
				envs: mergeEnv(validEnv, map[string]string{
					"DATABASE_IAM_AUTH_ENABLED": "true",
					"CLOUDSQL_CONNECTION_NAME":  "",
				}),
				wantErr: "CLOUDSQL_CONNECTION_NAME is required",
			},
			{
				name:    "INTERNAL_AUTH_PUBLIC_KEYが未設定のとき、エラーになる",
				envs:    mergeEnv(validEnv, map[string]string{"INTERNAL_AUTH_PUBLIC_KEY": ""}),
				wantErr: "INTERNAL_AUTH_PUBLIC_KEY is required",
			},
			{
				name:    "ACCOUNT_SERVICE_URLが未設定のとき、エラーになる",
				envs:    mergeEnv(validEnv, map[string]string{"ACCOUNT_SERVICE_URL": ""}),
				wantErr: "ACCOUNT_SERVICE_URL is required",
			},
		}
		for _, tc := range invalidCases {
			t.Run(tc.name, func(t *testing.T) {
				setEnv(t, tc.envs)

				_, err := FromEnv()

				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			})
		}
	})
}
