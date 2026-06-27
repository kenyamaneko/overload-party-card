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
)

// TestToHTTPStatus は各 sentinel error が対応する HTTP ステータスへ写像され、
// 未知 error が 500 に落ちることを検証します。
func TestToHTTPStatus(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"ErrNotFound は 404", port.ErrNotFound, http.StatusNotFound},
		{"ErrUnowned は 403", port.ErrUnowned, http.StatusForbidden},
		{"ErrInvalidDeck は 400", port.ErrInvalidDeck, http.StatusBadRequest},
		{"ErrRestrictionExceeded は 400", port.ErrRestrictionExceeded, http.StatusBadRequest},
		{"ErrInvalidArgument は 400", port.ErrInvalidArgument, http.StatusBadRequest},
		{"ラップされた sentinel も写像する", fmt.Errorf("find deck: %w", port.ErrNotFound), http.StatusNotFound},
		{"写像対象外の error は 500", errors.New("unexpected"), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantStatus, toHTTPStatus(tc.err))
		})
	}
}

// TestRespondError は写像したステータスと error メッセージを含む応答本文を
// 書き出すことを検証します。
func TestRespondError(t *testing.T) {
	cases := []struct {
		name        string
		err         error
		wantStatus  int
		wantMessage string
	}{
		{"ErrNotFound は 404 と本文", port.ErrNotFound, http.StatusNotFound, port.ErrNotFound.Error()},
		{"ErrUnowned は 403 と本文", port.ErrUnowned, http.StatusForbidden, port.ErrUnowned.Error()},
		{"写像対象外は 500 と本文", errors.New("unexpected"), http.StatusInternalServerError, "unexpected"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			respondError(c, tc.err)

			assert.Equal(t, tc.wantStatus, w.Code)
			assert.Contains(t, w.Body.String(), `"error"`)
			assert.Contains(t, w.Body.String(), tc.wantMessage)
		})
	}
}
