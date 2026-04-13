package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
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
		log.Fatalf("card: %v", err)
	}
}

func run() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("pgxpool new: %w", err)
	}
	defer pool.Close()

	fsClient, err := firestore.NewClient(ctx, cfg.FirestoreProjectID)
	if err != nil {
		return fmt.Errorf("firestore new client: %w", err)
	}
	defer func() { _ = fsClient.Close() }()

	cardRepo := repository.NewPgCardRepository(pool)
	playerCardRepo := repository.NewPgPlayerCardRepository(pool)
	deckRepo := repository.NewPgDeckRepository(pool)
	eventRepo := repository.NewPgProcessedEventRepository(pool)

	// game_config は現在 card の runtime パスから参照していない。
	// クライアント到達性は起動時に検証するため、repo を生成だけしておく。
	_ = repository.NewFirestoreGameConfigRepository(fsClient)

	cardCache := cache.NewCardCache()
	if err := cardCache.Load(ctx, cardRepo); err != nil {
		return fmt.Errorf("load card cache: %w", err)
	}

	cardSvc := service.NewCardService(cardRepo, playerCardRepo)
	deckSvc := service.NewDeckService(deckRepo, playerCardRepo, cardCache)
	grantSvc := service.NewGrantService(cardRepo, playerCardRepo)

	cardH := rest.NewCardHandler(cardSvc)
	deckH := rest.NewDeckHandler(deckSvc)
	playerCardH := rest.NewPlayerCardHandler(deckSvc)
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
			log.Fatalf("faction-selected subscriber error: %v", err)
		}
	}()

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("card: listening on %s (cache=%d cards, pubsub project=%s)",
			srv.Addr, cardCache.Count(), cfg.PubsubProjectID)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Printf("card: shutdown requested")
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
