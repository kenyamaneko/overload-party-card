package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	apiscenario "github.com/kenyamaneko/overload-party-scenario/packages/api-scenario"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// initialPackGranter は GrantService の GrantInitialPack に依存する
// 最小インターフェースです。テスト時に差し替えるために抽出しています。
type initialPackGranter interface {
	GrantInitialPack(ctx context.Context, playerID, faction string) (int, error)
}

// PlayerOnboardedSubscriber は player-onboarded subscription からイベントを取得し、
// オンボーディング完了プレイヤーへ初期カードパック
// (initial_faction_id のカード + Neutral カード) を配布します。
//
// 冪等性は event_id ベースの processed_events で担保します。
// 未知の event_type / malformed payload は Ack ではなく Nack して DLQ
// (player-onboarded-dlq) に寄せます。
type PlayerOnboardedSubscriber struct {
	stream       port.MessageStream
	grantService initialPackGranter
	eventRepo    port.ProcessedEventRepo
}

// NewPlayerOnboardedSubscriber は PlayerOnboardedSubscriber を生成します。
func NewPlayerOnboardedSubscriber(
	stream port.MessageStream,
	grantService initialPackGranter,
	eventRepo port.ProcessedEventRepo,
) *PlayerOnboardedSubscriber {
	return &PlayerOnboardedSubscriber{
		stream:       stream,
		grantService: grantService,
		eventRepo:    eventRepo,
	}
}

// Start は ctx がキャンセルされるか stream がエラーを返すまでブロックします。
func (s *PlayerOnboardedSubscriber) Start(ctx context.Context) error {
	slog.Info("player-onboarded-card subscriber: consuming")
	return s.stream.Consume(ctx, s.process)
}

// process は 1 イベントを処理する。戻り値 nil = ack、非 nil = nack。
func (s *PlayerOnboardedSubscriber) process(ctx context.Context, data []byte) error {
	var ev apiscenario.PlayerOnboardedEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		slog.Error("player-onboarded-card bad payload", "error", err)
		return fmt.Errorf("player-onboarded-card: bad payload: %w", err)
	}
	if ev.EventType != apiscenario.EventTypePlayerOnboarded {
		slog.Warn("player-onboarded-card unexpected event_type",
			"event_id", ev.EventID, "event_type", ev.EventType)
		return fmt.Errorf("player-onboarded-card: unexpected event_type %q", ev.EventType)
	}

	// GrantService は独自の UPSERT を使うため同一 tx に含められない。
	// processed_events INSERT は fast-path の重複適用ガードとして機能する。
	inserted, err := s.eventRepo.Insert(ctx, ev.EventID, ev.EventType)
	if err != nil {
		slog.Error("player-onboarded-card processed_events insert failed",
			"event_id", ev.EventID, "error", err)
		return fmt.Errorf("player-onboarded-card: insert processed_events: %w", err)
	}
	if !inserted {
		return nil
	}

	granted, err := s.grantService.GrantInitialPack(ctx, ev.PlayerID, ev.InitialFactionID)
	if err != nil {
		slog.Error("player-onboarded-card grant failed",
			"event_id", ev.EventID,
			"player_id", ev.PlayerID,
			"faction", ev.InitialFactionID,
			"error", err,
		)
		return fmt.Errorf("player-onboarded-card: grant: %w", err)
	}

	slog.Info("player-onboarded-card granted",
		"event_id", ev.EventID,
		"player_id", ev.PlayerID,
		"faction", ev.InitialFactionID,
		"copies", granted,
	)
	return nil
}
