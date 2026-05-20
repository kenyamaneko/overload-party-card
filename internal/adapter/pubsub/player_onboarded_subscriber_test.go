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

// topicPlayerOnboarded は apiscenariofake が PublishPlayerOnboarded /
// ExpectPlayerOnboarded で内部的にハードコードしているルーティングキー。
// raw bytes を broker.Publish するケース (不正 JSON / 未知 event_type) と
// NewStream の subscribe topic を一致させる必要がある。
const topicPlayerOnboarded = "player-onboarded"

// fakePackGranter は packGranter のテスト用スタブです。
// errOnPack に一致する pack_id だけエラーを返す形にすると、N 回呼びの中で
// 何回目に失敗したかを直接表現できます。
type fakePackGranter struct {
	errOnPack string
	err       error
	gotPacks  []string
}

func (f *fakePackGranter) GrantPack(_ context.Context, _, packID string) (int, error) {
	f.gotPacks = append(f.gotPacks, packID)
	if f.err != nil && (f.errOnPack == "" || f.errOnPack == packID) {
		return 0, f.err
	}
	return 6, nil
}

// TestPlayerOnboardedSubscriber_Start は「オンボーディング完了時に basic pack
// と選択 faction の基本セットを順次配布する」仕様を Start() → stream.Consume
// → process の経路で固定する。
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
		granterErrOnPack string
		wantAck          bool
		wantPacks        []string
	}{
		{
			name:             "新規イベントは basic と faction_set_<faction> を順次配布して Ack する",
			publish:          publishValid,
			repoInsertResult: true,
			wantAck:          true,
			wantPacks:        []string{"basic", "faction_set_Tuners"},
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
			// Pub/Sub の at-least-once 再配送に乗せ、次回成功時に dedup と配布が走る
			// ことを期待する仕様の固定。
			name:          "processed_events INSERT 失敗時は Pub/Sub の再配送に乗せるため Nack する",
			publish:       publishValid,
			repoInsertErr: errors.New("db down"),
			wantAck:       false,
			wantPacks:     nil,
		},
		{
			// basic → faction_set の順次配布は fail-fast: 1 回目失敗時点で 2 回目を
			// 呼ばずに Nack する。1 回目失敗後にそのまま続行すると DB 状態によって
			// 挙動が予測不能になるため、early exit を仕様として固定する。
			name:             "1 回目 (basic) GrantPack 失敗時は 2 回目を呼ばずに Nack する (fail-fast)",
			publish:          publishValid,
			repoInsertResult: true,
			granterErr:       errors.New("grant failed"),
			granterErrOnPack: "basic",
			wantAck:          false,
			wantPacks:        []string{"basic"},
		},
		{
			// 2 回目 (faction) で失敗した時点で 1 回目 (basic) は配布完了済み = 不完全
			// 配布が発生する。このとき Nack で再配送されても processed_events が
			// 既に書かれていれば dedup で skip され、不完全配布のまま固定される。
			// この at-most-once 相当の挙動 (docs/ARCHITECTURE.md「Pub/Sub subscriber
			// の冪等性」) を仕様として固定する。
			name:             "2 回目 (faction) GrantPack 失敗時は 1 回目だけ配布された状態で Nack する (不完全配布)",
			publish:          publishValid,
			repoInsertResult: true,
			granterErr:       errors.New("grant failed"),
			granterErrOnPack: "faction_set_Tuners",
			wantAck:          false,
			wantPacks:        []string{"basic", "faction_set_Tuners"},
		},
		{
			name: "malformed JSON: Nack して DLQ 送り",
			publish: func(_ context.Context, _ *apiscenariofake.Publisher, broker *apiscenariofake.Broker) {
				broker.Publish(topicPlayerOnboarded, []byte("not-json"))
			},
			wantAck:   false,
			wantPacks: nil,
		},
		{
			name: "未知 event_type: Nack して DLQ で publisher バグ検出",
			publish: func(_ context.Context, _ *apiscenariofake.Publisher, broker *apiscenariofake.Broker) {
				broker.Publish(topicPlayerOnboarded, mustMarshal(t, apiscenario.PlayerOnboardedEvent{
					EventType:        "unknown",
					EventID:          "evt-2",
					PlayerID:         "player-1",
					InitialFactionID: "SHE",
				}))
			},
			wantAck:   false,
			wantPacks: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker := apiscenariofake.NewBroker()
			pub := apiscenariofake.NewPublisher(broker)
			stream := apiscenariofake.NewStream(apiscenariofake.NewSubscriber(broker), topicPlayerOnboarded)

			granter := &fakePackGranter{err: tt.granterErr, errOnPack: tt.granterErrOnPack}
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
			assert.Equal(t, tt.wantPacks, granter.gotPacks)
		})
	}
}
