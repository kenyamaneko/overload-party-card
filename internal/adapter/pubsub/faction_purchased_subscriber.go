// Package pubsub は card サービスの Pub/Sub subscriber を管理します。
//
// faction-purchased-card-sub を購読し、shop でのファクション購入時に
// 対象 faction のカードのみ (Neutral 無し) を GrantService 経由で配布します。
// 初期パック (faction + Neutral) の配布は player-onboarded-card-sub が担当します
// (ADR-022)。
//
// 冪等性は event_id ベースの processed_events で担保します。
// 未知の event_type / malformed payload は Ack ではなく Nack します。
// リトライが無意味なケースは DLQ (faction-purchased-dlq) が拾うため、
// subscriber 側で握りつぶさずインフラに委ねる方針です。
package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"cloud.google.com/go/pubsub/v2"

	apishop "github.com/kenyamaneko/overload-party-shop/packages/api-shop"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// factionPackGranter は GrantService の GrantFactionPack に依存する
// 最小インターフェースです。テスト時に差し替えるために抽出しています。
type factionPackGranter interface {
	GrantFactionPack(ctx context.Context, playerID, faction string) (int, error)
}

// FactionPurchasedSubscriber は faction-purchased-card-sub からイベントを取得し、
// shop で購入された faction のカードをプレイヤーへ配布します。
type FactionPurchasedSubscriber struct {
	client       *pubsub.Client
	subscriber   *pubsub.Subscriber
	grantService factionPackGranter
	eventRepo    port.ProcessedEventRepo
}

// NewFactionPurchasedSubscriber は FactionPurchasedSubscriber を生成します。
func NewFactionPurchasedSubscriber(
	ctx context.Context,
	projectID, subscriptionID string,
	grantService factionPackGranter,
	eventRepo port.ProcessedEventRepo,
) (*FactionPurchasedSubscriber, error) {
	if projectID == "" || subscriptionID == "" {
		return nil, errors.New("faction-purchased-card subscriber: projectID and subscriptionID are required")
	}
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("faction-purchased-card subscriber: new client: %w", err)
	}
	return &FactionPurchasedSubscriber{
		client:       client,
		subscriber:   client.Subscriber(subscriptionID),
		grantService: grantService,
		eventRepo:    eventRepo,
	}, nil
}

// Start は ctx がキャンセルされるか Receive がエラーを返すまでブロックします。
func (s *FactionPurchasedSubscriber) Start(ctx context.Context) error {
	slog.Info("faction-purchased-card subscriber starting", "subscription", s.subscriber.ID())
	return s.subscriber.Receive(ctx, s.handle)
}

// Close は Pub/Sub クライアントを閉じます。
func (s *FactionPurchasedSubscriber) Close() error { return s.client.Close() }

func (s *FactionPurchasedSubscriber) handle(ctx context.Context, msg *pubsub.Message) {
	if s.process(ctx, msg.Data) {
		msg.Ack()
		return
	}
	msg.Nack()
}

// process は payload を読み取り、副作用を実行した結果 Ack してよいか (true) を返します。
// msg.Ack / msg.Nack から分離することで、subscriber 層の仕様をテーブル駆動でテストできます。
func (s *FactionPurchasedSubscriber) process(ctx context.Context, data []byte) bool {
	var ev apishop.FactionPurchasedEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		slog.Error("faction-purchased-card bad payload", "error", err)
		return false
	}
	if ev.EventType != apishop.EventTypeFactionPurchased {
		// 期待しない event_type はパブリッシャーのバグ。Nack → DLQ で回収し、
		// 握りつぶさずに人間が気付ける形に残す。
		slog.Warn("faction-purchased-card unexpected event_type",
			"event_id", ev.EventID, "event_type", ev.EventType)
		return false
	}

	// GrantService は独自の UPSERT を使うため同一 tx に含められない。
	// ここでの dedup は fast-path ガードとして機能する。
	inserted, err := s.eventRepo.Insert(ctx, ev.EventID, ev.EventType)
	if err != nil {
		slog.Error("faction-purchased-card processed_events insert failed",
			"event_id", ev.EventID, "error", err)
		return false
	}
	if !inserted {
		return true
	}

	granted, err := s.grantService.GrantFactionPack(ctx, ev.PlayerID, ev.Faction)
	if err != nil {
		slog.Error("faction-purchased-card grant failed",
			"event_id", ev.EventID,
			"player_id", ev.PlayerID,
			"faction", ev.Faction,
			"error", err,
		)
		return false
	}

	slog.Info("faction-purchased-card granted",
		"event_id", ev.EventID,
		"player_id", ev.PlayerID,
		"faction", ev.Faction,
		"copies", granted,
	)
	return true
}
