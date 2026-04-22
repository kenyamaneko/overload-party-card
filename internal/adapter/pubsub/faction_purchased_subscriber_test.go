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

// fakeFactionGranter は factionPackGranter のテスト用スタブです。
type fakeFactionGranter struct {
	err          error
	calls        int
	lastPlayerID string
	lastFaction  string
}

func (f *fakeFactionGranter) GrantFactionPack(_ context.Context, playerID, faction string) (int, error) {
	f.calls++
	f.lastPlayerID = playerID
	f.lastFaction = faction
	if f.err != nil {
		return 0, f.err
	}
	return 3, nil
}

// TestFactionPurchasedSubscriber_Consumes は「shop 購入起因の faction パック配布
// (Neutral 含まず) を 1 イベント単位で冪等に処理する」仕様を Start() →
// stream.Consume → process の経路で固定する。
//
// 契約検証は apishopfake 経由で shop 側の publish 型をそのまま使う
// (shop が schema を変えたら card のテストが compile / 実行で破綻するように
// 設計し、乖離を CI で検知する)。
func TestFactionPurchasedSubscriber_Consumes(t *testing.T) {
	publishValid := func(ctx context.Context, pub *apishopfake.Publisher, _ *apishopfake.Broker) {
		_ = apishopfake.PublishFactionPurchased(ctx, pub, apishop.FactionPurchasedEvent{
			PlayerID: "player-1",
			Faction:  "Tenki",
		})
	}

	tests := []struct {
		name             string
		publish          func(ctx context.Context, pub *apishopfake.Publisher, broker *apishopfake.Broker)
		repoInsertResult bool
		repoInsertErr    error
		granterErr       error
		wantAck          bool
		assertGranter    func(t *testing.T, g *fakeFactionGranter)
	}{
		{
			name:             "新規イベントは faction のみ配布して Ack する",
			publish:          publishValid,
			repoInsertResult: true,
			wantAck:          true,
			assertGranter: func(t *testing.T, g *fakeFactionGranter) {
				assert.Equal(t, 1, g.calls)
				assert.Equal(t, "player-1", g.lastPlayerID)
				assert.Equal(t, "Tenki", g.lastFaction)
			},
		},
		{
			name:             "重複イベント (processed_events 既存): 配布せず Ack",
			publish:          publishValid,
			repoInsertResult: false,
			wantAck:          true,
			assertGranter: func(t *testing.T, g *fakeFactionGranter) {
				assert.Equal(t, 0, g.calls, "冪等ガードにより granter 未呼び出し")
			},
		},
		{
			name:          "processed_events insert 失敗: Nack でリトライ",
			publish:       publishValid,
			repoInsertErr: errors.New("db down"),
			wantAck:       false,
			assertGranter: func(t *testing.T, g *fakeFactionGranter) {
				assert.Equal(t, 0, g.calls, "dedup 失敗時は granter まで到達しない")
			},
		},
		{
			name:             "GrantFactionPack 失敗: Nack でリトライ",
			publish:          publishValid,
			repoInsertResult: true,
			granterErr:       errors.New("grant failed"),
			wantAck:          false,
			assertGranter: func(t *testing.T, g *fakeFactionGranter) {
				assert.Equal(t, 1, g.calls)
			},
		},
		{
			name: "malformed JSON: Nack して DLQ 送り",
			publish: func(_ context.Context, _ *apishopfake.Publisher, broker *apishopfake.Broker) {
				broker.Publish(apishop.TopicFactionPurchased, []byte("not-json"))
			},
			wantAck: false,
			assertGranter: func(t *testing.T, g *fakeFactionGranter) {
				assert.Equal(t, 0, g.calls)
			},
		},
		{
			name: "未知 event_type: Nack して DLQ で publisher バグ検出",
			publish: func(_ context.Context, _ *apishopfake.Publisher, broker *apishopfake.Broker) {
				broker.Publish(apishop.TopicFactionPurchased, mustMarshal(t, apishop.FactionPurchasedEvent{
					EventType: "unknown",
					EventID:   "evt-2",
					PlayerID:  "player-1",
					Faction:   "SHE",
				}))
			},
			wantAck: false,
			assertGranter: func(t *testing.T, g *fakeFactionGranter) {
				assert.Equal(t, 0, g.calls, "event_type フィルタで granter に到達しない")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker := apishopfake.NewBroker()
			pub := apishopfake.NewPublisher(broker)
			stream := apishopfake.NewStream(apishopfake.NewSubscriber(broker), apishop.TopicFactionPurchased)

			granter := &fakeFactionGranter{err: tt.granterErr}
			repo := &fakeProcessedEventRepo{
				insertResult: tt.repoInsertResult,
				insertErr:    tt.repoInsertErr,
			}
			sub := NewFactionPurchasedSubscriber(stream, granter, repo)

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

			tt.assertGranter(t, granter)
		})
	}
}
