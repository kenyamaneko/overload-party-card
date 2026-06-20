package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/usecase"

	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
)

// PlayerCardHandler はプレイヤー所持カードの REST エンドポイントを処理します。
type PlayerCardHandler struct {
	playerCardInteractor *usecase.PlayerCardInteractor
}

// NewPlayerCardHandler は PlayerCardHandler を生成します。
func NewPlayerCardHandler(playerCardInteractor *usecase.PlayerCardInteractor) *PlayerCardHandler {
	return &PlayerCardHandler{playerCardInteractor: playerCardInteractor}
}

// GetPlayerCards はプレイヤーの所持カード一覧を返します。
func (h *PlayerCardHandler) GetPlayerCards(c *gin.Context) {
	playerID := c.GetString(internalauth.PlayerIDContextKey)
	cards, err := h.playerCardInteractor.GetPlayerCards(c.Request.Context(), playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cards)
}
