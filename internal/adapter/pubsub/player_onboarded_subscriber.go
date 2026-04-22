package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"cloud.google.com/go/pubsub/v2"

	apiscenario "github.com/kenyamaneko/overload-party-scenario/packages/api-scenario"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// initialPackGranter は GrantService の GrantInitialPack に依存する
// 最小インターフェースです。テスト時に差し替えるために抽出しています。
type initialPackGranter interface {
	GrantInitialPack(ctx context.Context, playerID, faction string) (int, error)
}

// PlayerOnboardedSubscriber は player-onboarded-card-sub からイベントを取得し、
// オンボーディング完了プレイヤーへ初期カードパック
// (initial_faction_id のカード + Neutral カード) を配布します (ADR-022)。
//
// 冪等性は event_id ベースの processed_events で担保します。
// 未知の event_type / malformed payload は Ack ではなく Nack して DLQ
// (player-onboarded-dlq) に寄せます。
type PlayerOnboardedSubscriber struct {
	client       *pubsub.Client
	subscriber   *pubsub.Subscriber
	grantService initialPackGranter
	eventRepo    port.ProcessedEventRepo
}

// NewPlayerOnboardedSubscriber は PlayerOnboardedSubscriber を生成します。
func NewPlayerOnboardedSubscriber(
	ctx context.Context,
	projectID, subscriptionID string,
	grantService initialPackGranter,
	eventRepo port.ProcessedEventRepo,
) (*PlayerOnboardedSubscriber, error) {
	if projectID == "" || subscriptionID == "" {
		return nil, errors.New("player-onboarded-card subscriber: projectID and subscriptionID are required")
	}
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("player-onboarded-card subscriber: new client: %w", err)
	}
	return &PlayerOnboardedSubscriber{
		client:       client,
		subscriber:   client.Subscriber(subscriptionID),
		grantService: grantService,
		eventRepo:    eventRepo,
	}, nil
}

// Start は ctx がキャンセルされるか Receive がエラーを返すまでブロックします。
func (s *PlayerOnboardedSubscriber) Start(ctx context.Context) error {
	slog.Info("player-onboarded-card subscriber starting", "subscription", s.subscriber.ID())
	return s.subscriber.Receive(ctx, s.handle)
}

// Close は Pub/Sub クライアントを閉じます。
func (s *PlayerOnboardedSubscriber) Close() error { return s.client.Close() }

func (s *PlayerOnboardedSubscriber) handle(ctx context.Context, msg *pubsub.Message) {
	if s.process(ctx, msg.Data) {
		msg.Ack()
		return
	}
	msg.Nack()
}

// process は payload を読み取り、副作用を実行した結果 Ack してよいか (true) を返します。
// msg.Ack / msg.Nack から分離することで、subscriber 層の仕様をテーブル駆動でテストできます。
func (s *PlayerOnboardedSubscriber) process(ctx context.Context, data []byte) bool {
	var ev apiscenario.PlayerOnboardedEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		slog.Error("player-onboarded-card bad payload", "error", err)
		return false
	}
	if ev.EventType != apiscenario.EventTypePlayerOnboarded {
		// publisher バグ・スキーマ拡張時の取りこぼしは握りつぶさず DLQ に寄せる。
		slog.Warn("player-onboarded-card unexpected event_type",
			"event_id", ev.EventID, "event_type", ev.EventType)
		return false
	}

	// GrantService は独自の UPSERT を使うため同一 tx に含められない。
	// processed_events INSERT は fast-path の重複適用ガードとして機能する。
	inserted, err := s.eventRepo.Insert(ctx, ev.EventID, ev.EventType)
	if err != nil {
		slog.Error("player-onboarded-card processed_events insert failed",
			"event_id", ev.EventID, "error", err)
		return false
	}
	if !inserted {
		return true
	}

	granted, err := s.grantService.GrantInitialPack(ctx, ev.PlayerID, ev.InitialFactionID)
	if err != nil {
		slog.Error("player-onboarded-card grant failed",
			"event_id", ev.EventID,
			"player_id", ev.PlayerID,
			"faction", ev.InitialFactionID,
			"error", err,
		)
		return false
	}

	slog.Info("player-onboarded-card granted",
		"event_id", ev.EventID,
		"player_id", ev.PlayerID,
		"faction", ev.InitialFactionID,
		"copies", granted,
	)
	return true
}
