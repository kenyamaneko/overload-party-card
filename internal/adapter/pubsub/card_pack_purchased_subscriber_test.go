package pubsub

import (
	"context"
	"errors"
	"testing"

	apishop "github.com/kenyamaneko/overload-party-shop/packages/api-shop"
	"github.com/stretchr/testify/assert"
)

// mustMarshalCardPackPurchased は apishop.CardPackPurchasedEvent の型を経由して
// payload を組み立てる (shop が schema を変えたら本テストが compile / 実行で
// 破綻し、乖離を CI で検知する)。EventType は契約固定値で上書きする。
func mustMarshalCardPackPurchased(t *testing.T, ev apishop.CardPackPurchasedEvent) []byte {
	t.Helper()
	ev.EventType = apishop.EventTypeCardPackPurchased
	return mustMarshal(t, ev)
}

func TestCardPackPurchasedSubscriber_Handle(t *testing.T) {
	t.Run("[pubsub]card_pack_purchasedイベントの処理", func(t *testing.T) {
		validPayload := mustMarshalCardPackPurchased(t, apishop.CardPackPurchasedEvent{
			EventID:    "evt-1",
			PlayerID:   "player-1",
			CardPackID: "faction_set_Tuners",
		})

		tests := []struct {
			name          string
			payload       []byte
			repoInsertErr error
			granterErr    error
			wantAck       bool
			wantPacks     []string
		}{
			{
				name:      "新規イベントのとき、pack_idをGrantPackに渡してAckする",
				payload:   validPayload,
				wantAck:   true,
				wantPacks: []string{"faction_set_Tuners"},
			},
			{
				name:          "processed_events INSERT失敗のとき、再配送に乗せるためNackする",
				payload:       validPayload,
				repoInsertErr: errors.New("db down"),
				wantAck:       false,
				wantPacks:     nil,
			},
			{
				name:       "GrantPack失敗のとき、Nackして再配送に乗せる",
				payload:    validPayload,
				granterErr: errors.New("grant failed"),
				wantAck:    false,
				wantPacks:  []string{"faction_set_Tuners"},
			},
			{
				name:      "malformed JSONのとき、NackしてDLQ送りになる",
				payload:   []byte("not-json"),
				wantAck:   false,
				wantPacks: nil,
			},
			{
				name: "未知event_typeのとき、NackしてDLQでpublisherバグを検出する",
				payload: mustMarshal(t, apishop.CardPackPurchasedEvent{
					EventType:  "unknown",
					EventID:    "evt-2",
					PlayerID:   "player-1",
					CardPackID: "faction_set_Tuners",
				}),
				wantAck:   false,
				wantPacks: nil,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				granter := &fakePackGranter{err: tt.granterErr}
				repo := newFakeProcessedEventRepo()
				repo.insertErr = tt.repoInsertErr
				sub := NewCardPackPurchasedSubscriber(granter, repo)

				err := sub.Handle(context.Background(), tt.payload)

				assert.Equal(t, tt.wantAck, err == nil, "ack 判定 (nil=ack, err=%v)", err)
				assert.Equal(t, tt.wantPacks, granter.gotPacks)
			})
		}

		t.Run("重複イベント (processed_events既存)のとき、配布せずAckする", func(t *testing.T) {
			granter := &fakePackGranter{}
			repo := newFakeProcessedEventRepo()
			repo.inserted["evt-1"] = true
			sub := NewCardPackPurchasedSubscriber(granter, repo)

			err := sub.Handle(context.Background(), validPayload)

			assert.NoError(t, err)
			assert.Nil(t, granter.gotPacks)
		})

		t.Run("同じevent_idを持つイベントを二度処理しても、2回目は配布されない", func(t *testing.T) {
			payload := mustMarshalCardPackPurchased(t, apishop.CardPackPurchasedEvent{
				EventID:    "evt-repeat",
				PlayerID:   "player-1",
				CardPackID: "faction_set_Tuners",
			})
			granter := &fakePackGranter{}
			repo := newFakeProcessedEventRepo()
			sub := NewCardPackPurchasedSubscriber(granter, repo)

			firstErr := sub.Handle(context.Background(), payload)
			secondErr := sub.Handle(context.Background(), payload)

			assert.NoError(t, firstErr)
			assert.NoError(t, secondErr)
			assert.Equal(t, []string{"faction_set_Tuners"}, granter.gotPacks, "GrantPack は 1 回だけ呼ばれる")
		})
	})
}
