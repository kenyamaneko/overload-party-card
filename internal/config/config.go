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

	// DatabaseIAMAuthEnabled は Cloud SQL への接続を Cloud SQL Go Connector 経由の
	// 自動 IAM データベース認証で行うかどうかを表す。
	DatabaseIAMAuthEnabled bool

	// CloudSQLConnectionName は Cloud SQL インスタンスの接続名 (project:region:instance)。
	// DatabaseIAMAuthEnabled が true のときのみ必須。
	CloudSQLConnectionName string

	// InternalAuthPublicKey は内部サービス間 JWT (RS256) を検証する gateway の公開鍵。PEM 形式。
	InternalAuthPublicKey string

	// AccountServiceURL はデッキ検証時に faction 所持を照会する account サービスの URL。
	AccountServiceURL string
}

// FromEnv は環境変数から Config を構築します。
// 未設定の必須環境変数があれば即エラーで返し、デフォルトへの暗黙 fallback は行いません。
func FromEnv() (*Config, error) {
	cfg := &Config{
		DatabaseConn:          os.Getenv("DATABASE_CONN"),
		InternalAuthPublicKey: os.Getenv("INTERNAL_AUTH_PUBLIC_KEY"),
		AccountServiceURL:     os.Getenv("ACCOUNT_SERVICE_URL"),
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

	rawIAMAuth := os.Getenv("DATABASE_IAM_AUTH_ENABLED")
	switch rawIAMAuth {
	case "true":
		cfg.DatabaseIAMAuthEnabled = true
	case "false":
		cfg.DatabaseIAMAuthEnabled = false
	default:
		return nil, fmt.Errorf("config: DATABASE_IAM_AUTH_ENABLED must be %q or %q, got %q", "true", "false", rawIAMAuth)
	}
	if cfg.DatabaseIAMAuthEnabled {
		cfg.CloudSQLConnectionName = os.Getenv("CLOUDSQL_CONNECTION_NAME")
		if cfg.CloudSQLConnectionName == "" {
			return nil, fmt.Errorf("config: CLOUDSQL_CONNECTION_NAME is required when DATABASE_IAM_AUTH_ENABLED is true")
		}
	}

	if cfg.InternalAuthPublicKey == "" {
		return nil, fmt.Errorf("config: INTERNAL_AUTH_PUBLIC_KEY is required")
	}
	if cfg.AccountServiceURL == "" {
		return nil, fmt.Errorf("config: ACCOUNT_SERVICE_URL is required (card validates deck faction ownership via account)")
	}
	return cfg, nil
}
