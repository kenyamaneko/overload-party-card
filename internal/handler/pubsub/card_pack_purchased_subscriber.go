package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	apishop "github.com/kenyamaneko/overload-party-shop/packages/api-shop"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"
)

// CardPackPurchasedSubscriber はカードパック購入イベントを受け取り、購入されたカードパックを配布します。
type CardPackPurchasedSubscriber struct {
	grantInteractor *usecase.GrantInteractor
}

// NewCardPackPurchasedSubscriber は CardPackPurchasedSubscriber を生成します。
func NewCardPackPurchasedSubscriber(grantInteractor *usecase.GrantInteractor) *CardPackPurchasedSubscriber {
	return &CardPackPurchasedSubscriber{grantInteractor: grantInteractor}
}

// GrantPurchasedCards はカードパック購入イベントの内容を解釈し、購入されたカードパックを配布します。
// 内容を解釈できないとき、イベント種別が想定と異なるとき、配布に失敗したときはエラーを返します。
func (s *CardPackPurchasedSubscriber) GrantPurchasedCards(ctx context.Context, payload []byte) error {
	var event apishop.CardPackPurchasedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("card-pack-purchased-card: %w: %w", ErrMalformedPayload, err)
	}
	if event.EventType != apishop.EventTypeCardPackPurchased {
		return fmt.Errorf("card-pack-purchased-card: %w %q", ErrUnexpectedEventType, event.EventType)
	}

	copies, err := s.grantInteractor.GrantPurchasedPack(ctx, domain.CardPackPurchasedEvent{
		EventID:    event.EventID,
		EventType:  event.EventType,
		PlayerID:   event.PlayerID,
		CardPackID: event.CardPackID,
	})
	if err != nil {
		return fmt.Errorf("card-pack-purchased-card: grant pack %q for event %q: %w", event.CardPackID, event.EventID, err)
	}

	slog.Info("card-pack-purchased-card processed",
		"event_id", event.EventID,
		"player_id", event.PlayerID,
		"pack_id", event.CardPackID,
		"copies", copies,
	)
	return nil
}
