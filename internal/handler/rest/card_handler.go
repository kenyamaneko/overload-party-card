package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/usecase"

	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
)

// CardHandler はカードマスター関連の REST エンドポイントを処理します。
type CardHandler struct {
	cardInteractor *usecase.CardInteractor
}

// NewCardHandler は CardHandler を生成します。
func NewCardHandler(cardInteractor *usecase.CardInteractor) *CardHandler {
	return &CardHandler{cardInteractor: cardInteractor}
}

// ListAllRaw はカードマスター全件を返します。
// battle 起動時と gateway キャッシュ構築で使用されます。
func (h *CardHandler) ListAllRaw(c *gin.Context) {
	cards, err := h.cardInteractor.ListCards(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cards)
}

// ListForPlayer は指定プレイヤーの所持状態を付与した全カードを返します。
func (h *CardHandler) ListForPlayer(c *gin.Context) {
	playerID := c.GetString(internalauth.PlayerIDContextKey)
	cards, err := h.cardInteractor.ListCardsWithOwnership(c.Request.Context(), playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cards)
}
