package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/service"
)

// PlayerCardHandler はプレイヤー所持カードの REST エンドポイントを処理します。
type PlayerCardHandler struct {
	deckService *service.DeckService
}

// NewPlayerCardHandler は PlayerCardHandler を生成します。
func NewPlayerCardHandler(deckService *service.DeckService) *PlayerCardHandler {
	return &PlayerCardHandler{deckService: deckService}
}

// GetPlayerCards はプレイヤーの所持カード一覧を返します。
func (h *PlayerCardHandler) GetPlayerCards(c *gin.Context) {
	playerID := c.Param("playerId")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "playerId is required"})
		return
	}
	cards, err := h.deckService.GetPlayerCards(c.Request.Context(), playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cards)
}
