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

// TestClient_HappyPath は、wire 契約 (OpenAPI spec) で定義された 2xx status と
// schema 付き response body を SDK が wire 型で返すことを検証する。
func TestClient_HappyPath(t *testing.T) {
	deckSample := apicard.Deck{
		PlayerID: "p1", DeckID: 42, DeckName: "sample", IsValid: true,
	}
	cards := []apicard.DeckCard{
		{PlayerID: "p1", DeckID: 42, CardID: "c1", ArtNo: 1, Count: 3},
	}

	cases := []struct {
		name   string
		setup  func(*apicardserverfake.Server)
		invoke func(*testing.T, *apicardclient.Client)
	}{
		{
			name: "ListCards 200 returns slice",
			setup: func(s *apicardserverfake.Server) {
				s.ListAllCardsFn = func() (int, any) {
					return http.StatusOK, []apicard.CardDefinition{{CardID: "c1"}}
				}
			},
			invoke: func(t *testing.T, c *apicardclient.Client) {
				got, err := c.ListCards(context.Background())
				require.NoError(t, err)
				assert.Len(t, got, 1)
				assert.Equal(t, "c1", got[0].CardID)
			},
		},
		{
			name: "ListPlayerCards 200 returns slice",
			setup: func(s *apicardserverfake.Server) {
				s.ListPlayerCardsFn = func() (int, any) {
					return http.StatusOK, []apicard.PlayerCardWithDef{{CardID: "c1", Count: 2}}
				}
			},
			invoke: func(t *testing.T, c *apicardclient.Client) {
				got, err := c.ListPlayerCards(context.Background())
				require.NoError(t, err)
				assert.Len(t, got, 1)
			},
		},
		{
			name: "ListCardsWithOwnership 200 returns slice",
			setup: func(s *apicardserverfake.Server) {
				s.ListCardsWithOwnershipFn = func() (int, any) {
					return http.StatusOK, []apicard.CardWithOwnership{{CardID: "c1", IsOwned: true}}
				}
			},
			invoke: func(t *testing.T, c *apicardclient.Client) {
				got, err := c.ListCardsWithOwnership(context.Background())
				require.NoError(t, err)
				assert.Len(t, got, 1)
				assert.True(t, got[0].IsOwned)
			},
		},
		{
			name: "ListDecks 200 returns slice",
			setup: func(s *apicardserverfake.Server) {
				s.ListDecksFn = func() (int, any) {
					return http.StatusOK, []apicard.Deck{deckSample}
				}
			},
			invoke: func(t *testing.T, c *apicardclient.Client) {
				got, err := c.ListDecks(context.Background())
				require.NoError(t, err)
				assert.Len(t, got, 1)
				assert.Equal(t, int64(42), got[0].DeckID)
			},
		},
		{
			name: "GetDeck 200 returns deck and cards",
			setup: func(s *apicardserverfake.Server) {
				s.GetDeckFn = func(_ string) (int, any) {
					return http.StatusOK, apicardserverfake.DeckWithCardsResponse{Deck: &deckSample, Cards: cards}
				}
			},
			invoke: func(t *testing.T, c *apicardclient.Client) {
				deck, gotCards, err := c.GetDeck(context.Background(), 42)
				require.NoError(t, err)
				require.NotNil(t, deck)
				assert.Equal(t, int64(42), deck.DeckID)
				assert.Len(t, gotCards, 1)
			},
		},
		{
			name: "CreateDeck 201 returns created deck",
			setup: func(s *apicardserverfake.Server) {
				s.CreateDeckFn = func(_ apicard.DeckCreateRequest) (int, any) {
					return http.StatusCreated, deckSample
				}
			},
			invoke: func(t *testing.T, c *apicardclient.Client) {
				got, err := c.CreateDeck(context.Background(), apicard.DeckCreateRequest{DeckName: "sample"})
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, int64(42), got.DeckID)
			},
		},
		{
			name: "UpdateDeck 200 returns updated deck",
			setup: func(s *apicardserverfake.Server) {
				s.UpdateDeckFn = func(_ string, _ apicard.DeckUpdateRequest) (int, any) {
					return http.StatusOK, deckSample
				}
			},
			invoke: func(t *testing.T, c *apicardclient.Client) {
				got, err := c.UpdateDeck(context.Background(), 42, apicard.DeckUpdateRequest{DeckName: "updated"})
				require.NoError(t, err)
				require.NotNil(t, got)
			},
		},
		{
			name: "DeleteDeck 204 returns nil",
			setup: func(s *apicardserverfake.Server) {
				s.DeleteDeckFn = func(_ string) (int, any) { return http.StatusNoContent, nil }
			},
			invoke: func(t *testing.T, c *apicardclient.Client) {
				err := c.DeleteDeck(context.Background(), 42)
				require.NoError(t, err)
			},
		},
		{
			name: "ValidateDeckForBattle 200 returns nil",
			setup: func(s *apicardserverfake.Server) {
				s.ValidateDeckForBattleFn = func(_ string) (int, any) { return http.StatusOK, nil }
			},
			invoke: func(t *testing.T, c *apicardclient.Client) {
				err := c.ValidateDeckForBattle(context.Background(), 42)
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			tc.setup(srv)

			c, err := apicardclient.New(srv.URL())
			require.NoError(t, err)
			tc.invoke(t, c)
		})
	}
}

