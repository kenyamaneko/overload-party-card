package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/config"
)

func TestSetupLogger(t *testing.T) {
	t.Run("ログ出力の初期化", func(t *testing.T) {
		acceptedCases := []struct {
			name string
			env  config.Env
		}{
			{name: `ENV が "local" のとき、ロガーを初期化できる`, env: config.Env("local")},
			{name: `ENV が "dev" のとき、ロガーを初期化できる`, env: config.Env("dev")},
			{name: `ENV が "stg" のとき、ロガーを初期化できる`, env: config.Env("stg")},
			{name: `ENV が "prod" のとき、ロガーを初期化できる`, env: config.Env("prod")},
		}
		for _, tc := range acceptedCases {
			t.Run(tc.name, func(t *testing.T) {
				err := setupLogger(tc.env)

				require.NoError(t, err)
			})
		}

		t.Run(`ENV が "staging" のとき、その値を含むエラーになる`, func(t *testing.T) {
			err := setupLogger(config.Env("staging"))

			require.Error(t, err)
			assert.Contains(t, err.Error(), "staging")
		})
	})
}
