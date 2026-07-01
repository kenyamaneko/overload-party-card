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

// TestDeckHandler_SentinelErrorToStatusAndBody は interactor が返す sentinel error が対応する HTTP ステータスと error 本文へ写像されることを検証します。
func TestDeckHandler_SentinelErrorToStatusAndBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{"ErrNotFound は 404", port.ErrNotFound, http.StatusNotFound, port.ErrNotFound.Error()},
		{"ErrUnowned は 403", port.ErrUnowned, http.StatusForbidden, port.ErrUnowned.Error()},
		{"ErrInvalidDeck は 400", port.ErrInvalidDeck, http.StatusBadRequest, port.ErrInvalidDeck.Error()},
		{"ErrRestrictionExceeded は 400", port.ErrRestrictionExceeded, http.StatusBadRequest, port.ErrRestrictionExceeded.Error()},
		{"ErrInvalidArgument は 400", port.ErrInvalidArgument, http.StatusBadRequest, port.ErrInvalidArgument.Error()},
		{"ラップされた sentinel も写像する", fmt.Errorf("db boom: %w", port.ErrNotFound), http.StatusNotFound, port.ErrNotFound.Error()},
		{"写像対象外の error は 500", errors.New("unexpected"), http.StatusInternalServerError, "unexpected"},
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
}
