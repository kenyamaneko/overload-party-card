package main

import (
	"context"
	"errors"
	"fmt"
	"net"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/config"
)

// newDatabasePool は設定に応じた接続方式で DB 接続プールを構築する。
// 戻り値の関数は、接続プールの利用が終わったあとに呼んで、接続方式が持つ資源を解放する。
func newDatabasePool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, func() error, error) {
	if !cfg.IsDatabaseIAMAuthEnabled {
		pool, err := pgxpool.New(ctx, cfg.DatabaseConn)
		if err != nil {
			return nil, nil, fmt.Errorf("pgxpool new: %w", err)
		}
		return pool, func() error { return nil }, nil
	}
	return newDatabasePoolWithIAMAuth(ctx, cfg)
}

func newDatabasePoolWithIAMAuth(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, func() error, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseConn)
	if err != nil {
		return nil, nil, fmt.Errorf("parse database conn: %w", err)
	}

	dialer, err := cloudsqlconn.NewDialer(ctx,
		cloudsqlconn.WithIAMAuthN(),
		cloudsqlconn.WithDefaultDialOptions(cloudsqlconn.WithPrivateIP()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("cloudsqlconn new dialer: %w", err)
	}

	connectionName := cfg.CloudSQLConnectionName
	poolCfg.ConnConfig.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) {
		return dialer.Dial(ctx, connectionName)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, nil, errors.Join(fmt.Errorf("pgxpool new with config: %w", err), dialer.Close())
	}

	return pool, dialer.Close, nil
}
