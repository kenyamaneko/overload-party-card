package rest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeckHandler_NonNumericDeckID_BadRequest は deckId が整数でない場合に 400 を返し、
// interactor に到達しないことを検証します。
func TestDeckHandler_NonNumericDeckID_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// interactor を nil で渡し、到達すれば nil 参照 panic になることを短絡の検出に用いる。
	h := NewDeckHandler(nil)

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
			c.Params = gin.Params{{Key: "deckId", Value: "not-a-number"}}
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

			require.NotPanics(t, func() { tc.invoke(c) })
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// TestDeckHandler_MalformedJSONBody_BadRequest は不正な JSON 本文の場合に
// 400 を返し、interactor に到達しないことを検証します。
func TestDeckHandler_MalformedJSONBody_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewDeckHandler(nil)

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
			c.Params = gin.Params{{Key: "deckId", Value: numericDeckID}}
			c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(malformedJSON))
			c.Request.Header.Set("Content-Type", "application/json")

			require.NotPanics(t, func() { tc.invoke(c) })
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}
