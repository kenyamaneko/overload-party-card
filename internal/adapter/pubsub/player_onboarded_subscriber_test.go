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
	t.Run("[pubsub]player_onboardedイベントの処理", func(t *testing.T) {
		validPayload := mustMarshalPlayerOnboarded(t, apiscenario.PlayerOnboardedEvent{
			EventID:          "evt-1",
			PlayerID:         "player-1",
			InitialFactionID: "Tuners",
		})

		tests := []struct {
			name             string
			payload          []byte
			repoInsertErr    error
			granterErr       error
			granterErrOnPack string
			wantAck          bool
			wantPacks        []string
		}{
			{
				name:      "新規イベントのとき、basicとfaction_set_<faction> を順次配布してAckする",
				payload:   validPayload,
				wantAck:   true,
				wantPacks: []string{"basic", "faction_set_tuners"},
			},
			{
				name:          "processed_events INSERT失敗のとき、再配送に乗せるためNackする",
				payload:       validPayload,
				repoInsertErr: errors.New("db down"),
				wantAck:       false,
				wantPacks:     nil,
			},
			{
				name:             "1回目 (basic) GrantPack失敗のとき、2回目を呼ばずにNackする (fail-fast)",
				payload:          validPayload,
				granterErr:       errors.New("grant failed"),
				granterErrOnPack: "basic",
				wantAck:          false,
				wantPacks:        []string{"basic"},
			},
			{
				name:             "2回目 (faction) GrantPack失敗のとき、1回目だけ配布された状態でNackする (不完全配布)",
				payload:          validPayload,
				granterErr:       errors.New("grant failed"),
				granterErrOnPack: "faction_set_tuners",
				wantAck:          false,
				wantPacks:        []string{"basic", "faction_set_tuners"},
			},
			{
				name:      "malformed JSONのとき、NackしてDLQ送りになる",
				payload:   []byte("not-json"),
				wantAck:   false,
				wantPacks: nil,
			},
			{
				name: "未知event_typeのとき、NackしてDLQでpublisherバグを検出する",
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
				sub := NewPlayerOnboardedSubscriber(granter, repo)

				err := sub.Handle(context.Background(), tt.payload)

				assert.Equal(t, tt.wantAck, err == nil, "ack 判定 (nil=ack, err=%v)", err)
				assert.Equal(t, tt.wantPacks, granter.gotPacks)
			})
		}

		t.Run("重複イベント (processed_events既存)のとき、配布せずAckする", func(t *testing.T) {
			granter := &fakePackGranter{}
			repo := newFakeProcessedEventRepo()
			repo.inserted["evt-1"] = true
			sub := NewPlayerOnboardedSubscriber(granter, repo)

			err := sub.Handle(context.Background(), validPayload)

			assert.NoError(t, err)
			assert.Nil(t, granter.gotPacks)
		})

		t.Run("同じevent_idを持つイベントを二度処理しても、2回目は配布されない", func(t *testing.T) {
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
