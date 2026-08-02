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

	"golang.org/x/sync/errgroup"

	accountadapter "github.com/kenyamaneko/overload-party-card/internal/adapter/account"
	pubsubadapter "github.com/kenyamaneko/overload-party-card/internal/adapter/pubsub"
	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/config"
	"github.com/kenyamaneko/overload-party-card/internal/handler/rest"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
	"github.com/kenyamaneko/overload-party-card/internal/router"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"

	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
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

	pool, closeDatabasePool, err := newDatabasePool(ctx, cfg)
	if err != nil {
		return fmt.Errorf("new database pool: %w", err)
	}
	defer closeDatabasePool()
	defer pool.Close()

	cardRepo := repository.NewPgCardRepository(pool)
	cardPackRepo := repository.NewPgCardPackRepository(pool)
	playerCardRepo := repository.NewPgPlayerCardRepository(pool)
	deckRepo := repository.NewPgDeckRepository(pool)
	productRepo := repository.NewPgProductRepository(pool)
	initiativeRepo := repository.NewPgInitiativeRepository(pool)
	eventRepo := repository.NewPgProcessedEventRepository(pool)

	cardCache := cache.NewCardCache()
	if err := cardCache.Load(ctx, cardRepo); err != nil {
		return fmt.Errorf("load card cache: %w", err)
	}

	productCache := cache.NewProductCache()
	if err := productCache.Load(ctx, productRepo); err != nil {
		return fmt.Errorf("load product cache: %w", err)
	}

	initiativeCache := cache.NewInitiativeCache()
	if err := initiativeCache.Load(ctx, initiativeRepo); err != nil {
		return fmt.Errorf("load initiative cache: %w", err)
	}

	accountClient := accountadapter.NewClient(cfg.AccountServiceURL)

	cardInteractor := usecase.NewCardInteractor(cardRepo, playerCardRepo)
	deckInteractor := usecase.NewDeckInteractor(deckRepo, playerCardRepo, cardCache, productCache, initiativeCache, accountClient)
	playerCardInteractor := usecase.NewPlayerCardInteractor(playerCardRepo, cardCache)
	grantInteractor := usecase.NewGrantInteractor(cardPackRepo, playerCardRepo)

	cardH := rest.NewCardHandler(cardInteractor)
	deckH := rest.NewDeckHandler(deckInteractor)
	playerCardH := rest.NewPlayerCardHandler(playerCardInteractor)
	productH := rest.NewProductHandler(productCache, initiativeCache)
	initiativeH := rest.NewInitiativeHandler(initiativeCache)

	internalAuthKey, err := internalauth.ParsePublicKeyPEM([]byte(cfg.InternalAuthPublicKey))
	if err != nil {
		return fmt.Errorf("INTERNAL_AUTH_PUBLIC_KEY is invalid: %w", err)
	}
	authVerifier := internalauth.NewVerifier(
		internalauth.StaticPublicKeyResolver(internalAuthKey, internalauth.DefaultKeyID),
	)

	onboardedSub := pubsubadapter.NewPlayerOnboardedSubscriber(grantInteractor, eventRepo)
	cardPackPurchasedSub := pubsubadapter.NewCardPackPurchasedSubscriber(grantInteractor, eventRepo)
	pubsubPushH := rest.NewPubSubPushHandler(onboardedSub.Handle, cardPackPurchasedSub.Handle)

	r := router.New(cardH, deckH, playerCardH, productH, initiativeH, pubsubPushH, authVerifier)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	slog.Info("card starting",
		"addr", srv.Addr,
		"card_cache_size", cardCache.Count(),
		"product_cache_size", productCache.Count(),
		"initiative_cache_size", initiativeCache.Count(),
	)

	return runHTTP(ctx, srv)
}

// runHTTP は HTTP server を起動し、ctx キャンセル (シグナル到来) で graceful に停止する。
func runHTTP(ctx context.Context, srv *http.Server) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("http server: %w", err)
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

// setupLogger は ENV に応じてグローバル slog ロガーを初期化する。
func setupLogger(env config.Env) error {
	switch env {
	case config.EnvProd, config.EnvStg:
		slog.SetDefault(slog.New(newCloudLoggingHandler()).With("service", "card"))
	case config.EnvLocal, config.EnvDev:
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
