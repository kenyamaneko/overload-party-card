package rest

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/usecase"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"

	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
)

// DeckHandler はデッキ CRUD の REST エンドポイントを処理します。
type DeckHandler struct {
	deckInteractor *usecase.DeckInteractor
}

// NewDeckHandler は DeckHandler を生成します。
func NewDeckHandler(deckInteractor *usecase.DeckInteractor) *DeckHandler {
	return &DeckHandler{deckInteractor: deckInteractor}
}

// GetDecks はプレイヤーのデッキ一覧を返します。
func (h *DeckHandler) GetDecks(c *gin.Context) {
	playerID := c.GetString(internalauth.PlayerIDContextKey)

	decks, err := h.deckInteractor.GetDecks(c.Request.Context(), playerID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, decks)
}

// GetDeck は指定デッキの詳細を返します。
func (h *DeckHandler) GetDeck(c *gin.Context) {
	playerID := c.GetString(internalauth.PlayerIDContextKey)
	deckID, err := strconv.ParseInt(c.Param("deckId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck_id"})
		return
	}

	deck, cards, err := h.deckInteractor.GetDeck(c.Request.Context(), playerID, deckID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deck": deck, "cards": cards})
}

// CreateDeck は新しいデッキを作成します。
func (h *DeckHandler) CreateDeck(c *gin.Context) {
	playerID := c.GetString(internalauth.PlayerIDContextKey)

	var req apicard.DeckCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deck, err := h.deckInteractor.CreateDeck(c.Request.Context(), playerID, req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, deck)
}

// UpdateDeck は既存デッキを更新します。
func (h *DeckHandler) UpdateDeck(c *gin.Context) {
	playerID := c.GetString(internalauth.PlayerIDContextKey)
	deckID, err := strconv.ParseInt(c.Param("deckId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck_id"})
		return
	}

	var req apicard.DeckUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deck, err := h.deckInteractor.UpdateDeck(c.Request.Context(), playerID, deckID, req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, deck)
}

// DeleteDeck は指定デッキを削除します。
func (h *DeckHandler) DeleteDeck(c *gin.Context) {
	playerID := c.GetString(internalauth.PlayerIDContextKey)
	deckID, err := strconv.ParseInt(c.Param("deckId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck_id"})
		return
	}

	if err := h.deckInteractor.DeleteDeck(c.Request.Context(), playerID, deckID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ValidateDeckForBattle はデッキがバトル可能かを検証します。
func (h *DeckHandler) ValidateDeckForBattle(c *gin.Context) {
	playerID := c.GetString(internalauth.PlayerIDContextKey)
	deckID, err := strconv.ParseInt(c.Param("deckId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deck_id"})
		return
	}

	if err := h.deckInteractor.ValidateDeckForBattle(c.Request.Context(), playerID, deckID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusOK)
}
