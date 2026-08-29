package pubsub_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	apishop "github.com/kenyamaneko/overload-party-shop/packages/api-shop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/adapter/pubsub"
)

func newCardPackPurchasedPayload(t *testing.T, eventID, playerID, cardPackID string) []byte {
	t.Helper()
	data, err := json.Marshal(apishop.CardPackPurchasedEvent{
		EventType:  apishop.EventTypeCardPackPurchased,
		EventID:    eventID,
		Timestamp:  time.Now(),
		PlayerID:   playerID,
		CardPackID: cardPackID,
	})
	require.NoError(t, err)
	return data
}

func TestCardPackPurchasedSubscriberHandle(t *testing.T) {
	t.Run("[pubsub] card-pack-purchased-card-subのイベント処理", func(t *testing.T) {
		t.Run("payloadがJSONとして解釈できないとき、エラーになる", func(t *testing.T) {
			sub := pubsub.NewCardPackPurchasedSubscriber(&fakePackGranter{}, &fakeProcessedEventRepo{})

			err := sub.Handle(context.Background(), []byte("not json"))

			assert.ErrorContains(t, err, "bad payload")
		})

		t.Run("デコードしたevent_typeが処理対象のイベント種別と一致しないとき、エラーになる", func(t *testing.T) {
			sub := pubsub.NewCardPackPurchasedSubscriber(&fakePackGranter{}, &fakeProcessedEventRepo{})
			payload := []byte(`{"event_type":"unrelated_event","event_id":"evt-1","player_id":"TST-0001","card_pack_id":"TST-0002"}`)

			err := sub.Handle(context.Background(), payload)

			assert.ErrorContains(t, err, "unexpected event_type")
		})

		t.Run("処理済みイベントの記録への保存が失敗したとき、エラーになる", func(t *testing.T) {
			sub := pubsub.NewCardPackPurchasedSubscriber(&fakePackGranter{}, &fakeProcessedEventRepo{err: errors.New("insert failed")})
			payload := newCardPackPurchasedPayload(t, "evt-1", "TST-0001", "TST-0002")

			err := sub.Handle(context.Background(), payload)

			assert.ErrorContains(t, err, "insert processed_events")
		})

		t.Run("当該event_idが既に処理済みとして記録されているとき、パックの配布は要求されず、正常終了になる", func(t *testing.T) {
			granter := &fakePackGranter{}
			sub := pubsub.NewCardPackPurchasedSubscriber(granter, &fakeProcessedEventRepo{inserted: false})
			payload := newCardPackPurchasedPayload(t, "evt-1", "TST-0001", "TST-0002")

			err := sub.Handle(context.Background(), payload)

			assert.NoError(t, err)
			assert.Empty(t, granter.calls)
		})

		t.Run("配布処理が失敗したとき、エラーになる", func(t *testing.T) {
			granter := &fakePackGranter{errFor: map[string]error{"TST-0002": errors.New("grant failed")}}
			sub := pubsub.NewCardPackPurchasedSubscriber(granter, &fakeProcessedEventRepo{inserted: true})
			payload := newCardPackPurchasedPayload(t, "evt-1", "TST-0001", "TST-0002")

			err := sub.Handle(context.Background(), payload)

			assert.ErrorContains(t, err, `grant "TST-0002"`)
		})

		t.Run("payloadが未処理のevent_id・player_id・card_pack_idを含む正常な内容のとき、そのplayer_idに対しcard_pack_idのパックの配布が要求され、正常終了になる", func(t *testing.T) {
			granter := &fakePackGranter{copiesFor: map[string]int{"TST-0002": 3}}
			sub := pubsub.NewCardPackPurchasedSubscriber(granter, &fakeProcessedEventRepo{inserted: true})
			payload := newCardPackPurchasedPayload(t, "evt-1", "TST-0001", "TST-0002")

			err := sub.Handle(context.Background(), payload)

			assert.NoError(t, err)
			assert.Equal(t, []grantCall{{playerID: "TST-0001", packID: "TST-0002"}}, granter.calls)
		})
	})
}
