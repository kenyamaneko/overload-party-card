package rest_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPubSubPushHandler(t *testing.T) {
	t.Run("[handler] Pub/Sub push受信", func(t *testing.T) {
		malformedEnvelopeCases := []struct {
			name string
			body string
		}{
			{name: "messageフィールドが欠落しているとき", body: `{}`},
			{name: "message.dataフィールドが欠落しているとき", body: `{"message":{}}`},
			{name: "message.dataが空文字のとき", body: `{"message":{"data":""}}`},
		}
		for _, tc := range malformedEnvelopeCases {
			t.Run(tc.name+"、400になりボディのerrorフィールドはpubsub push: malformed envelopeになる", func(t *testing.T) {
				engine := newTestRouter(t, withPlayerOnboardedHandler(failIfCalled(t)))

				rr := doRequest(t, engine, http.MethodPost, "/internal/v1/pubsub/player-onboarded", tc.body)

				require.Equal(t, http.StatusBadRequest, rr.Code)
				assert.Equal(t, "pubsub push: malformed envelope", decodeErrorMessage(t, rr))
			})
		}

		t.Run("message.dataがbase64として解釈できない文字列のとき、400になりボディのerrorフィールドはpubsub push: malformed base64 dataになる", func(t *testing.T) {
			engine := newTestRouter(t, withPlayerOnboardedHandler(failIfCalled(t)))

			rr := doRequest(t, engine, http.MethodPost, "/internal/v1/pubsub/player-onboarded", `{"message":{"data":"not valid base64!!"}}`)

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Equal(t, "pubsub push: malformed base64 data", decodeErrorMessage(t, rr))
		})

		t.Run("message.dataがbase64として解釈でき、注入したport.MessageHandlerがエラーを返すとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			engine := newTestRouter(t, withPlayerOnboardedHandler(func(ctx context.Context, data []byte) error {
				return errors.New("handler failed")
			}))

			rr := doRequest(t, engine, http.MethodPost, "/internal/v1/pubsub/player-onboarded", pubsubEnvelopeBody(t, "payload"))

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("message.dataがbase64として解釈でき、注入したport.MessageHandlerがエラーを返さないとき、200になりレスポンスボディは空になる", func(t *testing.T) {
			engine := newTestRouter(t, withPlayerOnboardedHandler(func(ctx context.Context, data []byte) error {
				return nil
			}))

			rr := doRequest(t, engine, http.MethodPost, "/internal/v1/pubsub/player-onboarded", pubsubEnvelopeBody(t, "payload"))

			require.Equal(t, http.StatusOK, rr.Code)
			assert.Empty(t, rr.Body.String())
		})

		t.Run("port.MessageHandlerが呼び出されるとき、渡される引数はデコード後のmessage.dataの内容と一致する", func(t *testing.T) {
			var received []byte
			engine := newTestRouter(t, withPlayerOnboardedHandler(func(ctx context.Context, data []byte) error {
				received = data
				return nil
			}))

			rr := doRequest(t, engine, http.MethodPost, "/internal/v1/pubsub/player-onboarded", pubsubEnvelopeBody(t, "decoded-payload"))

			require.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "decoded-payload", string(received))
		})

		t.Run("player-onboarded用とcard-pack-purchased用に別々のport.MessageHandlerを登録したとき、それぞれのエンドポイントは対応するハンドラだけを呼び出す", func(t *testing.T) {
			var onboardedCalled, purchasedCalled bool
			engine := newTestRouter(t,
				withPlayerOnboardedHandler(func(ctx context.Context, data []byte) error {
					onboardedCalled = true
					return nil
				}),
				withCardPackPurchasedHandler(func(ctx context.Context, data []byte) error {
					purchasedCalled = true
					return nil
				}),
			)

			rrOnboarded := doRequest(t, engine, http.MethodPost, "/internal/v1/pubsub/player-onboarded", pubsubEnvelopeBody(t, "onboarded"))
			require.Equal(t, http.StatusOK, rrOnboarded.Code)
			assert.True(t, onboardedCalled, "player-onboarded用ハンドラが呼ばれる")
			assert.False(t, purchasedCalled, "card-pack-purchased用ハンドラは呼ばれない")

			onboardedCalled, purchasedCalled = false, false
			rrPurchased := doRequest(t, engine, http.MethodPost, "/internal/v1/pubsub/card-pack-purchased", pubsubEnvelopeBody(t, "purchased"))
			require.Equal(t, http.StatusOK, rrPurchased.Code)
			assert.False(t, onboardedCalled, "player-onboarded用ハンドラは呼ばれない")
			assert.True(t, purchasedCalled, "card-pack-purchased用ハンドラが呼ばれる")
		})
	})
}

// failIfCalled returns a port.MessageHandler that fails the test if invoked, for cases that
// must be rejected before the handler is reached.
func failIfCalled(t *testing.T) func(ctx context.Context, data []byte) error {
	t.Helper()
	return func(ctx context.Context, data []byte) error {
		t.Fatal("message handler must not be called")
		return nil
	}
}

// pubsubEnvelopeBody builds a valid Cloud Pub/Sub push envelope JSON body carrying payload,
// base64-encoded as the wire format requires.
func pubsubEnvelopeBody(t *testing.T, payload string) string {
	t.Helper()
	envelope := struct {
		Message struct {
			Data string `json:"data"`
		} `json:"message"`
	}{}
	envelope.Message.Data = base64.StdEncoding.EncodeToString([]byte(payload))
	encoded, err := json.Marshal(envelope)
	require.NoError(t, err)
	return string(encoded)
}
