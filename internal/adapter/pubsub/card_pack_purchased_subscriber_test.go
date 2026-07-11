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

func TestCardPackPurchasedSubscriber_Start(t *testing.T) {
	t.Run("card_pack_purchased イベントの購読", func(t *testing.T) {
		// 契約検証は apishopfake 経由で shop 側の publish 型をそのまま使う
		// (shop が schema を変えたら card のテストが compile / 実行で破綻し、乖離を CI で検知する)。
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
				name:             "新規イベントのとき、pack_id を GrantPack に渡して Ack する",
				publish:          publishValid,
				repoInsertResult: true,
				wantAck:          true,
				wantPacks:        []string{"faction_set_Tuners"},
			},
			{
				name:             "重複イベント (processed_events 既存) のとき、配布せず Ack する",
				publish:          publishValid,
				repoInsertResult: false,
				wantAck:          true,
				wantPacks:        nil,
			},
			{
				// processed_events への INSERT (dedup ガード) が失敗した場合、event を
				// Ack で捨てると配布も dedup 記録もされずメッセージが失われる。Nack して
				// Pub/Sub の at-least-once 再配送に乗せる。
				name:          "processed_events INSERT 失敗のとき、再配送に乗せるため Nack する",
				publish:       publishValid,
				repoInsertErr: errors.New("db down"),
				wantAck:       false,
				wantPacks:     nil,
			},
			{
				name:             "GrantPack 失敗のとき、Nack して再配送に乗せる",
				publish:          publishValid,
				repoInsertResult: true,
				granterErr:       errors.New("grant failed"),
				wantAck:          false,
				wantPacks:        []string{"faction_set_Tuners"},
			},
			{
				name: "malformed JSON のとき、Nack して DLQ 送りになる",
				publish: func(_ context.Context, _ *apishopfake.Publisher, broker *apishopfake.Broker) {
					broker.Publish(topicCardPackPurchased, []byte("not-json"))
				},
				wantAck:   false,
				wantPacks: nil,
			},
			{
				name: "未知 event_type のとき、Nack して DLQ で publisher バグを検出する",
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
	})
}
