package router

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

// stubMessageHandler は router 単体テスト用の port.MessageHandler 最小 stub。
// 常に ack (nil) を返す。
func stubMessageHandler(context.Context, []byte) error {
	return nil
}

func newTestRouter(verifier internalauth.Verifier) *gin.Engine {
	return New(
		rest.NewCardHandler(nil),
		rest.NewDeckHandler(nil),
		rest.NewPlayerCardHandler(nil),
		rest.NewProductHandler(nil, nil),
		rest.NewInitiativeHandler(nil),
		rest.NewPubSubPushHandler(stubMessageHandler, stubMessageHandler),
		verifier,
	)
}

// newPubSubPushRequest は push subscription が送る envelope を模した POST body を返す。
func newPubSubPushRequest(path string) *http.Request {
	body := `{"message":{"data":"` + base64.StdEncoding.EncodeToString([]byte(`{}`)) + `"}}`
	return httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
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

		t.Run("/internal/v1/pubsub 配下 (push 受け口) は auth header なしで handler の成功応答まで到達する", func(t *testing.T) {
			// nullVerifier を使うと、もし auth middleware を経由してしまった場合は
			// Verify 呼び出しで panic し gin.Recovery が 500 に丸めるため、
			// stubMessageHandler の ack (200) との差で到達有無を検出できる。
			r := newTestRouter(nullVerifier{})

			cases := []struct {
				name string
				path string
			}{
				{name: "/internal/v1/pubsub/player-onboarded は 200 を返す", path: "/internal/v1/pubsub/player-onboarded"},
				{name: "/internal/v1/pubsub/card-pack-purchased は 200 を返す", path: "/internal/v1/pubsub/card-pack-purchased"},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					w := httptest.NewRecorder()
					r.ServeHTTP(w, newPubSubPushRequest(tc.path))
					assert.Equal(t, http.StatusOK, w.Code)
				})
			}
		})

		t.Run("/api/v1/cards 配下は auth header 欠落で 401 になる", func(t *testing.T) {
			r := newTestRouter(fakeRouterVerifier{playerID: "irrelevant"})

			cases := []struct {
				name string
				path string
			}{
				{name: "デッキ一覧の取得は 401 になり、認証ヘッダが無いことを示すエラーが応答本文に含まれる", path: "/api/v1/cards/decks"},
				{name: "所持カード一覧の取得は 401 になり、認証ヘッダが無いことを示すエラーが応答本文に含まれる", path: "/api/v1/cards/cards"},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					w := httptest.NewRecorder()
					r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
					assert.Equal(t, http.StatusUnauthorized, w.Code)
					assert.Contains(t, w.Body.String(), "header is required")
				})
			}
		})

		t.Run("認証トークンの検証が失敗するとき、401 になり、トークンの検証に失敗したことを示すエラーが応答本文に含まれる", func(t *testing.T) {
			r := newTestRouter(fakeRouterVerifier{err: errors.New("invalid token")})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/cards/decks", nil)
			req.Header.Set(internalauth.HeaderName, "any.token")
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Contains(t, w.Body.String(), "invalid internal auth token")
		})
	})
}
