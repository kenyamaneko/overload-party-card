package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	apiscenario "github.com/kenyamaneko/overload-party-scenario/packages/api-scenario"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// packGranter は GrantInteractor の GrantPack に依存する
// 最小インターフェースです。テスト時に差し替えるために抽出しています。
type packGranter interface {
	GrantPack(ctx context.Context, playerID, packID string) (int, error)
}

// PlayerOnboardedSubscriber は player-onboarded-card-sub の push 配信を処理し、
// オンボーディング完了プレイヤーへ basic pack と選択 faction の基本セットを配布します。
//
// 冪等性は event_id ベースの processed_events で担保します。
// 未知の event_type / malformed payload は ack ではなく nack 相当のエラーを返し、
// DLQ (player-onboarded-dlq) に寄せます。
type PlayerOnboardedSubscriber struct {
	grantInteractor packGranter
	eventRepo       port.ProcessedEventRepo
}

// NewPlayerOnboardedSubscriber は PlayerOnboardedSubscriber を生成します。
func NewPlayerOnboardedSubscriber(
	grantInteractor packGranter,
	eventRepo port.ProcessedEventRepo,
) *PlayerOnboardedSubscriber {
	return &PlayerOnboardedSubscriber{
		grantInteractor: grantInteractor,
		eventRepo:       eventRepo,
	}
}

// Handle は push delivery で届いた 1 イベントの payload (デコード済み bytes) を
// 処理します。戻り値 nil = ack、非 nil = nack として呼び出し元 (push handler) が
// 応答ステータスに変換します。port.MessageHandler を満たします。
func (s *PlayerOnboardedSubscriber) Handle(ctx context.Context, data []byte) error {
	var event apiscenario.PlayerOnboardedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("player-onboarded-card bad payload", "error", err)
		return fmt.Errorf("player-onboarded-card: bad payload: %w", err)
	}
	if event.EventType != apiscenario.EventTypePlayerOnboarded {
		slog.Warn("player-onboarded-card unexpected event_type",
			"event_id", event.EventID, "event_type", event.EventType)
		return fmt.Errorf("player-onboarded-card: unexpected event_type %q", event.EventType)
	}

	inserted, err := s.eventRepo.Insert(ctx, event.EventID, event.EventType)
	if err != nil {
		slog.Error("player-onboarded-card processed_events insert failed",
			"event_id", event.EventID, "error", err)
		return fmt.Errorf("player-onboarded-card: insert processed_events: %w", err)
	}
	if !inserted {
		return nil
	}

	// pack_id は card_packs.yaml で faction を小文字化した形 (faction_set_she 等) で定義される。
	factionPackID := "faction_set_" + strings.ToLower(event.InitialFactionID)
	totalGranted := 0
	// basic pack と faction pack を順に GrantPack するため、間でクラッシュすると
	// 「片方だけ配布済み」の状態が残りうる (稼働前のため許容)。
	for _, packID := range []string{"basic", factionPackID} {
		granted, err := s.grantInteractor.GrantPack(ctx, event.PlayerID, packID)
		if err != nil {
			slog.Error("player-onboarded-card grant failed",
				"event_id", event.EventID,
				"player_id", event.PlayerID,
				"pack_id", packID,
				"error", err,
			)
			return fmt.Errorf("player-onboarded-card: grant %q: %w", packID, err)
		}
		totalGranted += granted
	}

	slog.Info("player-onboarded-card granted",
		"event_id", event.EventID,
		"player_id", event.PlayerID,
		"faction", event.InitialFactionID,
		"copies", totalGranted,
	)
	return nil
}
