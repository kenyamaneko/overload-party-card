// Package pubsub は card サービスの Pub/Sub subscriber を管理します。
//
// faction-selected-card-sub を購読し、イベントソースに応じて
// 初期パック（選択ファクション + Neutral）またはショップパック
// （選択ファクションのみ）を GrantService 経由で配布します。
// 冪等性は event_id ベースの processed_events で担保します。
package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"cloud.google.com/go/pubsub"

	pubsubevents "github.com/kenyamaneko/overload-party-common/packages/pubsub-events"

	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/service"
)

// FactionSelectedSubscriber は faction-selected-card-sub からイベントを取得し、
// ファクション獲得時にプレイヤーへカードを配布します。
type FactionSelectedSubscriber struct {
	client       *pubsub.Client
	subscription *pubsub.Subscription
	grantService *service.GrantService
	eventRepo    port.ProcessedEventRepo
}

// NewFactionSelectedSubscriber は FactionSelectedSubscriber を生成します。
func NewFactionSelectedSubscriber(
	ctx context.Context,
	projectID, subscriptionID string,
	grantService *service.GrantService,
	eventRepo port.ProcessedEventRepo,
) (*FactionSelectedSubscriber, error) {
	if projectID == "" || subscriptionID == "" {
		return nil, errors.New("faction-selected-card subscriber: projectID and subscriptionID are required")
	}
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("faction-selected-card subscriber: new client: %w", err)
	}
	sub := client.Subscription(subscriptionID)
	ok, err := sub.Exists(ctx)
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("faction-selected-card subscriber: exists check: %w", err)
	}
	if !ok {
		_ = client.Close()
		return nil, fmt.Errorf("faction-selected-card subscriber: subscription %q not found", subscriptionID)
	}
	return &FactionSelectedSubscriber{
		client:       client,
		subscription: sub,
		grantService: grantService,
		eventRepo:    eventRepo,
	}, nil
}

// Start は ctx がキャンセルされるか Receive がエラーを返すまでブロックします。
func (s *FactionSelectedSubscriber) Start(ctx context.Context) error {
	log.Printf("faction-selected-card subscriber: pulling from %s", s.subscription.ID())
	return s.subscription.Receive(ctx, s.handle)
}

// Close は Pub/Sub クライアントを閉じます。
func (s *FactionSelectedSubscriber) Close() error { return s.client.Close() }

func (s *FactionSelectedSubscriber) handle(ctx context.Context, msg *pubsub.Message) {
	var ev pubsubevents.FactionSelectedEvent
	if err := json.Unmarshal(msg.Data, &ev); err != nil {
		log.Printf("faction-selected-card subscriber: bad payload (nack): %v", err)
		msg.Nack()
		return
	}
	if ev.EventType != pubsubevents.EventTypeFactionSelected {
		log.Printf("faction-selected-card subscriber: unknown event_type %q, acking", ev.EventType)
		msg.Ack()
		return
	}

	// GrantService は独自の UPSERT を使うため同一 tx に含められない。
	// ここでの dedup は fast-path ガードとして機能する。
	inserted, err := s.eventRepo.Insert(ctx, ev.EventID, ev.EventType)
	if err != nil {
		log.Printf("faction-selected-card subscriber: processed_events insert failed event=%s: %v", ev.EventID, err)
		msg.Nack()
		return
	}
	if !inserted {
		msg.Ack()
		return
	}

	var granted int
	switch ev.Source {
	case pubsubevents.FactionSourceScenarioInitial:
		granted, err = s.grantService.GrantInitialPack(ctx, ev.PlayerID, ev.Faction)
	case pubsubevents.FactionSourceShopPurchase:
		granted, err = s.grantService.GrantFactionPack(ctx, ev.PlayerID, ev.Faction)
	default:
		log.Printf("faction-selected-card subscriber: unknown source %q for event=%s, acking", ev.Source, ev.EventID)
		msg.Ack()
		return
	}

	if err != nil {
		log.Printf("faction-selected-card subscriber: grant failed event=%s player=%s faction=%s source=%s: %v",
			ev.EventID, ev.PlayerID, ev.Faction, ev.Source, err)
		msg.Nack()
		return
	}

	log.Printf("faction-selected-card subscriber: granted %d copies event=%s player=%s faction=%s",
		granted, ev.EventID, ev.PlayerID, ev.Faction)
	msg.Ack()
}
