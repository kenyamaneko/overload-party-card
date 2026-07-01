package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"

	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
)

// TestDeckHandler_CreateDeck_Success は所持カードで構成された有効なリクエストが
// interactor を通って 201 と作成済みデッキ本文を返すことを検証します。
func TestDeckHandler_CreateDeck_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, pcRepo := newDeckHandlerFixture(t)
	pcRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: fxCardID, ArtNo: 0, Count: 3})

	body, err := json.Marshal(apicard.DeckCreateRequest{
		DeckName: "My Deck", Faction: fxFaction, ProductID: fxProductID,
		RoutineID: fxRoutineID, SpecialID: fxSpecialID,
		Cards: []apicard.DeckCardEntry{{CardID: fxCardID, ArtNo: 0, Count: 1}},
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(internalauth.PlayerIDContextKey, fxPlayerID)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateDeck(c)

	require.Equal(t, http.StatusCreated, w.Code)
	var got apicard.Deck
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, "My Deck", got.DeckName)
	assert.Equal(t, fxFaction, got.Faction)
	assert.Equal(t, fxPlayerID, got.PlayerID)
	assert.NotZero(t, got.DeckID)
}

// TestDeckHandler_CreateDeck_UnownedCard_Forbidden は所持していないカードを含む
// リクエストが interactor の所持検証で弾かれ 403 を返すことを検証します。
func TestDeckHandler_CreateDeck_UnownedCard_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _ := newDeckHandlerFixture(t)

	body, err := json.Marshal(apicard.DeckCreateRequest{
		DeckName: "My Deck", Faction: fxFaction, ProductID: fxProductID,
		RoutineID: fxRoutineID, SpecialID: fxSpecialID,
		Cards: []apicard.DeckCardEntry{{CardID: fxCardID, ArtNo: 0, Count: 1}},
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(internalauth.PlayerIDContextKey, fxPlayerID)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateDeck(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), `"error"`)
}

// TestDeckHandler_NonNumericDeckID_BadRequest は deckId が整数でないとき
// interactor に到達せず 400 を返すことを検証します。
func TestDeckHandler_NonNumericDeckID_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _ := newDeckHandlerFixture(t)

	cases := []struct {
		name   string
		invoke func(c *gin.Context)
	}{
		{"GetDeck", h.GetDeck},
		{"UpdateDeck", h.UpdateDeck},
		{"DeleteDeck", h.DeleteDeck},
		{"ValidateDeckForBattle", h.ValidateDeckForBattle},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set(internalauth.PlayerIDContextKey, fxPlayerID)
			c.Params = gin.Params{{Key: "deckId", Value: "not-a-number"}}
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

			tc.invoke(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "invalid deck_id")
		})
	}
}

// TestDeckHandler_MalformedJSONBody_BadRequest は本文が不正な JSON のとき
// interactor に到達せず 400 を返すことを検証します。
func TestDeckHandler_MalformedJSONBody_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _ := newDeckHandlerFixture(t)

	const numericDeckID = "1"
	const malformedJSON = `{"deck_name":`

	cases := []struct {
		name   string
		invoke func(c *gin.Context)
	}{
		{"CreateDeck", h.CreateDeck},
		{"UpdateDeck", h.UpdateDeck},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set(internalauth.PlayerIDContextKey, fxPlayerID)
			c.Params = gin.Params{{Key: "deckId", Value: numericDeckID}}
			c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(malformedJSON))
			c.Request.Header.Set("Content-Type", "application/json")

			tc.invoke(c)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), `"error"`)
		})
	}
}
