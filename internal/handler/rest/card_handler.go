package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/service"
)

// CardHandler はカードマスター関連の REST エンドポイントを処理します。
type CardHandler struct {
	cardService *service.CardService
}

// NewCardHandler は CardHandler を生成します。
func NewCardHandler(cardService *service.CardService) *CardHandler {
	return &CardHandler{cardService: cardService}
}

// ListAllRaw はカードマスター全件を返します。
// battle 起動時と gateway キャッシュ構築で使用されます。
func (h *CardHandler) ListAllRaw(c *gin.Context) {
	cards, err := h.cardService.FindAllRaw(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cards)
}

// ListForPlayer は指定プレイヤーの所持状態を付与した全カードを返します。
func (h *CardHandler) ListForPlayer(c *gin.Context) {
	playerID := c.Param("playerId")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "playerId is required"})
		return
	}
	cards, err := h.cardService.GetAllCards(c.Request.Context(), playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cards)
}
