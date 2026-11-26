package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	apiscenario "github.com/kenyamaneko/overload-party-scenario/packages/api-scenario"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"
)

// PlayerOnboardedSubscriber はプレイヤー登録完了イベントを受け取り、初期カードを配布します。
type PlayerOnboardedSubscriber struct {
	grantInteractor *usecase.GrantInteractor
}

// NewPlayerOnboardedSubscriber は PlayerOnboardedSubscriber を生成します。
func NewPlayerOnboardedSubscriber(grantInteractor *usecase.GrantInteractor) *PlayerOnboardedSubscriber {
	return &PlayerOnboardedSubscriber{grantInteractor: grantInteractor}
}

// GrantInitialCards はプレイヤー登録完了イベントの内容を解釈し、初期カードを配布します。
// 内容を解釈できないとき、イベント種別が想定と異なるとき、配布に失敗したときはエラーを返します。
func (s *PlayerOnboardedSubscriber) GrantInitialCards(ctx context.Context, payload []byte) error {
	var event apiscenario.PlayerOnboardedEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("player-onboarded-card: %w: %w", ErrMalformedPayload, err)
	}
	if event.EventType != apiscenario.EventTypePlayerOnboarded {
		return fmt.Errorf("player-onboarded-card: %w %q", ErrUnexpectedEventType, event.EventType)
	}

	copies, err := s.grantInteractor.GrantInitialPacks(ctx, domain.PlayerOnboardedEvent{
		EventID:          event.EventID,
		EventType:        event.EventType,
		PlayerID:         event.PlayerID,
		InitialFactionID: event.InitialFactionID,
	})
	if err != nil {
		return fmt.Errorf("player-onboarded-card: grant initial packs for event %q: %w", event.EventID, err)
	}

	slog.Info("player-onboarded-card processed",
		"event_id", event.EventID,
		"player_id", event.PlayerID,
		"faction", event.InitialFactionID,
		"copies", copies,
	)
	return nil
}
