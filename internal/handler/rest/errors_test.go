package rest

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/port"

	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
)

func TestDeckHandler_SentinelErrorToStatusAndBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("sentinel error の HTTP ステータスと本文への写像", func(t *testing.T) {
		cases := []struct {
			name       string
			err        error
			wantStatus int
			wantBody   string
		}{
			{"ErrNotFound のとき、404 になる", port.ErrNotFound, http.StatusNotFound, port.ErrNotFound.Error()},
			{"ErrUnowned のとき、403 になる", port.ErrUnowned, http.StatusForbidden, port.ErrUnowned.Error()},
			{"ErrInvalidDeck のとき、400 になる", port.ErrInvalidDeck, http.StatusBadRequest, port.ErrInvalidDeck.Error()},
			{"ErrRestrictionExceeded のとき、400 になる", port.ErrRestrictionExceeded, http.StatusBadRequest, port.ErrRestrictionExceeded.Error()},
			{"ErrInvalidArgument のとき、400 になる", port.ErrInvalidArgument, http.StatusBadRequest, port.ErrInvalidArgument.Error()},
			{"ラップされた ErrNotFound のとき、404 に写像する", fmt.Errorf("db boom: %w", port.ErrNotFound), http.StatusNotFound, port.ErrNotFound.Error()},
			{"写像対象外の error のとき、500 になる", errors.New("unexpected"), http.StatusInternalServerError, "unexpected"},
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
				assert.Contains(t, w.Body.String(), `"error"`)
				assert.Contains(t, w.Body.String(), tc.wantBody)
			})
		}
	})
}
