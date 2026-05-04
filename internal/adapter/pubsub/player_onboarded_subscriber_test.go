package pubsub

import (
	"context"
	"errors"
	"testing"
	"time"

	apiscenario "github.com/kenyamaneko/overload-party-scenario/packages/api-scenario"
	"github.com/kenyamaneko/overload-party-scenario/packages/api-scenario/apiscenariofake"
	"github.com/stretchr/testify/assert"
)

// fakeInitialGranter は initialPackGranter のテスト用スタブです。
type fakeInitialGranter struct {
	err          error
	calls        int
	lastPlayerID string
	lastFaction  string
}

func (f *fakeInitialGranter) GrantInitialPack(_ context.Context, playerID, faction string) (int, error) {
	f.calls++
	f.lastPlayerID = playerID
	f.lastFaction = faction
	if f.err != nil {
		return 0, f.err
	}
	return 6, nil
}

// TestPlayerOnboardedSubscriber_Start は「オンボーディング完了時に初期パック
// (initial_faction_id + Neutral) を 1 イベント単位で冪等に配布する」仕様を
// Start() → stream.Consume → process の経路で固定する。
//
// 契約検証は apiscenariofake 経由で scenario 側の publish 型をそのまま使う
// (scenario が schema を変えたら card のテストが compile / 実行で破綻するように
// 設計し、乖離を CI で検知する)。
func TestPlayerOnboardedSubscriber_Start(t *testing.T) {
	publishValid := func(ctx context.Context, pub *apiscenariofake.Publisher, _ *apiscenariofake.Broker) {
		_ = apiscenariofake.PublishPlayerOnboarded(ctx, pub, apiscenario.PlayerOnboardedEvent{
			PlayerID:         "player-1",
			InitialFactionID: "Tuners",
		})
	}

	tests := []struct {
		name             string
		publish          func(ctx context.Context, pub *apiscenariofake.Publisher, broker *apiscenariofake.Broker)
		repoInsertResult bool
		repoInsertErr    error
		granterErr       error
		wantAck          bool
		assertGranter    func(t *testing.T, g *fakeInitialGranter)
	}{
		{
			name:             "新規イベントは初期パックを配布して Ack する",
			publish:          publishValid,
			repoInsertResult: true,
			wantAck:          true,
			assertGranter: func(t *testing.T, g *fakeInitialGranter) {
				assert.Equal(t, 1, g.calls)
				assert.Equal(t, "player-1", g.lastPlayerID)
				assert.Equal(t, "Tuners", g.lastFaction, "initial_faction_id がそのまま GrantInitialPack に渡る")
			},
		},
		{
			name:             "重複イベント (processed_events 既存): 配布せず Ack",
			publish:          publishValid,
			repoInsertResult: false,
			wantAck:          true,
			assertGranter: func(t *testing.T, g *fakeInitialGranter) {
				assert.Equal(t, 0, g.calls, "冪等ガードにより granter 未呼び出し")
			},
		},
		{
			name:          "processed_events insert 失敗: Nack でリトライ",
			publish:       publishValid,
			repoInsertErr: errors.New("db down"),
			wantAck:       false,
			assertGranter: func(t *testing.T, g *fakeInitialGranter) {
				assert.Equal(t, 0, g.calls, "dedup 失敗時は granter まで到達しない")
			},
		},
		{
			name:             "GrantInitialPack 失敗: Nack でリトライ",
			publish:          publishValid,
			repoInsertResult: true,
			granterErr:       errors.New("grant failed"),
			wantAck:          false,
			assertGranter: func(t *testing.T, g *fakeInitialGranter) {
				assert.Equal(t, 1, g.calls)
			},
		},
		{
			name: "malformed JSON: Nack して DLQ 送り",
			publish: func(_ context.Context, _ *apiscenariofake.Publisher, broker *apiscenariofake.Broker) {
				broker.Publish(apiscenario.TopicPlayerOnboarded, []byte("not-json"))
			},
			wantAck: false,
			assertGranter: func(t *testing.T, g *fakeInitialGranter) {
				assert.Equal(t, 0, g.calls)
			},
		},
		{
			name: "未知 event_type: Nack して DLQ で publisher バグ検出",
			publish: func(_ context.Context, _ *apiscenariofake.Publisher, broker *apiscenariofake.Broker) {
				broker.Publish(apiscenario.TopicPlayerOnboarded, mustMarshal(t, apiscenario.PlayerOnboardedEvent{
					EventType:        "unknown",
					EventID:          "evt-2",
					PlayerID:         "player-1",
					InitialFactionID: "SHE",
				}))
			},
			wantAck: false,
			assertGranter: func(t *testing.T, g *fakeInitialGranter) {
				assert.Equal(t, 0, g.calls, "event_type フィルタで granter に到達しない")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker := apiscenariofake.NewBroker()
			pub := apiscenariofake.NewPublisher(broker)
			stream := apiscenariofake.NewStream(apiscenariofake.NewSubscriber(broker), apiscenario.TopicPlayerOnboarded)

			granter := &fakeInitialGranter{err: tt.granterErr}
			repo := &fakeProcessedEventRepo{
				insertResult: tt.repoInsertResult,
				insertErr:    tt.repoInsertErr,
			}
			sub := NewPlayerOnboardedSubscriber(stream, granter, repo)

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
