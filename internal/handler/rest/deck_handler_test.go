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

func TestDeckHandler_CreateDeck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("デッキ作成", func(t *testing.T) {
		t.Run("所持カードで構成した有効なリクエストのとき、201 と作成済みデッキを返す", func(t *testing.T) {
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
		})

		t.Run("未所持カードを含むとき、所持検証で弾かれ 403 になる", func(t *testing.T) {
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
		})
	})
}

func TestDeckHandler_NonNumericDeckID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("非整数 deckId の拒否", func(t *testing.T) {
		h, _, _ := newDeckHandlerFixture(t)

		cases := []struct {
			name   string
			invoke func(c *gin.Context)
		}{
			{"GetDeck のとき、400 invalid deck_id になる", h.GetDeck},
			{"UpdateDeck のとき、400 invalid deck_id になる", h.UpdateDeck},
			{"DeleteDeck のとき、400 invalid deck_id になる", h.DeleteDeck},
			{"ValidateDeckForBattle のとき、400 invalid deck_id になる", h.ValidateDeckForBattle},
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
	})
}

func TestDeckHandler_MalformedJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("不正な JSON 本文の拒否", func(t *testing.T) {
		h, _, _ := newDeckHandlerFixture(t)

		const numericDeckID = "1"
		const malformedJSON = `{"deck_name":`

		cases := []struct {
			name   string
			invoke func(c *gin.Context)
		}{
			{"CreateDeck のとき、400 になる", h.CreateDeck},
			{"UpdateDeck のとき、400 になる", h.UpdateDeck},
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
	})
}
