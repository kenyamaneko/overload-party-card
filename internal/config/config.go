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

	PubsubProjectID             string
	PlayerOnboardedSubscription string
}

// FromEnv は環境変数から Config を構築します。
// 未設定の必須環境変数があれば即エラーで返し、デフォルトへの暗黙 fallback は行いません。
func FromEnv() (*Config, error) {
	cfg := &Config{
		DatabaseConn:                os.Getenv("DATABASE_CONN"),
		PubsubProjectID:             os.Getenv("PUBSUB_PROJECT_ID"),
		PlayerOnboardedSubscription: os.Getenv("PLAYER_ONBOARDED_SUBSCRIPTION"),
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
	if cfg.PubsubProjectID == "" {
		return nil, fmt.Errorf("config: PUBSUB_PROJECT_ID is required (card subscribes to player-onboarded events)")
	}
	if cfg.PlayerOnboardedSubscription == "" {
		return nil, fmt.Errorf("config: PLAYER_ONBOARDED_SUBSCRIPTION is required")
	}
	return cfg, nil
}
