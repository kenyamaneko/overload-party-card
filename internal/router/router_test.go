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

	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// fakeRouterVerifier は router 単体テスト用の internalauth.Verifier 最小 fake。
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

func newTestRouter(verifier internalauth.Verifier) *gin.Engine {
	return New(
		rest.NewCardHandler(nil),
		rest.NewDeckHandler(nil),
		rest.NewPlayerCardHandler(nil),
		rest.NewProductHandler(nil, nil),
		rest.NewInitiativeHandler(nil),
		verifier,
	)
}

func TestNew(t *testing.T) {
	t.Run("ルーターの認証配線", func(t *testing.T) {
		t.Run("/health は auth middleware を通らず 200 を返す", func(t *testing.T) {
			r := newTestRouter(nullVerifier{})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("/internal/v1/cards (master 配信) は auth middleware を通らず 401 にならない", func(t *testing.T) {
			// nullVerifier は呼ばれると panic するので、NotPanics で auth middleware を
			// 通らないことを、401 でないことで認証が要求されないことを確かめる。
			r := newTestRouter(nullVerifier{})
			w := httptest.NewRecorder()
			require.NotPanics(t, func() {
				r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/internal/v1/cards", nil))
			})
			assert.NotEqual(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("/api/v1/cards 配下は auth header 欠落で 401 になる", func(t *testing.T) {
			r := newTestRouter(fakeRouterVerifier{playerID: "irrelevant"})

			cases := []struct {
				name string
				path string
			}{
				{name: "/api/v1/cards/decks は 401 になる", path: "/api/v1/cards/decks"},
				{name: "/api/v1/cards/cards は 401 になる", path: "/api/v1/cards/cards"},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					w := httptest.NewRecorder()
					r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
					assert.Equal(t, http.StatusUnauthorized, w.Code)
				})
			}
		})

		t.Run("verifier がエラーを返すとき、401 になる", func(t *testing.T) {
			r := newTestRouter(fakeRouterVerifier{err: errors.New("invalid token")})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/cards/decks", nil)
			req.Header.Set(internalauth.HeaderName, "any.token")
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	})
}
