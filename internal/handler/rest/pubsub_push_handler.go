package rest

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/handler/pubsub"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

var (
	errMalformedEnvelope   = errors.New(malformedEnvelopeMessage)
	errMalformedBase64Data = errors.New(malformedBase64DataMessage)
)

// PubSubPushHandler は Cloud Pub/Sub push subscription からの配信を受け取り、対応する購読へ渡します。
type PubSubPushHandler struct {
	playerOnboardedSubscriber   *pubsub.PlayerOnboardedSubscriber
	cardPackPurchasedSubscriber *pubsub.CardPackPurchasedSubscriber
}

// NewPubSubPushHandler は PubSubPushHandler を生成します。
func NewPubSubPushHandler(
	playerOnboardedSubscriber *pubsub.PlayerOnboardedSubscriber,
	cardPackPurchasedSubscriber *pubsub.CardPackPurchasedSubscriber,
) *PubSubPushHandler {
	return &PubSubPushHandler{
		playerOnboardedSubscriber:   playerOnboardedSubscriber,
		cardPackPurchasedSubscriber: cardPackPurchasedSubscriber,
	}
}

// ReceivePlayerOnboarded は player-onboarded-card-sub の push 配信を受け取ります。
func (h *PubSubPushHandler) ReceivePlayerOnboarded(c *gin.Context) {
	receivePush(c, h.playerOnboardedSubscriber.GrantInitialCards)
}

// ReceiveCardPackPurchased は card-pack-purchased-card-sub の push 配信を受け取ります。
func (h *PubSubPushHandler) ReceiveCardPackPurchased(c *gin.Context) {
	receivePush(c, h.cardPackPurchasedSubscriber.GrantPurchasedCards)
}

// receivePush は push 配信のメッセージを購読に渡し、成否を応答にします。
func receivePush(c *gin.Context, subscribe func(ctx context.Context, payload []byte) error) {
	payload, err := readPushedPayload(c)
	if err != nil {
		respondBadRequest(c, err.Error())
		return
	}
	if err := subscribe(c.Request.Context(), payload); err != nil {
		respondInternalError(c, err)
		return
	}
	c.Status(http.StatusOK)
}

// readPushedPayload は push 配信の本文から、イベントの内容を取り出します。
// message・data の欠落と空の data は同じ不正な本文として扱います。
func readPushedPayload(c *gin.Context) ([]byte, error) {
	var envelope apicard.PubSubPushEnvelope
	if err := c.ShouldBindJSON(&envelope); err != nil {
		var corruptInputError base64.CorruptInputError
		if errors.As(err, &corruptInputError) {
			return nil, errMalformedBase64Data
		}
		return nil, errMalformedEnvelope
	}
	if len(envelope.Message.Data) == 0 {
		return nil, errMalformedEnvelope
	}
	return envelope.Message.Data, nil
}
