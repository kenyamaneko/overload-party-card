package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/config"
)

func setRequiredEnv(t *testing.T, overrides map[string]string) {
	t.Helper()
	defaults := map[string]string{
		"PORT":                             "8080",
		"ENV":                              "dev",
		"DATABASE_CONN":                    "postgres://test",
		"GOOGLE_CLOUD_PROJECT_ID":          "test-project",
		"PLAYER_ONBOARDED_SUBSCRIPTION":    "test-player-onboarded-sub",
		"CARD_PACK_PURCHASED_SUBSCRIPTION": "test-card-pack-purchased-sub",
		"INTERNAL_AUTH_SECRET":             "test-internal-auth-secret",
		"ACCOUNT_SERVICE_URL":              "http://account.test",
	}
	for k, v := range defaults {
		t.Setenv(k, v)
	}
	for k, v := range overrides {
		t.Setenv(k, v)
	}
}

func TestFromEnv(t *testing.T) {
	t.Run("環境変数からの設定構築", func(t *testing.T) {
		t.Run("必須の環境変数が全て設定されているとき、設定が構築される", func(t *testing.T) {
			setRequiredEnv(t, nil)

			cfg, err := config.FromEnv()

			require.NoError(t, err)
			assert.Equal(t, 8080, cfg.Port)
			assert.Equal(t, config.EnvDev, cfg.Env)
			assert.Equal(t, "postgres://test", cfg.DatabaseConn)
			assert.Equal(t, "test-project", cfg.GoogleCloudProjectID)
			assert.Equal(t, "test-player-onboarded-sub", cfg.PlayerOnboardedSubscription)
			assert.Equal(t, "test-card-pack-purchased-sub", cfg.CardPackPurchasedSubscription)
			assert.Equal(t, "test-internal-auth-secret", cfg.InternalAuthSecret)
			assert.Equal(t, "http://account.test", cfg.AccountServiceURL)
		})

		tests := []struct {
			name       string
			override   map[string]string
			wantErrMsg string
		}{
			{"PORT が未設定のとき、エラーになる", map[string]string{"PORT": ""}, "PORT is required"},
			{"PORT が数値でないとき、エラーになる", map[string]string{"PORT": "abc"}, `PORT "abc"`},
			{"ENV が未設定のとき、エラーになる", map[string]string{"ENV": ""}, "ENV is required"},
			{"ENV が未知の値のとき、エラーになる", map[string]string{"ENV": "local"}, "ENV must be"},
			{"DATABASE_CONN が未設定のとき、エラーになる", map[string]string{"DATABASE_CONN": ""}, "DATABASE_CONN is required"},
			{"GOOGLE_CLOUD_PROJECT_ID が未設定のとき、エラーになる", map[string]string{"GOOGLE_CLOUD_PROJECT_ID": ""}, "GOOGLE_CLOUD_PROJECT_ID is required"},
			{"PLAYER_ONBOARDED_SUBSCRIPTION が未設定のとき、エラーになる", map[string]string{"PLAYER_ONBOARDED_SUBSCRIPTION": ""}, "PLAYER_ONBOARDED_SUBSCRIPTION is required"},
			{"CARD_PACK_PURCHASED_SUBSCRIPTION が未設定のとき、エラーになる", map[string]string{"CARD_PACK_PURCHASED_SUBSCRIPTION": ""}, "CARD_PACK_PURCHASED_SUBSCRIPTION is required"},
			{"INTERNAL_AUTH_SECRET が未設定のとき、エラーになる", map[string]string{"INTERNAL_AUTH_SECRET": ""}, "INTERNAL_AUTH_SECRET is required"},
			{"ACCOUNT_SERVICE_URL が未設定のとき、エラーになる", map[string]string{"ACCOUNT_SERVICE_URL": ""}, "ACCOUNT_SERVICE_URL is required"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				setRequiredEnv(t, tt.override)

				_, err := config.FromEnv()

				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			})
		}
	})
}
