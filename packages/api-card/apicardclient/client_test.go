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

// 以下の TestClient_<Endpoint>_StatusMapping 群は、SDK の固有責務である
// 「OpenAPI spec で宣言された 4xx/5xx status を sentinel error に変換する」契約を
// endpoint ごとに検証する。各テスト関数は data/openapi.yaml が宣言する error status を
// 網羅し、errors.Is で意図した sentinel に一致することを確認する。

func TestClient_ListCards_StatusMapping(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		wantTarget error
	}{
		{"500 を受けたとき ErrInternalServer", http.StatusInternalServerError, apicardclient.ErrInternalServer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.ListAllCardsFn = func() (int, any) { return tc.status, nil }

			c := newTestClient(t, srv.URL())
			_, err := c.ListCards(context.Background())
			assertSentinel(t, err, tc.wantTarget)
		})
	}
}

func TestClient_ListPlayerCards_StatusMapping(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		wantTarget error
	}{
		{"401 を受けたとき ErrUnauthorized", http.StatusUnauthorized, apicardclient.ErrUnauthorized},
		{"500 を受けたとき ErrInternalServer", http.StatusInternalServerError, apicardclient.ErrInternalServer},
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
}

func TestClient_ListCardsWithOwnership_StatusMapping(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		wantTarget error
	}{
		{"401 を受けたとき ErrUnauthorized", http.StatusUnauthorized, apicardclient.ErrUnauthorized},
		{"500 を受けたとき ErrInternalServer", http.StatusInternalServerError, apicardclient.ErrInternalServer},
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
}

func TestClient_ListDecks_StatusMapping(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		wantTarget error
	}{
		{"401 を受けたとき ErrUnauthorized", http.StatusUnauthorized, apicardclient.ErrUnauthorized},
		{"404 を受けたとき ErrNotFound", http.StatusNotFound, apicardclient.ErrNotFound},
		{"500 を受けたとき ErrInternalServer", http.StatusInternalServerError, apicardclient.ErrInternalServer},
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
}

func TestClient_CreateDeck_StatusMapping(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		wantTarget error
	}{
		{"400 を受けたとき ErrBadRequest", http.StatusBadRequest, apicardclient.ErrBadRequest},
		{"401 を受けたとき ErrUnauthorized", http.StatusUnauthorized, apicardclient.ErrUnauthorized},
		{"403 を受けたとき ErrForbidden", http.StatusForbidden, apicardclient.ErrForbidden},
		{"500 を受けたとき ErrInternalServer", http.StatusInternalServerError, apicardclient.ErrInternalServer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.CreateDeckFn = func(_ apicard.DeckCreateRequest) (int, any) { return tc.status, nil }

			c := newTestClient(t, srv.URL())
			_, err := c.CreateDeck(context.Background(), apicard.DeckCreateRequest{DeckName: "x"})
			assertSentinel(t, err, tc.wantTarget)
		})
	}
}

func TestClient_GetDeck_StatusMapping(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		wantTarget error
	}{
		{"400 を受けたとき ErrBadRequest", http.StatusBadRequest, apicardclient.ErrBadRequest},
		{"401 を受けたとき ErrUnauthorized", http.StatusUnauthorized, apicardclient.ErrUnauthorized},
		{"404 を受けたとき ErrNotFound", http.StatusNotFound, apicardclient.ErrNotFound},
		{"500 を受けたとき ErrInternalServer", http.StatusInternalServerError, apicardclient.ErrInternalServer},
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
}

func TestClient_UpdateDeck_StatusMapping(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		wantTarget error
	}{
		{"400 を受けたとき ErrBadRequest", http.StatusBadRequest, apicardclient.ErrBadRequest},
		{"401 を受けたとき ErrUnauthorized", http.StatusUnauthorized, apicardclient.ErrUnauthorized},
		{"403 を受けたとき ErrForbidden", http.StatusForbidden, apicardclient.ErrForbidden},
		{"404 を受けたとき ErrNotFound", http.StatusNotFound, apicardclient.ErrNotFound},
		{"500 を受けたとき ErrInternalServer", http.StatusInternalServerError, apicardclient.ErrInternalServer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.UpdateDeckFn = func(_ string, _ apicard.DeckUpdateRequest) (int, any) { return tc.status, nil }

			c := newTestClient(t, srv.URL())
			_, err := c.UpdateDeck(context.Background(), 1, apicard.DeckUpdateRequest{DeckName: "x"})
			assertSentinel(t, err, tc.wantTarget)
		})
	}
}

func TestClient_DeleteDeck_StatusMapping(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		wantTarget error
	}{
		{"400 を受けたとき ErrBadRequest", http.StatusBadRequest, apicardclient.ErrBadRequest},
		{"401 を受けたとき ErrUnauthorized", http.StatusUnauthorized, apicardclient.ErrUnauthorized},
		{"404 を受けたとき ErrNotFound", http.StatusNotFound, apicardclient.ErrNotFound},
		{"500 を受けたとき ErrInternalServer", http.StatusInternalServerError, apicardclient.ErrInternalServer},
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
}

// ValidateDeckForBattle は 400 を ErrBadRequest ではなく ErrDeckInvalid に変換する点が
// 他 endpoint と異なる (wire 契約上「デッキ不正」を意味するため特別扱い)。
func TestClient_ValidateDeckForBattle_StatusMapping(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		wantTarget error
	}{
		{"400 を受けたとき ErrBadRequest ではなく ErrDeckInvalid", http.StatusBadRequest, apicardclient.ErrDeckInvalid},
		{"401 を受けたとき ErrUnauthorized", http.StatusUnauthorized, apicardclient.ErrUnauthorized},
		{"403 を受けたとき ErrForbidden", http.StatusForbidden, apicardclient.ErrForbidden},
		{"404 を受けたとき ErrNotFound", http.StatusNotFound, apicardclient.ErrNotFound},
		{"500 を受けたとき ErrInternalServer", http.StatusInternalServerError, apicardclient.ErrInternalServer},
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
}

// TestClient_RequestEditor は Option pattern の契約 (WithRequestEditorFn で渡した
// editor が全リクエストに適用される) を検証する。X-Internal-Auth header 注入の
// 接続点として SDK が機能することを担保する。
func TestClient_RequestEditor(t *testing.T) {
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
