package pubsub

import (
	"context"
	"errors"
	"testing"
	"time"

	apishop "github.com/kenyamaneko/overload-party-shop/packages/api-shop"
	"github.com/kenyamaneko/overload-party-shop/packages/api-shop/apishopfake"
	"github.com/stretchr/testify/assert"
)

// topicCardPackPurchased は apishopfake.PublishCardPackPurchased が内部で
// publish する topic 名と一致させる必要がある (raw bytes を投げる test ケースで利用)。
const topicCardPackPurchased = "card-pack-purchased"

// TestCardPackPurchasedSubscriber_Start は「shop 購入起因の card_pack 配布」を
// Start() → stream.Consume → process の経路で固定する。
//
// 契約検証は apishopfake 経由で shop 側の publish 型をそのまま使う
// (shop が schema を変えたら card のテストが compile / 実行で破綻するように
// 設計し、乖離を CI で検知する)。
func TestCardPackPurchasedSubscriber_Start(t *testing.T) {
	publishValid := func(ctx context.Context, pub *apishopfake.Publisher, _ *apishopfake.Broker) {
		_ = apishopfake.PublishCardPackPurchased(ctx, pub, apishop.CardPackPurchasedEvent{
			PlayerID:   "player-1",
			CardPackID: "faction_set_Tuners",
		})
	}

	tests := []struct {
		name             string
		publish          func(ctx context.Context, pub *apishopfake.Publisher, broker *apishopfake.Broker)
		repoInsertResult bool
		repoInsertErr    error
		granterErr       error
		wantAck          bool
		wantPacks        []string
	}{
		{
			name:             "新規イベントは pack_id を GrantPack に渡して Ack する",
			publish:          publishValid,
			repoInsertResult: true,
			wantAck:          true,
			wantPacks:        []string{"faction_set_Tuners"},
		},
		{
			name:             "重複イベント (processed_events 既存): 配布せず Ack",
			publish:          publishValid,
			repoInsertResult: false,
			wantAck:          true,
			wantPacks:        nil,
		},
		{
			// processed_events への INSERT (dedup ガード) が失敗した場合、event を
			// Ack で捨てると配布も dedup 記録もされずメッセージが失われる。Nack して
			// Pub/Sub の at-least-once 再配送に乗せる。
			name:          "processed_events INSERT 失敗時は Pub/Sub の再配送に乗せるため Nack する",
			publish:       publishValid,
			repoInsertErr: errors.New("db down"),
			wantAck:       false,
			wantPacks:     nil,
		},
		{
			name:             "GrantPack 失敗時は Nack して再配送に乗せる",
			publish:          publishValid,
			repoInsertResult: true,
			granterErr:       errors.New("grant failed"),
			wantAck:          false,
			wantPacks:        []string{"faction_set_Tuners"},
		},
		{
			name: "malformed JSON: Nack して DLQ 送り",
			publish: func(_ context.Context, _ *apishopfake.Publisher, broker *apishopfake.Broker) {
				broker.Publish(topicCardPackPurchased, []byte("not-json"))
			},
			wantAck:   false,
			wantPacks: nil,
		},
		{
			name: "未知 event_type: Nack して DLQ で publisher バグ検出",
			publish: func(_ context.Context, _ *apishopfake.Publisher, broker *apishopfake.Broker) {
				broker.Publish(topicCardPackPurchased, mustMarshal(t, apishop.CardPackPurchasedEvent{
					EventType:  "unknown",
					EventID:    "evt-2",
					PlayerID:   "player-1",
					CardPackID: "faction_set_Tuners",
				}))
			},
			wantAck:   false,
			wantPacks: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker := apishopfake.NewBroker()
			pub := apishopfake.NewPublisher(broker)
			stream := apishopfake.NewStream(apishopfake.NewSubscriber(broker), topicCardPackPurchased)

			granter := &fakePackGranter{err: tt.granterErr}
			repo := &fakeProcessedEventRepo{
				insertResult: tt.repoInsertResult,
				insertErr:    tt.repoInsertErr,
			}
			sub := NewCardPackPurchasedSubscriber(stream, granter, repo)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			started := make(chan struct{})
			go func() {
				close(started)
				_ = sub.Start(ctx)
			}()
			<-started

			tt.publish(ctx, pub, broker)

			handlerErr := stream.ExpectHandled(t, time.Second)
			assert.Equal(t, tt.wantAck, handlerErr == nil, "ack 判定 (nil=ack, err=%v)", handlerErr)
			assert.Equal(t, tt.wantPacks, granter.gotPacks)
		})
	}
}
