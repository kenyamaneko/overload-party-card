package pubsub_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	apiscenario "github.com/kenyamaneko/overload-party-scenario/packages/api-scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/adapter/pubsub"
)

func newPlayerOnboardedPayload(t *testing.T, eventID, playerID, initialFactionID string) []byte {
	t.Helper()
	data, err := json.Marshal(apiscenario.PlayerOnboardedEvent{
		EventType:        apiscenario.EventTypePlayerOnboarded,
		EventID:          eventID,
		Timestamp:        time.Now(),
		PlayerID:         playerID,
		InitialFactionID: initialFactionID,
	})
	require.NoError(t, err)
	return data
}

func TestPlayerOnboardedSubscriberHandle(t *testing.T) {
	t.Run("[pubsub] player-onboarded-card-subのイベント処理", func(t *testing.T) {
		t.Run("payloadがJSONとして解釈できないとき、エラーになる", func(t *testing.T) {
			sub := pubsub.NewPlayerOnboardedSubscriber(&fakePackGranter{}, &fakeProcessedEventRepo{})

			err := sub.Handle(context.Background(), []byte("not json"))

			assert.ErrorContains(t, err, "bad payload")
		})

		t.Run("デコードしたevent_typeが処理対象のイベント種別と一致しないとき、エラーになる", func(t *testing.T) {
			sub := pubsub.NewPlayerOnboardedSubscriber(&fakePackGranter{}, &fakeProcessedEventRepo{})
			payload := []byte(`{"event_type":"unrelated_event","event_id":"evt-1","player_id":"TST-0001","initial_faction_id":"SHE"}`)

			err := sub.Handle(context.Background(), payload)

			assert.ErrorContains(t, err, "unexpected event_type")
		})

		t.Run("処理済みイベントの記録への保存が失敗したとき、エラーになる", func(t *testing.T) {
			sub := pubsub.NewPlayerOnboardedSubscriber(&fakePackGranter{}, &fakeProcessedEventRepo{err: errors.New("insert failed")})
			payload := newPlayerOnboardedPayload(t, "evt-1", "TST-0001", "SHE")

			err := sub.Handle(context.Background(), payload)

			assert.ErrorContains(t, err, "insert processed_events")
		})

		t.Run("当該event_idが既に処理済みとして記録されているとき、パックの配布は要求されず、正常終了になる", func(t *testing.T) {
			granter := &fakePackGranter{}
			sub := pubsub.NewPlayerOnboardedSubscriber(granter, &fakeProcessedEventRepo{inserted: false})
			payload := newPlayerOnboardedPayload(t, "evt-1", "TST-0001", "SHE")

			err := sub.Handle(context.Background(), payload)

			assert.NoError(t, err)
			assert.Empty(t, granter.calls)
		})

		t.Run("basicパックの配布に失敗したとき、faction_set_*パックの配布は要求されず、エラーになる", func(t *testing.T) {
			granter := &fakePackGranter{errFor: map[string]error{"basic": errors.New("grant failed")}}
			sub := pubsub.NewPlayerOnboardedSubscriber(granter, &fakeProcessedEventRepo{inserted: true})
			payload := newPlayerOnboardedPayload(t, "evt-1", "TST-0001", "SHE")

			err := sub.Handle(context.Background(), payload)

			assert.ErrorContains(t, err, `grant "basic"`)
			assert.Equal(t, []grantCall{{playerID: "TST-0001", packID: "basic"}}, granter.calls)
		})

		t.Run("payloadが未処理のevent_id・player_id・initial_faction_id SHEを含む正常な内容のとき、basicパックとfaction_set_sheパックの配布が要求され、正常終了になる", func(t *testing.T) {
			granter := &fakePackGranter{copiesFor: map[string]int{"basic": 3, "faction_set_she": 5}}
			sub := pubsub.NewPlayerOnboardedSubscriber(granter, &fakeProcessedEventRepo{inserted: true})
			payload := newPlayerOnboardedPayload(t, "evt-1", "TST-0001", "SHE")

			err := sub.Handle(context.Background(), payload)

			assert.NoError(t, err)
			assert.Equal(t, []grantCall{
				{playerID: "TST-0001", packID: "basic"},
				{playerID: "TST-0001", packID: "faction_set_she"},
			}, granter.calls)
		})
	})
}
