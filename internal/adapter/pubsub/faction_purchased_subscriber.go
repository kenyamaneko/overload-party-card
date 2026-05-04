// Package pubsub は card サービスの Pub/Sub subscriber を管理します。
//
// faction-purchased-card-sub を購読し、shop でのファクション購入時に
// 対象 faction のカードのみ (Neutral 無し) を GrantInteractor 経由で配布します。
// 初期パック (faction + Neutral) の配布は player-onboarded-card-sub が担当します。
//
// 冪等性は event_id ベースの processed_events で担保します。
// 未知の event_type / malformed payload は Ack ではなく Nack します。
// リトライが無意味なケースは DLQ (faction-purchased-dlq) が拾うため、
// subscriber 側で握りつぶさずインフラに委ねる方針です。
package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	apishop "github.com/kenyamaneko/overload-party-shop/packages/api-shop"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// factionPackGranter は GrantInteractor の GrantFactionPack に依存する
// 最小インターフェースです。テスト時に差し替えるために抽出しています。
type factionPackGranter interface {
	GrantFactionPack(ctx context.Context, playerID, faction string) (int, error)
}

// FactionPurchasedSubscriber は faction-purchased subscription からイベントを取得し、
// shop で購入された faction のカードをプレイヤーへ配布します。
type FactionPurchasedSubscriber struct {
	stream       port.MessageStream
	grantInteractor factionPackGranter
	eventRepo    port.ProcessedEventRepo
}

// NewFactionPurchasedSubscriber は FactionPurchasedSubscriber を生成します。
// subscription 接続は stream に委ねるため、本コンストラクタでは Cloud Pub/Sub SDK に
// 触れない (クリーンアーキテクチャの依存方向遵守)。
func NewFactionPurchasedSubscriber(
	stream port.MessageStream,
	grantInteractor factionPackGranter,
	eventRepo port.ProcessedEventRepo,
) *FactionPurchasedSubscriber {
	return &FactionPurchasedSubscriber{
		stream:       stream,
		grantInteractor: grantInteractor,
		eventRepo:    eventRepo,
	}
}

// Start は ctx がキャンセルされるか stream がエラーを返すまでブロックします。
func (s *FactionPurchasedSubscriber) Start(ctx context.Context) error {
	slog.Info("faction-purchased-card subscriber: consuming")
	return s.stream.Consume(ctx, s.process)
}

// process は 1 イベントを処理する。戻り値 nil = ack、非 nil = nack。
//
// bad payload / unknown event_type / handler 失敗はすべて nack で DLQ に寄せる
// (publisher バグ / スキーマ拡張時の取りこぼしを握りつぶさず人間が気付ける
// 形に残す方針)。
func (s *FactionPurchasedSubscriber) process(ctx context.Context, data []byte) error {
	var ev apishop.FactionPurchasedEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		slog.Error("faction-purchased-card bad payload", "error", err)
		return fmt.Errorf("faction-purchased-card: bad payload: %w", err)
	}
	if ev.EventType != apishop.EventTypeFactionPurchased {
		slog.Warn("faction-purchased-card unexpected event_type",
			"event_id", ev.EventID, "event_type", ev.EventType)
		return fmt.Errorf("faction-purchased-card: unexpected event_type %q", ev.EventType)
	}

	// GrantInteractor は独自の UPSERT を使うため同一 tx に含められない。
	// ここでの dedup は fast-path ガードとして機能する。
	inserted, err := s.eventRepo.Insert(ctx, ev.EventID, ev.EventType)
	if err != nil {
		slog.Error("faction-purchased-card processed_events insert failed",
			"event_id", ev.EventID, "error", err)
		return fmt.Errorf("faction-purchased-card: insert processed_events: %w", err)
	}
	if !inserted {
		return nil
	}

	granted, err := s.grantInteractor.GrantFactionPack(ctx, ev.PlayerID, ev.Faction)
	if err != nil {
		slog.Error("faction-purchased-card grant failed",
			"event_id", ev.EventID,
			"player_id", ev.PlayerID,
			"faction", ev.Faction,
			"error", err,
		)
		return fmt.Errorf("faction-purchased-card: grant: %w", err)
	}

	slog.Info("faction-purchased-card granted",
		"event_id", ev.EventID,
		"player_id", ev.PlayerID,
		"faction", ev.Faction,
		"copies", granted,
	)
	return nil
}
