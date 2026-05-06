package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	apiscenario "github.com/kenyamaneko/overload-party-scenario/packages/api-scenario"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// packGranter は GrantInteractor の GrantPack に依存する
// 最小インターフェースです。テスト時に差し替えるために抽出しています。
type packGranter interface {
	GrantPack(ctx context.Context, playerID, packID string) (int, error)
}

// PlayerOnboardedSubscriber は player-onboarded subscription からイベントを取得し、
// オンボーディング完了プレイヤーへ basic pack と選択 faction の基本セットを配布します。
//
// 冪等性は event_id ベースの processed_events で担保します。
// 未知の event_type / malformed payload は Ack ではなく Nack して DLQ
// (player-onboarded-dlq) に寄せます。
type PlayerOnboardedSubscriber struct {
	stream          port.MessageStream
	grantInteractor packGranter
	eventRepo       port.ProcessedEventRepo
}

// NewPlayerOnboardedSubscriber は PlayerOnboardedSubscriber を生成します。
func NewPlayerOnboardedSubscriber(
	stream port.MessageStream,
	grantInteractor packGranter,
	eventRepo port.ProcessedEventRepo,
) *PlayerOnboardedSubscriber {
	return &PlayerOnboardedSubscriber{
		stream:          stream,
		grantInteractor: grantInteractor,
		eventRepo:       eventRepo,
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

	inserted, err := s.eventRepo.Insert(ctx, ev.EventID, ev.EventType)
	if err != nil {
		slog.Error("player-onboarded-card processed_events insert failed",
			"event_id", ev.EventID, "error", err)
		return fmt.Errorf("player-onboarded-card: insert processed_events: %w", err)
	}
	if !inserted {
		return nil
	}

	factionPackID := "faction_set_" + ev.InitialFactionID
	totalGranted := 0
	for _, packID := range []string{"basic", factionPackID} {
		granted, err := s.grantInteractor.GrantPack(ctx, ev.PlayerID, packID)
		if err != nil {
			slog.Error("player-onboarded-card grant failed",
				"event_id", ev.EventID,
				"player_id", ev.PlayerID,
				"pack_id", packID,
				"error", err,
			)
			return fmt.Errorf("player-onboarded-card: grant %q: %w", packID, err)
		}
		totalGranted += granted
	}

	slog.Info("player-onboarded-card granted",
		"event_id", ev.EventID,
		"player_id", ev.PlayerID,
		"faction", ev.InitialFactionID,
		"copies", totalGranted,
	)
	return nil
}
