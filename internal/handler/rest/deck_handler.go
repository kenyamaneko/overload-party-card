package rest

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/service"
)

// DeckHandler はデッキ CRUD の REST エンドポイントを処理します。
type DeckHandler struct {
	deckService *service.DeckService
}

// NewDeckHandler は DeckHandler を生成します。
func NewDeckHandler(deckService *service.DeckService) *DeckHandler {
	return &DeckHandler{deckService: deckService}
}

// GetDecks はプレイヤーのデッキ一覧を返します。
func (h *DeckHandler) GetDecks(c *gin.Context) {
	playerID := c.Param("playerId")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "playerId is required"})
		return
	}

	decks, err := h.deckService.GetDecks(c.Request.Context(), playerID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, decks)
}

// GetDeck は指定デッキの詳細を返します。
func (h *DeckHandler) GetDeck(c *gin.Context) {
	playerID := c.Param("playerId")
	deckID, err := strconv.ParseInt(c.Param("deckId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck_id"})
		return
	}

	deck, cards, err := h.deckService.GetDeck(c.Request.Context(), playerID, deckID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deck": deck, "cards": cards})
}

// CreateDeck は新しいデッキを作成します。
func (h *DeckHandler) CreateDeck(c *gin.Context) {
	playerID := c.Param("playerId")

	var req service.CreateDeckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deck, err := h.deckService.CreateDeck(c.Request.Context(), playerID, req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, deck)
}

// UpdateDeck は既存デッキを更新します。
func (h *DeckHandler) UpdateDeck(c *gin.Context) {
	playerID := c.Param("playerId")
	deckID, err := strconv.ParseInt(c.Param("deckId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck_id"})
		return
	}

	var req service.UpdateDeckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deck, err := h.deckService.UpdateDeck(c.Request.Context(), playerID, deckID, req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, deck)
}

// DeleteDeck は指定デッキを削除します。
func (h *DeckHandler) DeleteDeck(c *gin.Context) {
	playerID := c.Param("playerId")
	deckID, err := strconv.ParseInt(c.Param("deckId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck_id"})
		return
	}

	if err := h.deckService.DeleteDeck(c.Request.Context(), playerID, deckID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ValidateDeckForBattle はデッキがバトル可能かを検証します。
func (h *DeckHandler) ValidateDeckForBattle(c *gin.Context) {
	playerID := c.Param("playerId")
	deckID, err := strconv.ParseInt(c.Param("deckId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck_id"})
		return
	}

	if err := h.deckService.ValidateDeckForBattle(c.Request.Context(), playerID, deckID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusOK)
}
