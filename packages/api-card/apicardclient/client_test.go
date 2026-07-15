package apicardclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	"github.com/kenyamaneko/overload-party-card/packages/api-card/apicardclient"
	"github.com/kenyamaneko/overload-party-card/packages/api-card/apicardserverfake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ListCards(t *testing.T) {
	t.Run("ListCards", func(t *testing.T) {
		t.Run("500 を受けたとき、ErrInternalServer になる", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.ListAllCardsFn = func() (int, any) { return http.StatusInternalServerError, nil }

			c := newTestClient(t, srv.URL())
			_, err := c.ListCards(context.Background())
			assertSentinel(t, err, apicardclient.ErrInternalServer)
		})
	})
}

func TestClient_ListPlayerCards(t *testing.T) {
	t.Run("ListPlayerCards", func(t *testing.T) {
		cases := []struct {
			name       string
			status     int
			wantTarget error
		}{
			{
				name:       "401 を受けたとき、ErrUnauthorized になる",
				status:     http.StatusUnauthorized,
				wantTarget: apicardclient.ErrUnauthorized,
			},
			{
				name:       "500 を受けたとき、ErrInternalServer になる",
				status:     http.StatusInternalServerError,
				wantTarget: apicardclient.ErrInternalServer,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				srv := apicardserverfake.NewServer()
				defer srv.Close()
				srv.ListPlayerCardsFn = func() (int, any) { return tc.status, nil }

				c := newTestClient(t, srv.URL())
				_, err := c.ListPlayerCards(context.Background())
				assertSentinel(t, err, tc.wantTarget)
			})
		}
	})
}

func TestClient_ListCardsWithOwnership(t *testing.T) {
	t.Run("ListCardsWithOwnership", func(t *testing.T) {
		cases := []struct {
			name       string
			status     int
			wantTarget error
		}{
			{
				name:       "401 を受けたとき、ErrUnauthorized になる",
				status:     http.StatusUnauthorized,
				wantTarget: apicardclient.ErrUnauthorized,
			},
			{
				name:       "500 を受けたとき、ErrInternalServer になる",
				status:     http.StatusInternalServerError,
				wantTarget: apicardclient.ErrInternalServer,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				srv := apicardserverfake.NewServer()
				defer srv.Close()
				srv.ListCardsWithOwnershipFn = func() (int, any) { return tc.status, nil }

				c := newTestClient(t, srv.URL())
				_, err := c.ListCardsWithOwnership(context.Background())
				assertSentinel(t, err, tc.wantTarget)
			})
		}
	})
}

func TestClient_ListDecks(t *testing.T) {
	t.Run("ListDecks", func(t *testing.T) {
		cases := []struct {
			name       string
			status     int
			wantTarget error
		}{
			{
				name:       "401 を受けたとき、ErrUnauthorized になる",
				status:     http.StatusUnauthorized,
				wantTarget: apicardclient.ErrUnauthorized,
			},
			{
				name:       "404 を受けたとき、ErrNotFound になる",
				status:     http.StatusNotFound,
				wantTarget: apicardclient.ErrNotFound,
			},
			{
				name:       "500 を受けたとき、ErrInternalServer になる",
				status:     http.StatusInternalServerError,
				wantTarget: apicardclient.ErrInternalServer,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				srv := apicardserverfake.NewServer()
				defer srv.Close()
				srv.ListDecksFn = func() (int, any) { return tc.status, nil }

				c := newTestClient(t, srv.URL())
				_, err := c.ListDecks(context.Background())
				assertSentinel(t, err, tc.wantTarget)
			})
		}
	})
}

