package rest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
)

func TestPlayerCardHandler_GetPlayerCards(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("所持カード取得", func(t *testing.T) {
		t.Run("所持カードの取得に失敗したとき、500になり応答本文のerrorがinternal server errorと完全に一致する", func(t *testing.T) {
			h, pcRepo := newPlayerCardHandlerFixture()
			pcRepo.getErr = errors.New("pq: connection refused")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set(internalauth.PlayerIDContextKey, fxPlayerID)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

			h.GetPlayerCards(c)

			assert.Equal(t, http.StatusInternalServerError, w.Code)
			assert.Equal(t, internalErrorMessage, decodeBody(t, w)["error"])
		})
	})
}
