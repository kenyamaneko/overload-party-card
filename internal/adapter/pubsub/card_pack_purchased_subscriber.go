package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	apishop "github.com/kenyamaneko/overload-party-shop/packages/api-shop"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// CardPackPurchasedSubscriber は card-pack-purchased-card-sub の push 配信を
// 処理し、購入された card_pack をプレイヤーに配布する。配布 SSoT は card.card_pack マスター。
type CardPackPurchasedSubscriber struct {
	grantInteractor packGranter
	eventRepo       port.ProcessedEventRepo
}

// NewCardPackPurchasedSubscriber は CardPackPurchasedSubscriber を生成する。
func NewCardPackPurchasedSubscriber(
	grantInteractor packGranter,
	eventRepo port.ProcessedEventRepo,
) *CardPackPurchasedSubscriber {
	return &CardPackPurchasedSubscriber{
		grantInteractor: grantInteractor,
		eventRepo:       eventRepo,
	}
}

// Handle は push delivery で届いた 1 イベントの payload (デコード済み bytes) を
// 処理する。戻り値 nil = ack、非 nil = nack として呼び出し元 (push handler) が
// 応答ステータスに変換する。port.MessageHandler を満たす。
func (s *CardPackPurchasedSubscriber) Handle(ctx context.Context, data []byte) error {
	var event apishop.CardPackPurchasedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("card-pack-purchased-card bad payload", "error", err)
		return fmt.Errorf("card-pack-purchased-card: bad payload: %w", err)
	}
	if event.EventType != apishop.EventTypeCardPackPurchased {
		slog.Warn("card-pack-purchased-card unexpected event_type",
			"event_id", event.EventID, "event_type", event.EventType)
		return fmt.Errorf("card-pack-purchased-card: unexpected event_type %q", event.EventType)
	}

	inserted, err := s.eventRepo.Insert(ctx, event.EventID, event.EventType)
	if err != nil {
		slog.Error("card-pack-purchased-card processed_events insert failed",
			"event_id", event.EventID, "error", err)
		return fmt.Errorf("card-pack-purchased-card: insert processed_events: %w", err)
	}
	if !inserted {
		return nil
	}

	granted, err := s.grantInteractor.GrantPack(ctx, event.PlayerID, event.CardPackID)
	if err != nil {
		slog.Error("card-pack-purchased-card grant failed",
			"event_id", event.EventID,
			"player_id", event.PlayerID,
			"pack_id", event.CardPackID,
			"error", err,
		)
		return fmt.Errorf("card-pack-purchased-card: grant %q: %w", event.CardPackID, err)
	}

	slog.Info("card-pack-purchased-card granted",
		"event_id", event.EventID,
		"player_id", event.PlayerID,
		"pack_id", event.CardPackID,
		"copies", granted,
	)
	return nil
}