func TestClient_CreateDeck(t *testing.T) {
	t.Run("CreateDeck", func(t *testing.T) {
		cases := []struct {
			name       string
			status     int
			wantTarget error
		}{
			{
				name:       "400 を受けたとき、ErrBadRequest になる",
				status:     http.StatusBadRequest,
				wantTarget: apicardclient.ErrBadRequest,
			},
			{
				name:       "401 を受けたとき、ErrUnauthorized になる",
				status:     http.StatusUnauthorized,
				wantTarget: apicardclient.ErrUnauthorized,
			},
			{
				name:       "403 を受けたとき、ErrForbidden になる",
				status:     http.StatusForbidden,
				wantTarget: apicardclient.ErrForbidden,
			},
			{
				name:       "500 を受けたとき、ErrInternalServer になる",
				status:     http.StatusInternalServerError,
				wantTarget: apicardclient.ErrInternalServer,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				srv := apicardserverfake.NewServer()
				defer srv.Close()
				srv.CreateDeckFn = func(_ apicard.DeckCreateRequest) (int, any) { return tc.status, nil }

				c := newTestClient(t, srv.URL())
				_, err := c.CreateDeck(context.Background(), apicard.DeckCreateRequest{})
				assertSentinel(t, err, tc.wantTarget)
			})
		}
	})
}

func TestClient_GetDeck(t *testing.T) {
	t.Run("GetDeck", func(t *testing.T) {
		cases := []struct {
			name       string
			status     int
			wantTarget error
		}{
			{
				name:       "400 を受けたとき、ErrBadRequest になる",
				status:     http.StatusBadRequest,
				wantTarget: apicardclient.ErrBadRequest,
			},
			{
				name:       "401 を受けたとき、ErrUnauthorized になる",
				status:     http.StatusUnauthorized,
				wantTarget: apicardclient.ErrUnauthorized,
			},
			{
				name:       "404 を受けたとき、ErrNotFound になる",
				status:     http.StatusNotFound,
				wantTarget: apicardclient.ErrNotFound,
			},
			{
				name:       "500 を受けたとき、ErrInternalServer になる",
				status:     http.StatusInternalServerError,
				wantTarget: apicardclient.ErrInternalServer,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				srv := apicardserverfake.NewServer()
				defer srv.Close()
				srv.GetDeckFn = func(_ string) (int, any) { return tc.status, nil }

				c := newTestClient(t, srv.URL())
				_, _, err := c.GetDeck(context.Background(), 1)
				assertSentinel(t, err, tc.wantTarget)
			})
		}
	})
}

func TestClient_UpdateDeck(t *testing.T) {
	t.Run("UpdateDeck", func(t *testing.T) {
		cases := []struct {
			name       string
			status     int
			wantTarget error
		}{
			{
				name:       "400 を受けたとき、ErrBadRequest になる",
				status:     http.StatusBadRequest,
				wantTarget: apicardclient.ErrBadRequest,
			},
			{
				name:       "401 を受けたとき、ErrUnauthorized になる",
				status:     http.StatusUnauthorized,
				wantTarget: apicardclient.ErrUnauthorized,
			},
			{
				name:       "403 を受けたとき、ErrForbidden になる",
				status:     http.StatusForbidden,
				wantTarget: apicardclient.ErrForbidden,
			},
			{
				name:       "404 を受けたとき、ErrNotFound になる",
				status:     http.StatusNotFound,
				wantTarget: apicardclient.ErrNotFound,
			},
			{
				name:       "500 を受けたとき、ErrInternalServer になる",
				status:     http.StatusInternalServerError,
				wantTarget: apicardclient.ErrInternalServer,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				srv := apicardserverfake.NewServer()
				defer srv.Close()
				srv.UpdateDeckFn = func(_ string, _ apicard.DeckUpdateRequest) (int, any) { return tc.status, nil }

				c := newTestClient(t, srv.URL())
				_, err := c.UpdateDeck(context.Background(), 1, apicard.DeckUpdateRequest{})
				assertSentinel(t, err, tc.wantTarget)
			})
		}
	})
}

