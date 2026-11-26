package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/presenter"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"

	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
)

// DeckHandler はデッキ CRUD の REST エンドポイントを処理します。
// デッキ ID を URL に含むエンドポイントは、ParseDeckID を通したあとに処理します。
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
	c.JSON(http.StatusOK, presenter.ToDecks(decks))
}

// GetDeck は指定デッキの詳細を返します。
func (h *DeckHandler) GetDeck(c *gin.Context) {
	playerID := c.GetString(internalauth.PlayerIDContextKey)
	deckID := c.GetInt64(deckIDContextKey)

	deck, err := h.deckInteractor.GetDeck(c.Request.Context(), playerID, deckID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, presenter.ToDeckWithCards(*deck))
}

// CreateDeck は新しいデッキを作成します。
func (h *DeckHandler) CreateDeck(c *gin.Context) {
	playerID := c.GetString(internalauth.PlayerIDContextKey)

	var request apicard.DeckCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c, malformedRequestBodyMessage)
		return
	}

	deck, err := h.deckInteractor.CreateDeck(c.Request.Context(), playerID, presenter.ToDeckDraftFromCreateRequest(request))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, presenter.ToDeck(*deck))
}

// UpdateDeck は既存デッキを更新します。
func (h *DeckHandler) UpdateDeck(c *gin.Context) {
	playerID := c.GetString(internalauth.PlayerIDContextKey)
	deckID := c.GetInt64(deckIDContextKey)

	var request apicard.DeckUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondBadRequest(c, malformedRequestBodyMessage)
		return
	}

	deck, err := h.deckInteractor.UpdateDeck(c.Request.Context(), playerID, deckID, presenter.ToDeckDraftFromUpdateRequest(request))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, presenter.ToDeck(*deck))
}

// DeleteDeck は指定デッキを削除します。
func (h *DeckHandler) DeleteDeck(c *gin.Context) {
	playerID := c.GetString(internalauth.PlayerIDContextKey)
	deckID := c.GetInt64(deckIDContextKey)

	if err := h.deckInteractor.DeleteDeck(c.Request.Context(), playerID, deckID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ValidateDeckForBattle はデッキがバトル可能かを検証します。
func (h *DeckHandler) ValidateDeckForBattle(c *gin.Context) {
	playerID := c.GetString(internalauth.PlayerIDContextKey)
	deckID := c.GetInt64(deckIDContextKey)

	if err := h.deckInteractor.ValidateDeckForBattle(c.Request.Context(), playerID, deckID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusOK)
}
