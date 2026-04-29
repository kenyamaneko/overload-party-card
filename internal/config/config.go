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
	Port        int
	Env         Env
	DatabaseConn string

	PubsubProjectID              string
	FactionPurchasedSubscription string
	PlayerOnboardedSubscription  string
}

// FromEnv は環境変数から Config を構築します。
// 未設定の必須環境変数があれば即エラーで返し、デフォルトへの暗黙 fallback は行いません。
func FromEnv() (*Config, error) {
	cfg := &Config{
		Port:                         9003,
		DatabaseConn:                 os.Getenv("DATABASE_CONN"),
		PubsubProjectID:              os.Getenv("PUBSUB_PROJECT_ID"),
		FactionPurchasedSubscription: getEnv("FACTION_PURCHASED_SUBSCRIPTION", "faction-purchased-card-sub"),
		PlayerOnboardedSubscription:  getEnv("PLAYER_ONBOARDED_SUBSCRIPTION", "player-onboarded-card-sub"),
	}

	if raw := os.Getenv("PORT"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("config: PORT %q: %w", raw, err)
		}
		cfg.Port = n
	}

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
	if cfg.PubsubProjectID == "" {
		return nil, fmt.Errorf("config: PUBSUB_PROJECT_ID is required (card subscribes to faction-purchased / player-onboarded events)")
	}
	return cfg, nil
}

// getEnv は未設定時に fallback を返すヘルパです。
// fallback は subscription 名のリテラル (SSoT は overload-party-infra の Terraform) を直接指定します。
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
