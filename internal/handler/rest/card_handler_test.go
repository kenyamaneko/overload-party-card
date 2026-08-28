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

func TestCardHandler_RepositoryFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("[handler]カードマスター取得", func(t *testing.T) {
		h, cardRepo, _ := newCardHandlerFixture()
		cardRepo.findAllErr = errors.New("pq: connection refused")

		cases := []struct {
			name   string
			invoke func(c *gin.Context)
		}{
			{"カードマスター全件取得で内部エラーが起きたとき、500 になり応答本文の error が internal server error と完全に一致する", h.ListAllRaw},
			{"プレイヤー所持状態付きカード取得で内部エラーが起きたとき、500 になり応答本文の error が internal server error と完全に一致する", h.ListForPlayer},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Set(internalauth.PlayerIDContextKey, fxPlayerID)
				c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

				tc.invoke(c)

				assert.Equal(t, http.StatusInternalServerError, w.Code)
				assert.Equal(t, internalErrorMessage, decodeBody(t, w)["error"])
			})
		}
	})
}
