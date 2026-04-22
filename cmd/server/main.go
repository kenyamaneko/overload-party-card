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
	"golang.org/x/sync/errgroup"

	pubsubadapter "github.com/kenyamaneko/overload-party-card/internal/adapter/pubsub"
	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/config"
	"github.com/kenyamaneko/overload-party-card/internal/handler/rest"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
	"github.com/kenyamaneko/overload-party-card/internal/router"
	"github.com/kenyamaneko/overload-party-card/internal/service"
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

	factionSub, err := pubsubadapter.NewFactionPurchasedSubscriber(
		ctx, cfg.PubsubProjectID, cfg.FactionPurchasedSubscription,
		grantSvc, eventRepo,
	)
	if err != nil {
		return fmt.Errorf("faction-purchased subscriber: %w", err)
	}
	defer func() {
		if cerr := factionSub.Close(); cerr != nil {
			slog.Warn("faction-purchased subscriber close", "error", cerr)
		}
	}()

	onboardedSub, err := pubsubadapter.NewPlayerOnboardedSubscriber(
		ctx, cfg.PubsubProjectID, cfg.PlayerOnboardedSubscription,
		grantSvc, eventRepo,
	)
	if err != nil {
		return fmt.Errorf("player-onboarded subscriber: %w", err)
	}
	defer func() {
		if cerr := onboardedSub.Close(); cerr != nil {
			slog.Warn("player-onboarded subscriber close", "error", cerr)
		}
	}()

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	slog.Info("card starting",
		"addr", srv.Addr,
		"card_cache_size", cardCache.Count(),
		"pubsub_project", cfg.PubsubProjectID,
	)

	return runHTTPAndSubscribers(ctx, srv, factionSub, onboardedSub)
}

// runHTTPAndSubscribers は HTTP server と Pub/Sub subscriber 群を並行起動し、
// どれかの失敗・シグナル到来で全体を graceful に停止する。
func runHTTPAndSubscribers(ctx context.Context, srv *http.Server, factionSub, onboardedSub subscriber) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		if err := factionSub.Start(gCtx); err != nil && gCtx.Err() == nil {
			return fmt.Errorf("faction-purchased subscriber: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		if err := onboardedSub.Start(gCtx); err != nil && gCtx.Err() == nil {
			return fmt.Errorf("player-onboarded subscriber: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		<-gCtx.Done()
		slog.Info("card shutdown requested")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http shutdown: %w", err)
		}
		return nil
	})

	return g.Wait()
}

// subscriber は Pub/Sub subscriber の起動インターフェース。
type subscriber interface {
	Start(ctx context.Context) error
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