// TestClient_ErrorMapping は、wire 契約で定義された 4xx/5xx status を
// SDK が sentinel error (errors.Is で分岐可能) に変換することを検証する。
// 各 status code に対し、代表的な endpoint で挙動を確認する。
func TestClient_ErrorMapping(t *testing.T) {
	cases := []struct {
		name       string
		setup      func(*apicardserverfake.Server)
		invoke     func(*apicardclient.Client) error
		wantTarget error
	}{
		{
			name: "GetDeck 404 → ErrNotFound",
			setup: func(s *apicardserverfake.Server) {
				s.GetDeckFn = func(_ string) (int, any) { return http.StatusNotFound, nil }
			},
			invoke:     func(c *apicardclient.Client) error { _, _, err := c.GetDeck(context.Background(), 1); return err },
			wantTarget: apicardclient.ErrNotFound,
		},
		{
			name: "GetDeck 401 → ErrUnauthorized",
			setup: func(s *apicardserverfake.Server) {
				s.GetDeckFn = func(_ string) (int, any) { return http.StatusUnauthorized, nil }
			},
			invoke:     func(c *apicardclient.Client) error { _, _, err := c.GetDeck(context.Background(), 1); return err },
			wantTarget: apicardclient.ErrUnauthorized,
		},
		{
			name: "CreateDeck 403 → ErrForbidden",
			setup: func(s *apicardserverfake.Server) {
				s.CreateDeckFn = func(_ apicard.DeckCreateRequest) (int, any) { return http.StatusForbidden, nil }
			},
			invoke: func(c *apicardclient.Client) error {
				_, err := c.CreateDeck(context.Background(), apicard.DeckCreateRequest{DeckName: "x"})
				return err
			},
			wantTarget: apicardclient.ErrForbidden,
		},
		{
			name: "CreateDeck 400 → ErrBadRequest",
			setup: func(s *apicardserverfake.Server) {
				s.CreateDeckFn = func(_ apicard.DeckCreateRequest) (int, any) { return http.StatusBadRequest, nil }
			},
			invoke: func(c *apicardclient.Client) error {
				_, err := c.CreateDeck(context.Background(), apicard.DeckCreateRequest{DeckName: "x"})
				return err
			},
			wantTarget: apicardclient.ErrBadRequest,
		},
		{
			name: "ListCards 500 → ErrInternalServer",
			setup: func(s *apicardserverfake.Server) {
				s.ListAllCardsFn = func() (int, any) { return http.StatusInternalServerError, nil }
			},
			invoke:     func(c *apicardclient.Client) error { _, err := c.ListCards(context.Background()); return err },
			wantTarget: apicardclient.ErrInternalServer,
		},
		{
			name: "DeleteDeck 404 → ErrNotFound",
			setup: func(s *apicardserverfake.Server) {
				s.DeleteDeckFn = func(_ string) (int, any) { return http.StatusNotFound, nil }
			},
			invoke:     func(c *apicardclient.Client) error { return c.DeleteDeck(context.Background(), 1) },
			wantTarget: apicardclient.ErrNotFound,
		},
		{
			name: "ValidateDeckForBattle 400 → ErrDeckInvalid (NOT ErrBadRequest)",
			setup: func(s *apicardserverfake.Server) {
				s.ValidateDeckForBattleFn = func(_ string) (int, any) { return http.StatusBadRequest, nil }
			},
			invoke:     func(c *apicardclient.Client) error { return c.ValidateDeckForBattle(context.Background(), 1) },
			wantTarget: apicardclient.ErrDeckInvalid,
		},
		{
			name: "ValidateDeckForBattle 404 → ErrNotFound",
			setup: func(s *apicardserverfake.Server) {
				s.ValidateDeckForBattleFn = func(_ string) (int, any) { return http.StatusNotFound, nil }
			},
			invoke:     func(c *apicardclient.Client) error { return c.ValidateDeckForBattle(context.Background(), 1) },
			wantTarget: apicardclient.ErrNotFound,
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

// TestClient_RequestEditor は、WithRequestEditorFn で渡した editor が
// 全リクエストに適用されることを検証する。X-Internal-Auth ヘッダ注入の接続点として機能することを担保する。
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
