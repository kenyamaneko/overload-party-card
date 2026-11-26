package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/presenter"
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

// ListForPlayer は指定プレイヤーの所持状態を付与した全カードを返します。
func (h *CardHandler) ListForPlayer(c *gin.Context) {
	playerID := c.GetString(internalauth.PlayerIDContextKey)
	cards, err := h.cardInteractor.ListCardsWithOwnership(c.Request.Context(), playerID)
	if err != nil {
		respondError(c, err)
		return
	}
	response, err := presenter.ToCardsWithOwnership(cards)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}
