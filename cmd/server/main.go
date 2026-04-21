package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/config"
	"github.com/kenyamaneko/overload-party-card/internal/handler/rest"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
	"github.com/kenyamaneko/overload-party-card/internal/router"
	"github.com/kenyamaneko/overload-party-card/internal/service"
	pubsubadapter "github.com/kenyamaneko/overload-party-card/internal/adapter/pubsub"
)

func main() {
	if err := run(); err != nil {
		slog.Error("card fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}

	if err := setupLogger(cfg.Env); err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("pgxpool new: %w", err)
	}
	defer pool.Close()

	cardRepo := repository.NewPgCardRepository(pool)
	playerCardRepo := repository.NewPgPlayerCardRepository(pool)
	deckRepo := repository.NewPgDeckRepository(pool)
	eventRepo := repository.NewPgProcessedEventRepository(pool)

	cardCache := cache.NewCardCache()
	if err := cardCache.Load(ctx, cardRepo); err != nil {
		return fmt.Errorf("load card cache: %w", err)
	}

	cardSvc := service.NewCardService(cardRepo, playerCardRepo)
	deckSvc := service.NewDeckService(deckRepo, playerCardRepo, cardCache)
	playerCardSvc := service.NewPlayerCardService(playerCardRepo, cardCache)
	grantSvc := service.NewGrantService(cardRepo, playerCardRepo)

	cardH := rest.NewCardHandler(cardSvc)
	deckH := rest.NewDeckHandler(deckSvc)
	playerCardH := rest.NewPlayerCardHandler(playerCardSvc)
	grantH := rest.NewGrantHandler(grantSvc)

	r := router.New(cardH, deckH, playerCardH, grantH)

	factionSub, err := pubsubadapter.NewFactionSelectedSubscriber(
		ctx, cfg.PubsubProjectID, cfg.FactionSelectedSubscription,
		grantSvc, eventRepo,
	)
	if err != nil {
		return fmt.Errorf("faction-selected subscriber: %w", err)
	}
	defer func() { _ = factionSub.Close() }()

	go func() {
		if err := factionSub.Start(ctx); err != nil && ctx.Err() == nil {
			slog.Error("faction-selected subscriber terminated", "error", err)
			os.Exit(1)
		}
	}()

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("card listening",
			"addr", srv.Addr,
			"card_cache_size", cardCache.Count(),
			"pubsub_project", cfg.PubsubProjectID,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("card shutdown requested")
	case err := <-errCh:
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	return nil
}

// setupLogger は ENV に応じてグローバル slog ロガーを初期化する。
// prod/stg は Cloud Logging 互換 JSON、dev は人間可読なテキストで出力する。
func setupLogger(env config.Env) error {
	switch env {
	case config.EnvProd, config.EnvStg:
		slog.SetDefault(slog.New(newCloudLoggingHandler()).With("service", "card"))
	case config.EnvDev:
		h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
		slog.SetDefault(slog.New(h).With("service", "card"))
	default:
		return fmt.Errorf("unexpected ENV: %s", env)
	}
	return nil
}

// newCloudLoggingHandler は Cloud Logging 互換の JSON ハンドラを返す。
// slog のデフォルトフィールド名・値では Cloud Logging が認識しないため変換する。
func newCloudLoggingHandler() slog.Handler {
	return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				a.Key = "severity"
				if level, ok := a.Value.Any().(slog.Level); ok {
					switch {
					case level >= slog.LevelError:
						a.Value = slog.StringValue("ERROR")
					case level >= slog.LevelWarn:
						a.Value = slog.StringValue("WARNING")
					case level >= slog.LevelInfo:
						a.Value = slog.StringValue("INFO")
					default:
						a.Value = slog.StringValue("DEBUG")
					}
				}
			}
			if a.Key == slog.MessageKey {
				a.Key = "message"
			}
			return a
		},
	})
}
