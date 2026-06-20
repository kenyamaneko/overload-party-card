package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	apishop "github.com/kenyamaneko/overload-party-shop/packages/api-shop"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// CardPackPurchasedSubscriber は card-pack-purchased subscription を消費し、
// 購入された card_pack をプレイヤーに配布する。配布 SSoT は card.card_pack マスター。
type CardPackPurchasedSubscriber struct {
	stream          port.MessageStream
	grantInteractor packGranter
	eventRepo       port.ProcessedEventRepo
}

// NewCardPackPurchasedSubscriber は CardPackPurchasedSubscriber を生成する。
func NewCardPackPurchasedSubscriber(
	stream port.MessageStream,
	grantInteractor packGranter,
	eventRepo port.ProcessedEventRepo,
) *CardPackPurchasedSubscriber {
	return &CardPackPurchasedSubscriber{
		stream:          stream,
		grantInteractor: grantInteractor,
		eventRepo:       eventRepo,
	}
}

// Start は ctx がキャンセルされるか stream がエラーを返すまでブロックする。
func (s *CardPackPurchasedSubscriber) Start(ctx context.Context) error {
	slog.Info("card-pack-purchased-card subscriber: consuming")
	return s.stream.Consume(ctx, s.process)
}

// process は 1 イベントを処理する。戻り値 nil = ack、非 nil = nack。
func (s *CardPackPurchasedSubscriber) process(ctx context.Context, data []byte) error {
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
