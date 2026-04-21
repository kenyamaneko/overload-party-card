// Package pubsub は card サービスの Pub/Sub subscriber を管理します。
//
// faction-selected-card-sub を購読し、イベントソースに応じて
// 初期パック（選択ファクション + Neutral）またはショップパック
// （選択ファクションのみ）を GrantService 経由で配布します。
// 冪等性は event_id ベースの processed_events で担保します。
//
// 未知の event_type / source / malformed payload は Ack ではなく Nack します。
// リトライが無意味なケースは DLQ (faction-selected-dlq) が拾うため、
// subscriber 側で握りつぶさずインフラに委ねる方針です。
package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"cloud.google.com/go/pubsub/v2"

	pubsubevents "github.com/kenyamaneko/overload-party-common/packages/pubsub-events"

	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/service"
)

// FactionSelectedSubscriber は faction-selected-card-sub からイベントを取得し、
// ファクション獲得時にプレイヤーへカードを配布します。
type FactionSelectedSubscriber struct {
	client       *pubsub.Client
	subscriber   *pubsub.Subscriber
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
	return &FactionSelectedSubscriber{
		client:       client,
		subscriber:   client.Subscriber(subscriptionID),
		grantService: grantService,
		eventRepo:    eventRepo,
	}, nil
}

// Start は ctx がキャンセルされるか Receive がエラーを返すまでブロックします。
func (s *FactionSelectedSubscriber) Start(ctx context.Context) error {
	slog.Info("faction-selected-card subscriber starting", "subscription", s.subscriber.ID())
	return s.subscriber.Receive(ctx, s.handle)
}

// Close は Pub/Sub クライアントを閉じます。
func (s *FactionSelectedSubscriber) Close() error { return s.client.Close() }

func (s *FactionSelectedSubscriber) handle(ctx context.Context, msg *pubsub.Message) {
	var ev pubsubevents.FactionSelectedEvent
	if err := json.Unmarshal(msg.Data, &ev); err != nil {
		slog.Error("faction-selected-card bad payload", "error", err)
		msg.Nack()
		return
	}
	if ev.EventType != pubsubevents.EventTypeFactionSelected {
		// 期待しない event_type はパブリッシャーのバグ。Nack → DLQ で回収し、
		// 握りつぶさずに人間が気付ける形に残す。
		slog.Warn("faction-selected-card unexpected event_type",
			"event_id", ev.EventID, "event_type", ev.EventType)
		msg.Nack()
		return
	}

	// GrantService は独自の UPSERT を使うため同一 tx に含められない。
	// ここでの dedup は fast-path ガードとして機能する。
	inserted, err := s.eventRepo.Insert(ctx, ev.EventID, ev.EventType)
	if err != nil {
		slog.Error("faction-selected-card processed_events insert failed",
			"event_id", ev.EventID, "error", err)
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
		// 未知 source もパブリッシャー拡張時の取りこぼし。Nack → DLQ で検出する。
		slog.Warn("faction-selected-card unknown source",
			"event_id", ev.EventID, "source", ev.Source)
		msg.Nack()
		return
	}

	if err != nil {
		slog.Error("faction-selected-card grant failed",
			"event_id", ev.EventID,
			"player_id", ev.PlayerID,
			"faction", ev.Faction,
			"source", ev.Source,
			"error", err,
		)
		msg.Nack()
		return
	}

	slog.Info("faction-selected-card granted",
		"event_id", ev.EventID,
		"player_id", ev.PlayerID,
		"faction", ev.Faction,
		"copies", granted,
	)
	msg.Ack()
}
