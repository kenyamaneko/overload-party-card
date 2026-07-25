package rest

import (
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// pubsubPushEnvelope は Cloud Pub/Sub push subscription が送るリクエスト本文。
type pubsubPushEnvelope struct {
	Message struct {
		Data string `json:"data"`
	} `json:"message"`
}

// PubSubPushHandler は Cloud Pub/Sub push subscription からの配信を受け取り、
// デコードした payload を対応する port.MessageHandler へ渡します。
type PubSubPushHandler struct {
	playerOnboarded   port.MessageHandler
	cardPackPurchased port.MessageHandler
}

// NewPubSubPushHandler は PubSubPushHandler を生成します。
func NewPubSubPushHandler(playerOnboarded, cardPackPurchased port.MessageHandler) *PubSubPushHandler {
	return &PubSubPushHandler{
		playerOnboarded:   playerOnboarded,
		cardPackPurchased: cardPackPurchased,
	}
}

// HandlePlayerOnboarded は player-onboarded-card-sub の push 配信を受け取ります。
func (h *PubSubPushHandler) HandlePlayerOnboarded(c *gin.Context) {
	handlePubSubPush(c, h.playerOnboarded)
}

// HandleCardPackPurchased は card-pack-purchased-card-sub の push 配信を受け取ります。
func (h *PubSubPushHandler) HandleCardPackPurchased(c *gin.Context) {
	handlePubSubPush(c, h.cardPackPurchased)
}

// handlePubSubPush は push envelope をデコードして handler に渡し、結果を応答へ
// 変換する。envelope 自体を読めない (JSON / base64 が壊れている) 場合は 400 を、
// handler がエラーを返した (ペイロード不正・副作用失敗) 場合は 500 を返す。
// Cloud Pub/Sub は 2xx を ack、それ以外を nack として再配信するため、この 2 系統は
// 応答コードとしては同じ再配信対象になるが、原因を応答本文で区別できるようにする。
func handlePubSubPush(c *gin.Context, handler port.MessageHandler) {
	var envelope pubsubPushEnvelope
	if err := c.ShouldBindJSON(&envelope); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pubsub push: malformed envelope: " + err.Error()})
		return
	}

	data, err := base64.StdEncoding.DecodeString(envelope.Message.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pubsub push: malformed base64 data: " + err.Error()})
		return
	}

	if err := handler(c.Request.Context(), data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
