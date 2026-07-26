package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/config"
)

// newDatabasePool は cfg.DatabaseIAMAuthEnabled に応じて pgxpool を構築する。
// 有効なときは Cloud SQL Go Connector の dialer を通した private IP 経由の自動 IAM
// データベース認証で接続し、無効なときは cfg.DatabaseConn への直接接続を行う。
// 返す cleanup は dialer を保持している場合のみそれを閉じ、それ以外は no-op になる。
func newDatabasePool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, func(), error) {
	if !cfg.DatabaseIAMAuthEnabled {
		pool, err := pgxpool.New(ctx, cfg.DatabaseConn)
		if err != nil {
			return nil, nil, fmt.Errorf("pgxpool new: %w", err)
		}
		return pool, func() {}, nil
	}
	return newDatabasePoolWithIAMAuth(ctx, cfg)
}

// newDatabasePoolWithIAMAuth は Cloud SQL Go Connector の dialer を pgxpool の
// DialFunc に差し込み、private IP 経由の自動 IAM データベース認証で接続する。
func newDatabasePoolWithIAMAuth(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, func(), error) {
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
		closeDialer(dialer)
		return nil, nil, fmt.Errorf("pgxpool new with config: %w", err)
	}

	return pool, func() { closeDialer(dialer) }, nil
}

// closeDialer は Cloud SQL Go Connector の dialer を閉じる。呼び出し元はプロセス
// 終了処理の中で呼ぶため、失敗しても処理を継続しログにのみ残す。
func closeDialer(dialer *cloudsqlconn.Dialer) {
	if err := dialer.Close(); err != nil {
		slog.Error("cloudsqlconn dialer close failed", "error", err)
	}
}
