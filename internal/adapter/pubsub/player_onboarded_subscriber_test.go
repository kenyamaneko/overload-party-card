package pubsub

import (
	"context"
	"errors"
	"testing"

	apiscenario "github.com/kenyamaneko/overload-party-scenario/packages/api-scenario"
	"github.com/stretchr/testify/assert"
)

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

// mustMarshalPlayerOnboarded は apiscenario.PlayerOnboardedEvent の型を経由して
// payload を組み立てる (scenario が schema を変えたら本テストが compile / 実行で
// 破綻し、乖離を CI で検知する)。EventType は契約固定値で上書きする。
func mustMarshalPlayerOnboarded(t *testing.T, ev apiscenario.PlayerOnboardedEvent) []byte {
	t.Helper()
	ev.EventType = apiscenario.EventTypePlayerOnboarded
	return mustMarshal(t, ev)
}

func TestPlayerOnboardedSubscriber_Handle(t *testing.T) {
	t.Run("player_onboarded イベントの処理", func(t *testing.T) {
		validPayload := mustMarshalPlayerOnboarded(t, apiscenario.PlayerOnboardedEvent{
			EventID:          "evt-1",
			PlayerID:         "player-1",
			InitialFactionID: "Tuners",
		})

		tests := []struct {
			name             string
			payload          []byte
			seedInserted     string
			repoInsertErr    error
			granterErr       error
			granterErrOnPack string
			wantAck          bool
			wantPacks        []string
		}{
			{
				name:      "新規イベントのとき、basic と faction_set_<faction> を順次配布して Ack する",
				payload:   validPayload,
				wantAck:   true,
				wantPacks: []string{"basic", "faction_set_tuners"},
			},
			{
				name:         "重複イベント (processed_events 既存) のとき、配布せず Ack する",
				payload:      validPayload,
				seedInserted: "evt-1",
				wantAck:      true,
				wantPacks:    nil,
			},
			{
				// processed_events への INSERT (dedup ガード) が失敗した場合、event を
				// Ack で捨てると配布も dedup 記録もされずメッセージが失われる。Nack して
				// Pub/Sub の at-least-once 再配送に乗せ、次回成功時に dedup と配布が走る
				// ことを期待する仕様の固定。
				name:          "processed_events INSERT 失敗のとき、再配送に乗せるため Nack する",
				payload:       validPayload,
				repoInsertErr: errors.New("db down"),
				wantAck:       false,
				wantPacks:     nil,
			},
			{
				// basic → faction_set の順次配布は fail-fast: 1 回目失敗時点で 2 回目を
				// 呼ばずに Nack する。1 回目失敗後にそのまま続行すると DB 状態によって
				// 挙動が予測不能になるため、early exit を仕様として固定する。
				name:             "1 回目 (basic) GrantPack 失敗のとき、2 回目を呼ばずに Nack する (fail-fast)",
				payload:          validPayload,
				granterErr:       errors.New("grant failed"),
				granterErrOnPack: "basic",
				wantAck:          false,
				wantPacks:        []string{"basic"},
			},
			{
				// 2 回目 (faction) で失敗した時点で 1 回目 (basic) は配布完了済み = 不完全
				// 配布が発生する。このとき Nack で再配送されても processed_events が
				// 既に書かれていれば dedup で skip され、不完全配布のまま固定される。
				// この at-most-once 相当の挙動を仕様として固定する。
				name:             "2 回目 (faction) GrantPack 失敗のとき、1 回目だけ配布された状態で Nack する (不完全配布)",
				payload:          validPayload,
				granterErr:       errors.New("grant failed"),
				granterErrOnPack: "faction_set_tuners",
				wantAck:          false,
				wantPacks:        []string{"basic", "faction_set_tuners"},
			},
			{
				name:      "malformed JSON のとき、Nack して DLQ 送りになる",
				payload:   []byte("not-json"),
				wantAck:   false,
				wantPacks: nil,
			},
			{
				name: "未知 event_type のとき、Nack して DLQ で publisher バグを検出する",
				payload: mustMarshal(t, apiscenario.PlayerOnboardedEvent{
					EventType:        "unknown",
					EventID:          "evt-2",
					PlayerID:         "player-1",
					InitialFactionID: "SHE",
				}),
				wantAck:   false,
				wantPacks: nil,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				granter := &fakePackGranter{err: tt.granterErr, errOnPack: tt.granterErrOnPack}
				repo := newFakeProcessedEventRepo()
				repo.insertErr = tt.repoInsertErr
				if tt.seedInserted != "" {
					repo.inserted[tt.seedInserted] = true
				}
				sub := NewPlayerOnboardedSubscriber(granter, repo)

				err := sub.Handle(context.Background(), tt.payload)

				assert.Equal(t, tt.wantAck, err == nil, "ack 判定 (nil=ack, err=%v)", err)
				assert.Equal(t, tt.wantPacks, granter.gotPacks)
			})
		}

		t.Run("同じ event_id を持つイベントを二度処理しても、2 回目は配布されない", func(t *testing.T) {
			payload := mustMarshalPlayerOnboarded(t, apiscenario.PlayerOnboardedEvent{
				EventID:          "evt-repeat",
				PlayerID:         "player-1",
				InitialFactionID: "Tuners",
			})
			granter := &fakePackGranter{}
			repo := newFakeProcessedEventRepo()
			sub := NewPlayerOnboardedSubscriber(granter, repo)

			firstErr := sub.Handle(context.Background(), payload)
			secondErr := sub.Handle(context.Background(), payload)

			assert.NoError(t, firstErr)
			assert.NoError(t, secondErr)
			assert.Equal(t, []string{"basic", "faction_set_tuners"}, granter.gotPacks, "GrantPack は 1 巡だけ呼ばれる")
		})
	})
}
