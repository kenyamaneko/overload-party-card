package rest

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/port"

	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
)

func TestDeckHandler_SentinelErrorToStatusAndBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("[デッキAPI]sentinel errorのHTTPステータスと本文への写像", func(t *testing.T) {
		cases := []struct {
			name       string
			err        error
			wantStatus int
			wantBody   string
		}{
			{"ErrNotFound のとき、404 になる", port.ErrNotFound, http.StatusNotFound, "find deck: " + port.ErrNotFound.Error()},
			{"ErrUnowned のとき、403 になる", port.ErrUnowned, http.StatusForbidden, "find deck: " + port.ErrUnowned.Error()},
			{"ErrInvalidDeck のとき、400 になる", port.ErrInvalidDeck, http.StatusBadRequest, "find deck: " + port.ErrInvalidDeck.Error()},
			{"ErrRestrictionExceeded のとき、400 になる", port.ErrRestrictionExceeded, http.StatusBadRequest, "find deck: " + port.ErrRestrictionExceeded.Error()},
			{"ErrInvalidArgument のとき、400 になる", port.ErrInvalidArgument, http.StatusBadRequest, "find deck: " + port.ErrInvalidArgument.Error()},
			{"ラップされた ErrNotFound のとき、404 に写像する", fmt.Errorf("db boom: %w", port.ErrNotFound), http.StatusNotFound, "find deck: db boom: not found"},
			{"写像対象外の error のとき、500 になり応答本文の error が internal server error と完全に一致する", errors.New("unexpected"), http.StatusInternalServerError, internalErrorMessage},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				h, deckRepo, _ := newDeckHandlerFixture(t)
				deckRepo.findByIDErr = tc.err

				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Set(internalauth.PlayerIDContextKey, fxPlayerID)
				c.Params = gin.Params{{Key: "deckId", Value: "1"}}
				c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

				h.GetDeck(c)

				assert.Equal(t, tc.wantStatus, w.Code)
				assert.Equal(t, tc.wantBody, decodeBody(t, w)["error"])
			})
		}
	})
}

func TestRespondError_InternalServerErrorLogging(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("[デッキAPI]500になるエラーのログ出力", func(t *testing.T) {
		t.Run("写像対象外のerrorのとき、応答本文には出ない原因の文字列と要求パスがログに出る", func(t *testing.T) {
			buf := captureSlog(t)
			cause := errors.New("pq: connection refused")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/cards/decks", nil)

			respondError(c, cause)

			assert.Equal(t, http.StatusInternalServerError, w.Code)
			assert.Equal(t, internalErrorMessage, decodeBody(t, w)["error"])

			records := decodeLogRecords(t, buf)
			require.Len(t, records, 1)
			assert.Equal(t, "/api/v1/cards/decks", records[0]["path"])
			assert.Equal(t, cause.Error(), records[0]["error"])
		})
	})
}
