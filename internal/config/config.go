package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config は card サービスの起動設定を保持します。
type Config struct {
	Port        int
	Env         string
	DatabaseURL string

	PubsubProjectID             string
	FactionSelectedSubscription string
}

// FromEnv は環境変数から Config を構築します。
func FromEnv() (*Config, error) {
	cfg := &Config{
		Port:                        9003,
		Env:                         getEnv("ENV", "dev"),
		DatabaseURL:                 os.Getenv("DATABASE_URL"),
		PubsubProjectID:             os.Getenv("PUBSUB_PROJECT_ID"),
		FactionSelectedSubscription: getEnv("FACTION_SELECTED_SUBSCRIPTION", "faction-selected-card-sub"),
	}

	if raw := os.Getenv("PORT"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("config: PORT %q: %w", raw, err)
		}
		cfg.Port = n
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}
	if cfg.PubsubProjectID == "" {
		return nil, fmt.Errorf("config: PUBSUB_PROJECT_ID is required (card subscribes to faction-selected events)")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