func TestClient_DeleteDeck(t *testing.T) {
	t.Run("DeleteDeck", func(t *testing.T) {
		cases := []struct {
			name       string
			status     int
			wantTarget error
		}{
			{
				name:       "400 を受けたとき、ErrBadRequest になる",
				status:     http.StatusBadRequest,
				wantTarget: apicardclient.ErrBadRequest,
			},
			{
				name:       "401 を受けたとき、ErrUnauthorized になる",
				status:     http.StatusUnauthorized,
				wantTarget: apicardclient.ErrUnauthorized,
			},
			{
				name:       "404 を受けたとき、ErrNotFound になる",
				status:     http.StatusNotFound,
				wantTarget: apicardclient.ErrNotFound,
			},
			{
				name:       "500 を受けたとき、ErrInternalServer になる",
				status:     http.StatusInternalServerError,
				wantTarget: apicardclient.ErrInternalServer,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				srv := apicardserverfake.NewServer()
				defer srv.Close()
				srv.DeleteDeckFn = func(_ string) (int, any) { return tc.status, nil }

				c := newTestClient(t, srv.URL())
				err := c.DeleteDeck(context.Background(), 1)
				assertSentinel(t, err, tc.wantTarget)
			})
		}
	})
}

func TestClient_ValidateDeckForBattle(t *testing.T) {
	t.Run("ValidateDeckForBattle", func(t *testing.T) {
		// 400 のみ ErrBadRequest ではなく ErrDeckInvalid に変換する (wire 契約上「デッキ不正」を意味するため特別扱い)。
		cases := []struct {
			name       string
			status     int
			wantTarget error
		}{
			{
				name:       "400 を受けたとき、ErrBadRequest ではなく ErrDeckInvalid になる",
				status:     http.StatusBadRequest,
				wantTarget: apicardclient.ErrDeckInvalid,
			},
			{
				name:       "401 を受けたとき、ErrUnauthorized になる",
				status:     http.StatusUnauthorized,
				wantTarget: apicardclient.ErrUnauthorized,
			},
			{
				name:       "403 を受けたとき、ErrForbidden になる",
				status:     http.StatusForbidden,
				wantTarget: apicardclient.ErrForbidden,
			},
			{
				name:       "404 を受けたとき、ErrNotFound になる",
				status:     http.StatusNotFound,
				wantTarget: apicardclient.ErrNotFound,
			},
			{
				name:       "500 を受けたとき、ErrInternalServer になる",
				status:     http.StatusInternalServerError,
				wantTarget: apicardclient.ErrInternalServer,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				srv := apicardserverfake.NewServer()
				defer srv.Close()
				srv.ValidateDeckForBattleFn = func(_ string) (int, any) { return tc.status, nil }

				c := newTestClient(t, srv.URL())
				err := c.ValidateDeckForBattle(context.Background(), 1)
				assertSentinel(t, err, tc.wantTarget)
			})
		}
	})
}

func TestClient_RequestEditor(t *testing.T) {
	t.Run("リクエストエディタの適用", func(t *testing.T) {
		t.Run("設定したヘッダが送信先の全リクエストに付与される", func(t *testing.T) {
			// X-Internal-Auth header 注入の接続点として SDK が機能することを担保する。
			var gotHeader string
			spy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotHeader = r.Header.Get("X-Internal-Auth")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("[]"))
			}))
			defer spy.Close()

			c, err := apicardclient.New(spy.URL,
				apicardclient.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
					req.Header.Set("X-Internal-Auth", "test-token")
					return nil
				}),
			)
			require.NoError(t, err)

			_, err = c.ListDecks(context.Background())
			require.NoError(t, err)
			assert.Equal(t, "test-token", gotHeader)
		})
	})
}

func newTestClient(t *testing.T, baseURL string) *apicardclient.Client {
	t.Helper()
	c, err := apicardclient.New(baseURL)
	require.NoError(t, err)
	return c
}

func assertSentinel(t *testing.T, gotErr, wantTarget error) {
	t.Helper()
	require.Error(t, gotErr)
	assert.ErrorIs(t, gotErr, wantTarget)
}
