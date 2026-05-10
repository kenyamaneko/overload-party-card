package router

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/handler/rest"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// fakeRouterVerifier は router 単体テスト用の port.InternalAuthVerifier 最小 fake。
type fakeRouterVerifier struct {
	playerID string
	err      error
}

func (f fakeRouterVerifier) Verify(string) (string, error) {
	return f.playerID, f.err
}

// nullVerifier は auth 経路を通らない (health / internal master) の検証用に Verify を呼ばれてはならないことを示す。
type nullVerifier struct{}

func (nullVerifier) Verify(string) (string, error) {
	panic("Verify should not be called for routes outside /api/v1/cards")
}

func newTestRouter(verifier port.InternalAuthVerifier) *gin.Engine {
	return New(
		rest.NewCardHandler(nil),
		rest.NewDeckHandler(nil),
		rest.NewPlayerCardHandler(nil),
		verifier,
	)
}

// /health は auth middleware を通らず常に 200 を返す。
func TestNew_HealthEndpoint(t *testing.T) {
	r := newTestRouter(nullVerifier{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

// /internal/v1/cards (master 配信) は battle 互換のため auth middleware を通らず、
// handler が登録されている (Verify が呼ばれないこと自体が登録の証拠)。
// nil 依存により 500 を返すが、auth middleware が走っていれば nullVerifier が panic する。
func TestNew_InternalCardsRouteIsAuthFree(t *testing.T) {
	r := newTestRouter(nullVerifier{})
	w := httptest.NewRecorder()
	require.NotPanics(t, func() {
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/internal/v1/cards", nil))
	})
	assert.NotEqual(t, http.StatusUnauthorized, w.Code)
}

// /api/v1/cards 配下は auth middleware を通る。
// header 欠落時は 401 を返し、handler に到達しないことを確認する。
func TestNew_ApiRouteRequiresInternalAuth(t *testing.T) {
	r := newTestRouter(fakeRouterVerifier{playerID: "irrelevant"})

	cases := []struct {
		name string
		path string
	}{
		{name: "/api/v1/cards/decks は auth header 欠落で 401", path: "/api/v1/cards/decks"},
		{name: "/api/v1/cards/cards は auth header 欠落で 401", path: "/api/v1/cards/cards"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// verifier が error を返すと 401 を返し handler に到達しない。
func TestNew_ApiRouteRejectsVerifierError(t *testing.T) {
	r := newTestRouter(fakeRouterVerifier{err: errors.New("invalid token")})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cards/decks", nil)
	req.Header.Set(rest.HeaderName, "any.token")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
