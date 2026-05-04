package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/usecase"
)

// GrantHandler はカードパック配布の REST エンドポイントを処理します。
type GrantHandler struct {
	grantInteractor *usecase.GrantInteractor
}

// NewGrantHandler は GrantHandler を生成します。
func NewGrantHandler(grantInteractor *usecase.GrantInteractor) *GrantHandler {
	return &GrantHandler{grantInteractor: grantInteractor}
}

type grantPackRequest struct {
	Faction string `json:"faction"`
}

type grantPackResponse struct {
	CardsGranted int `json:"cardsGranted"`
}

// GrantInitialPack は初期ファクション選択時のカードパック配布を処理します。
func (h *GrantHandler) GrantInitialPack(c *gin.Context) {
	playerID := c.Param("playerId")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "playerId is required"})
		return
	}

	var req grantPackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	granted, err := h.grantInteractor.GrantInitialPack(c.Request.Context(), playerID, req.Faction)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, grantPackResponse{CardsGranted: granted})
}

// GrantFactionPack はショップ購入時のファクションパック配布を処理します。
func (h *GrantHandler) GrantFactionPack(c *gin.Context) {
	playerID := c.Param("playerId")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "playerId is required"})
		return
	}

	var req grantPackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	granted, err := h.grantInteractor.GrantFactionPack(c.Request.Context(), playerID, req.Faction)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, grantPackResponse{CardsGranted: granted})
}
