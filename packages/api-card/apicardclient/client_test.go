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

// TestClient_StatusToSentinelMapping は SDK の固有責務である
// 「OpenAPI spec で宣言された 4xx/5xx status を sentinel error に変換する」契約を検証する。
// 各 endpoint について spec (data/openapi.yaml) が宣言する error status を網羅し、
// errors.Is で意図した sentinel に一致することを確認する。
func TestClient_StatusToSentinelMapping(t *testing.T) {
	cases := []struct {
		name       string
		setup      func(*apicardserverfake.Server)
		invoke     func(*apicardclient.Client) error
		wantTarget error
	}{
		// ListCards: spec 宣言は 500 のみ
		{
			name:       "ListCards: 500 を受けたとき ErrInternalServer に変換される",
			setup:      func(s *apicardserverfake.Server) { s.ListAllCardsFn = stubStatus(http.StatusInternalServerError) },
			invoke:     func(c *apicardclient.Client) error { _, err := c.ListCards(context.Background()); return err },
			wantTarget: apicardclient.ErrInternalServer,
		},

		// ListPlayerCards: spec 宣言は 401, 500
		{
			name:       "ListPlayerCards: 401 を受けたとき ErrUnauthorized に変換される",
			setup:      func(s *apicardserverfake.Server) { s.ListPlayerCardsFn = stubStatus(http.StatusUnauthorized) },
			invoke:     func(c *apicardclient.Client) error { _, err := c.ListPlayerCards(context.Background()); return err },
			wantTarget: apicardclient.ErrUnauthorized,
		},
		{
			name:       "ListPlayerCards: 500 を受けたとき ErrInternalServer に変換される",
			setup:      func(s *apicardserverfake.Server) { s.ListPlayerCardsFn = stubStatus(http.StatusInternalServerError) },
			invoke:     func(c *apicardclient.Client) error { _, err := c.ListPlayerCards(context.Background()); return err },
			wantTarget: apicardclient.ErrInternalServer,
		},

		// ListCardsWithOwnership: spec 宣言は 401, 500
		{
			name:       "ListCardsWithOwnership: 401 を受けたとき ErrUnauthorized に変換される",
			setup:      func(s *apicardserverfake.Server) { s.ListCardsWithOwnershipFn = stubStatus(http.StatusUnauthorized) },
			invoke:     func(c *apicardclient.Client) error { _, err := c.ListCardsWithOwnership(context.Background()); return err },
			wantTarget: apicardclient.ErrUnauthorized,
		},
		{
			name:       "ListCardsWithOwnership: 500 を受けたとき ErrInternalServer に変換される",
			setup:      func(s *apicardserverfake.Server) { s.ListCardsWithOwnershipFn = stubStatus(http.StatusInternalServerError) },
			invoke:     func(c *apicardclient.Client) error { _, err := c.ListCardsWithOwnership(context.Background()); return err },
			wantTarget: apicardclient.ErrInternalServer,
		},

		// ListDecks: spec 宣言は 401, 404, 500
		{
			name:       "ListDecks: 401 を受けたとき ErrUnauthorized に変換される",
			setup:      func(s *apicardserverfake.Server) { s.ListDecksFn = stubStatus(http.StatusUnauthorized) },
			invoke:     func(c *apicardclient.Client) error { _, err := c.ListDecks(context.Background()); return err },
			wantTarget: apicardclient.ErrUnauthorized,
		},
		{
			name:       "ListDecks: 404 を受けたとき ErrNotFound に変換される",
			setup:      func(s *apicardserverfake.Server) { s.ListDecksFn = stubStatus(http.StatusNotFound) },
			invoke:     func(c *apicardclient.Client) error { _, err := c.ListDecks(context.Background()); return err },
			wantTarget: apicardclient.ErrNotFound,
		},
		{
			name:       "ListDecks: 500 を受けたとき ErrInternalServer に変換される",
			setup:      func(s *apicardserverfake.Server) { s.ListDecksFn = stubStatus(http.StatusInternalServerError) },
			invoke:     func(c *apicardclient.Client) error { _, err := c.ListDecks(context.Background()); return err },
			wantTarget: apicardclient.ErrInternalServer,
		},

		// CreateDeck: spec 宣言は 400, 401, 403, 500
		{
			name:       "CreateDeck: 400 を受けたとき ErrBadRequest に変換される",
			setup:      func(s *apicardserverfake.Server) { s.CreateDeckFn = stubCreateStatus(http.StatusBadRequest) },
			invoke:     invokeCreateDeck,
			wantTarget: apicardclient.ErrBadRequest,
		},
		{
			name:       "CreateDeck: 401 を受けたとき ErrUnauthorized に変換される",
			setup:      func(s *apicardserverfake.Server) { s.CreateDeckFn = stubCreateStatus(http.StatusUnauthorized) },
			invoke:     invokeCreateDeck,
			wantTarget: apicardclient.ErrUnauthorized,
		},
		{
			name:       "CreateDeck: 403 を受けたとき ErrForbidden に変換される",
			setup:      func(s *apicardserverfake.Server) { s.CreateDeckFn = stubCreateStatus(http.StatusForbidden) },
			invoke:     invokeCreateDeck,
			wantTarget: apicardclient.ErrForbidden,
		},
		{
			name:       "CreateDeck: 500 を受けたとき ErrInternalServer に変換される",
			setup:      func(s *apicardserverfake.Server) { s.CreateDeckFn = stubCreateStatus(http.StatusInternalServerError) },
			invoke:     invokeCreateDeck,
			wantTarget: apicardclient.ErrInternalServer,
		},

		// GetDeck: spec 宣言は 400, 401, 404, 500
		{
			name:       "GetDeck: 400 を受けたとき ErrBadRequest に変換される",
			setup:      func(s *apicardserverfake.Server) { s.GetDeckFn = stubGetDeckStatus(http.StatusBadRequest) },
			invoke:     invokeGetDeck,
			wantTarget: apicardclient.ErrBadRequest,
		},
		{
			name:       "GetDeck: 401 を受けたとき ErrUnauthorized に変換される",
			setup:      func(s *apicardserverfake.Server) { s.GetDeckFn = stubGetDeckStatus(http.StatusUnauthorized) },
			invoke:     invokeGetDeck,
			wantTarget: apicardclient.ErrUnauthorized,
		},
		{
			name:       "GetDeck: 404 を受けたとき ErrNotFound に変換される",
			setup:      func(s *apicardserverfake.Server) { s.GetDeckFn = stubGetDeckStatus(http.StatusNotFound) },
			invoke:     invokeGetDeck,
			wantTarget: apicardclient.ErrNotFound,
		},
		{
			name:       "GetDeck: 500 を受けたとき ErrInternalServer に変換される",
			setup:      func(s *apicardserverfake.Server) { s.GetDeckFn = stubGetDeckStatus(http.StatusInternalServerError) },
			invoke:     invokeGetDeck,
			wantTarget: apicardclient.ErrInternalServer,
		},

		// UpdateDeck: spec 宣言は 400, 401, 403, 404, 500
		{
			name:       "UpdateDeck: 400 を受けたとき ErrBadRequest に変換される",
			setup:      func(s *apicardserverfake.Server) { s.UpdateDeckFn = stubUpdateStatus(http.StatusBadRequest) },
			invoke:     invokeUpdateDeck,
			wantTarget: apicardclient.ErrBadRequest,
		},
		{
			name:       "UpdateDeck: 401 を受けたとき ErrUnauthorized に変換される",
			setup:      func(s *apicardserverfake.Server) { s.UpdateDeckFn = stubUpdateStatus(http.StatusUnauthorized) },
			invoke:     invokeUpdateDeck,
			wantTarget: apicardclient.ErrUnauthorized,
		},
		{
			name:       "UpdateDeck: 403 を受けたとき ErrForbidden に変換される",
			setup:      func(s *apicardserverfake.Server) { s.UpdateDeckFn = stubUpdateStatus(http.StatusForbidden) },
			invoke:     invokeUpdateDeck,
			wantTarget: apicardclient.ErrForbidden,
		},
		{
			name:       "UpdateDeck: 404 を受けたとき ErrNotFound に変換される",
			setup:      func(s *apicardserverfake.Server) { s.UpdateDeckFn = stubUpdateStatus(http.StatusNotFound) },
			invoke:     invokeUpdateDeck,
			wantTarget: apicardclient.ErrNotFound,
		},
		{
			name:       "UpdateDeck: 500 を受けたとき ErrInternalServer に変換される",
			setup:      func(s *apicardserverfake.Server) { s.UpdateDeckFn = stubUpdateStatus(http.StatusInternalServerError) },
			invoke:     invokeUpdateDeck,
			wantTarget: apicardclient.ErrInternalServer,
		},

		// DeleteDeck: spec 宣言は 400, 401, 404, 500
		{
			name:       "DeleteDeck: 400 を受けたとき ErrBadRequest に変換される",
			setup:      func(s *apicardserverfake.Server) { s.DeleteDeckFn = stubDeleteStatus(http.StatusBadRequest) },
			invoke:     func(c *apicardclient.Client) error { return c.DeleteDeck(context.Background(), 1) },
			wantTarget: apicardclient.ErrBadRequest,
		},
		{
			name:       "DeleteDeck: 401 を受けたとき ErrUnauthorized に変換される",
			setup:      func(s *apicardserverfake.Server) { s.DeleteDeckFn = stubDeleteStatus(http.StatusUnauthorized) },
			invoke:     func(c *apicardclient.Client) error { return c.DeleteDeck(context.Background(), 1) },
			wantTarget: apicardclient.ErrUnauthorized,
		},
		{
			name:       "DeleteDeck: 404 を受けたとき ErrNotFound に変換される",
			setup:      func(s *apicardserverfake.Server) { s.DeleteDeckFn = stubDeleteStatus(http.StatusNotFound) },
			invoke:     func(c *apicardclient.Client) error { return c.DeleteDeck(context.Background(), 1) },
			wantTarget: apicardclient.ErrNotFound,
		},
		{
			name:       "DeleteDeck: 500 を受けたとき ErrInternalServer に変換される",
			setup:      func(s *apicardserverfake.Server) { s.DeleteDeckFn = stubDeleteStatus(http.StatusInternalServerError) },
			invoke:     func(c *apicardclient.Client) error { return c.DeleteDeck(context.Background(), 1) },
			wantTarget: apicardclient.ErrInternalServer,
		},

		// ValidateDeckForBattle: spec 宣言は 400, 401, 403, 404, 500
		// 400 だけ ErrDeckInvalid (wire 契約上「デッキ不正」を意味するため特別扱い)
		{
			name:       "ValidateDeckForBattle: 400 を受けたとき ErrBadRequest ではなく ErrDeckInvalid に変換される",
			setup:      func(s *apicardserverfake.Server) { s.ValidateDeckForBattleFn = stubValidateStatus(http.StatusBadRequest) },
			invoke:     func(c *apicardclient.Client) error { return c.ValidateDeckForBattle(context.Background(), 1) },
			wantTarget: apicardclient.ErrDeckInvalid,
		},
		{
			name:       "ValidateDeckForBattle: 401 を受けたとき ErrUnauthorized に変換される",
			setup:      func(s *apicardserverfake.Server) { s.ValidateDeckForBattleFn = stubValidateStatus(http.StatusUnauthorized) },
			invoke:     func(c *apicardclient.Client) error { return c.ValidateDeckForBattle(context.Background(), 1) },
			wantTarget: apicardclient.ErrUnauthorized,
		},
		{
			name:       "ValidateDeckForBattle: 403 を受けたとき ErrForbidden に変換される",
			setup:      func(s *apicardserverfake.Server) { s.ValidateDeckForBattleFn = stubValidateStatus(http.StatusForbidden) },
			invoke:     func(c *apicardclient.Client) error { return c.ValidateDeckForBattle(context.Background(), 1) },
			wantTarget: apicardclient.ErrForbidden,
		},
		{
			name:       "ValidateDeckForBattle: 404 を受けたとき ErrNotFound に変換される",
			setup:      func(s *apicardserverfake.Server) { s.ValidateDeckForBattleFn = stubValidateStatus(http.StatusNotFound) },
			invoke:     func(c *apicardclient.Client) error { return c.ValidateDeckForBattle(context.Background(), 1) },
			wantTarget: apicardclient.ErrNotFound,
		},
		{
			name:       "ValidateDeckForBattle: 500 を受けたとき ErrInternalServer に変換される",
			setup:      func(s *apicardserverfake.Server) { s.ValidateDeckForBattleFn = stubValidateStatus(http.StatusInternalServerError) },
			invoke:     func(c *apicardclient.Client) error { return c.ValidateDeckForBattle(context.Background(), 1) },
			wantTarget: apicardclient.ErrInternalServer,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			tc.setup(srv)

			c, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			gotErr := tc.invoke(c)
			require.Error(t, gotErr)
			assert.ErrorIs(t, gotErr, tc.wantTarget)
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

// stubStatus は body 不要の Fn (引数 0 個の endpoint 用) で固定 status を返す stub を作る。
func stubStatus(status int) func() (int, any) {
	return func() (int, any) { return status, nil }
}

// stubCreateStatus は CreateDeck endpoint 用 (request body を受ける署名) の status 固定 stub を作る。
func stubCreateStatus(status int) func(apicard.DeckCreateRequest) (int, any) {
	return func(_ apicard.DeckCreateRequest) (int, any) { return status, nil }
}

// stubUpdateStatus は UpdateDeck endpoint 用 (deckID + request body を受ける署名) の status 固定 stub を作る。
func stubUpdateStatus(status int) func(string, apicard.DeckUpdateRequest) (int, any) {
	return func(_ string, _ apicard.DeckUpdateRequest) (int, any) { return status, nil }
}

// stubGetDeckStatus / stubDeleteStatus / stubValidateStatus は deckID のみを path param に取る endpoint 用の stub。
func stubGetDeckStatus(status int) func(string) (int, any) {
	return func(_ string) (int, any) { return status, nil }
}

func stubDeleteStatus(status int) func(string) (int, any) {
	return func(_ string) (int, any) { return status, nil }
}

func stubValidateStatus(status int) func(string) (int, any) {
	return func(_ string) (int, any) { return status, nil }
}

func invokeCreateDeck(c *apicardclient.Client) error {
	_, err := c.CreateDeck(context.Background(), apicard.DeckCreateRequest{DeckName: "x"})
	return err
}

func invokeGetDeck(c *apicardclient.Client) error {
	_, _, err := c.GetDeck(context.Background(), 1)
	return err
}

func invokeUpdateDeck(c *apicardclient.Client) error {
	_, err := c.UpdateDeck(context.Background(), 1, apicard.DeckUpdateRequest{DeckName: "x"})
	return err
}
