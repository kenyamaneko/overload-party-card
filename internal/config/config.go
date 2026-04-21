package config

import (
	"fmt"
	"os"
	"strconv"

	pubsubevents "github.com/kenyamaneko/overload-party-common/packages/pubsub-events"
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
	DatabaseURL string

	PubsubProjectID             string
	FactionSelectedSubscription string

	// FirestoreProjectID は game_config の読み取り先プロジェクト ID。
	// ローカル/CI では FIRESTORE_EMULATOR_HOST を別途設定することでエミュレーターに接続。
	FirestoreProjectID string
}

// FromEnv は環境変数から Config を構築します。
// 未設定の必須環境変数があれば即エラーで返し、デフォルトへの暗黙 fallback は行いません。
func FromEnv() (*Config, error) {
	cfg := &Config{
		Port:                        9003,
		DatabaseURL:                 os.Getenv("DATABASE_URL"),
		PubsubProjectID:             os.Getenv("PUBSUB_PROJECT_ID"),
		FactionSelectedSubscription: getEnv("FACTION_SELECTED_SUBSCRIPTION", pubsubevents.SubFactionSelectedCard),
		FirestoreProjectID:          os.Getenv("FIRESTORE_PROJECT_ID"),
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

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}
	if cfg.PubsubProjectID == "" {
		return nil, fmt.Errorf("config: PUBSUB_PROJECT_ID is required (card subscribes to faction-selected events)")
	}
	if cfg.FirestoreProjectID == "" {
		return nil, fmt.Errorf("config: FIRESTORE_PROJECT_ID is required (game_config)")
	}
	return cfg, nil
}

// getEnv は未設定時に定数の fallback を返すヘルパです。リテラル fallback は禁止 (CLAUDE.md) のため、
// 呼び出し側は fallback に common パッケージの定数のみを渡します。
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
