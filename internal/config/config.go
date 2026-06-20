package config

import (
	"fmt"
	"os"
	"strconv"
)

// Env は card サービスの動作環境を表します。
type Env string

const (
	// EnvDev は dev 環境。ローカル開発・CI も含む。
	EnvDev Env = "dev"
	// EnvStg は stg 環境。
	EnvStg Env = "stg"
	// EnvProd は prod 環境。
	EnvProd Env = "prod"
)

// Config は card サービスの起動設定を保持します。
type Config struct {
	Port         int
	Env          Env
	DatabaseConn string

	GoogleCloudProjectID          string
	PlayerOnboardedSubscription   string
	CardPackPurchasedSubscription string

	// InternalAuthSecret は内部サービス間 JWT (HS256) 検証の共有秘密鍵。
	InternalAuthSecret string

	// AccountServiceURL はデッキ検証時に faction 所持を照会する account サービスの URL。
	AccountServiceURL string
}

// FromEnv は環境変数から Config を構築します。
// 未設定の必須環境変数があれば即エラーで返し、デフォルトへの暗黙 fallback は行いません。
func FromEnv() (*Config, error) {
	cfg := &Config{
		DatabaseConn:                  os.Getenv("DATABASE_CONN"),
		GoogleCloudProjectID:          os.Getenv("GOOGLE_CLOUD_PROJECT_ID"),
		PlayerOnboardedSubscription:   os.Getenv("PLAYER_ONBOARDED_SUBSCRIPTION"),
		CardPackPurchasedSubscription: os.Getenv("CARD_PACK_PURCHASED_SUBSCRIPTION"),
		InternalAuthSecret:            os.Getenv("INTERNAL_AUTH_SECRET"),
		AccountServiceURL:             os.Getenv("ACCOUNT_SERVICE_URL"),
	}

	rawPort := os.Getenv("PORT")
	if rawPort == "" {
		return nil, fmt.Errorf("config: PORT is required")
	}
	n, err := strconv.Atoi(rawPort)
	if err != nil {
		return nil, fmt.Errorf("config: PORT %q: %w", rawPort, err)
	}
	cfg.Port = n

	envRaw := os.Getenv("ENV")
	if envRaw == "" {
		return nil, fmt.Errorf("config: ENV is required (one of %q, %q, %q)", EnvDev, EnvStg, EnvProd)
	}
	switch Env(envRaw) {
	case EnvDev, EnvStg, EnvProd:
		cfg.Env = Env(envRaw)
	default:
		return nil, fmt.Errorf("config: ENV must be %q, %q, or %q, got %q", EnvDev, EnvStg, EnvProd, envRaw)
	}

	if cfg.DatabaseConn == "" {
		return nil, fmt.Errorf("config: DATABASE_CONN is required")
	}
	if cfg.GoogleCloudProjectID == "" {
		return nil, fmt.Errorf("config: GOOGLE_CLOUD_PROJECT_ID is required (card subscribes to player-onboarded / card-pack-purchased events)")
	}
	if cfg.PlayerOnboardedSubscription == "" {
		return nil, fmt.Errorf("config: PLAYER_ONBOARDED_SUBSCRIPTION is required")
	}
	if cfg.CardPackPurchasedSubscription == "" {
		return nil, fmt.Errorf("config: CARD_PACK_PURCHASED_SUBSCRIPTION is required")
	}
	if cfg.InternalAuthSecret == "" {
		return nil, fmt.Errorf("config: INTERNAL_AUTH_SECRET is required")
	}
	if cfg.AccountServiceURL == "" {
		return nil, fmt.Errorf("config: ACCOUNT_SERVICE_URL is required (card validates deck faction ownership via account)")
	}
	return cfg, nil
}
