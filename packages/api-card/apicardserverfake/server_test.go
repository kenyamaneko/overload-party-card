package apicardserverfake_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	"github.com/kenyamaneko/overload-party-card/packages/api-card/apicardserverfake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	t.Run("サーバフェイク", func(t *testing.T) {
		t.Run("Fn未設定のendpointは既定応答を返す", func(t *testing.T) {
			tests := []struct {
				name       string
				method     string
				path       string
				reqBody    []byte
				wantStatus int
				verifyBody func(t *testing.T, body []byte)
			}{
				{
					name: "ListAllCards (Fn未設定)のとき、200になる", method: http.MethodGet, path: "/internal/v1/cards", wantStatus: http.StatusOK,
					verifyBody: func(t *testing.T, body []byte) {
						var cards []*apicard.CardDefinition
						require.NoError(t, json.Unmarshal(body, &cards))
						assert.Empty(t, cards)
					},
				},
				{
					name: "ListCardsWithOwnership (Fn未設定)のとき、200になる", method: http.MethodGet, path: "/api/v1/cards/cards/with-ownership", wantStatus: http.StatusOK,
					verifyBody: func(t *testing.T, body []byte) {
						var cards []*apicard.CardWithOwnership
						require.NoError(t, json.Unmarshal(body, &cards))
						assert.Empty(t, cards)
					},
				},
				{
					name: "ListPlayerCards (Fn未設定)のとき、200になる", method: http.MethodGet, path: "/api/v1/cards/cards", wantStatus: http.StatusOK,
					verifyBody: func(t *testing.T, body []byte) {
						var cards []*apicard.PlayerCardWithDef
						require.NoError(t, json.Unmarshal(body, &cards))
						assert.Empty(t, cards)
					},
				},
				{
					name: "ListDecks (Fn未設定)のとき、200になる", method: http.MethodGet, path: "/api/v1/cards/decks", wantStatus: http.StatusOK,
					verifyBody: func(t *testing.T, body []byte) {
						var decks []*apicard.Deck
						require.NoError(t, json.Unmarshal(body, &decks))
						assert.Empty(t, decks)
					},
				},
				{
					name: "GetDeck (Fn未設定)のとき、200になる", method: http.MethodGet, path: "/api/v1/cards/decks/1", wantStatus: http.StatusOK,
					verifyBody: func(t *testing.T, body []byte) {
						var got apicardserverfake.DeckWithCardsResponse
						require.NoError(t, json.Unmarshal(body, &got))
						assert.Nil(t, got.Deck)
						assert.Empty(t, got.Cards)
					},
				},
				{
					name: "CreateDeck (Fn未設定)のとき、200になる", method: http.MethodPost, path: "/api/v1/cards/decks", reqBody: []byte(`{}`), wantStatus: http.StatusOK,
					verifyBody: func(t *testing.T, body []byte) {
						var deck apicard.Deck
						require.NoError(t, json.Unmarshal(body, &deck))
						assert.Equal(t, apicard.Deck{}, deck)
					},
				},
				{
					name: "UpdateDeck (Fn未設定)のとき、200になる", method: http.MethodPut, path: "/api/v1/cards/decks/1", reqBody: []byte(`{}`), wantStatus: http.StatusOK,
					verifyBody: func(t *testing.T, body []byte) {
						var deck apicard.Deck
						require.NoError(t, json.Unmarshal(body, &deck))
						assert.Equal(t, apicard.Deck{}, deck)
					},
				},
				{
					name: "DeleteDeck (Fn未設定)のとき、204になる", method: http.MethodDelete, path: "/api/v1/cards/decks/1", wantStatus: http.StatusNoContent,
					verifyBody: func(t *testing.T, body []byte) {
						assert.Empty(t, body)
					},
				},
				{
					name: "ValidateDeckForBattle (Fn未設定)のとき、200になる", method: http.MethodPost, path: "/api/v1/cards/decks/1/validate-for-battle", wantStatus: http.StatusOK,
					verifyBody: func(t *testing.T, body []byte) {
						assert.Empty(t, body)
					},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					srv := apicardserverfake.NewServer()
					defer srv.Close()

					req, err := http.NewRequest(tt.method, srv.URL()+tt.path, bytes.NewReader(tt.reqBody))
					require.NoError(t, err)
					req.Header.Set("Content-Type", "application/json")
					resp, err := http.DefaultClient.Do(req)
					require.NoError(t, err)
					defer resp.Body.Close()

					body, err := io.ReadAll(resp.Body)
					require.NoError(t, err)

					assert.Equal(t, tt.wantStatus, resp.StatusCode)
					tt.verifyBody(t, body)
				})
			}
		})

		t.Run("CreateDeckFnはtyped requestを受け取りtyped responseを返せる", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()

			var gotReq apicard.DeckCreateRequest
			srv.CreateDeckFn = func(req apicard.DeckCreateRequest) (int, any) {
				gotReq = req
				return http.StatusOK, apicard.Deck{DeckID: 42, DeckName: req.DeckName}
			}

			reqBody, _ := json.Marshal(apicard.DeckCreateRequest{DeckName: "my deck"})
			req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/cards/decks", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "my deck", gotReq.DeckName)

			var decoded apicard.Deck
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&decoded))
			assert.Equal(t, int64(42), decoded.DeckID)
			assert.Equal(t, "my deck", decoded.DeckName)
		})

		t.Run("ValidateDeckForBattleFnは400とerror bodyを返せる", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()

			srv.ValidateDeckForBattleFn = func(_ string) (int, any) {
				return http.StatusBadRequest, map[string]string{"error": "deck must have 30 cards"}
			}

			req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/cards/decks/1/validate-for-battle", nil)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			var errBody struct {
				Error string `json:"error"`
			}
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&errBody))
			assert.Equal(t, "deck must have 30 cards", errBody.Error)
		})

		t.Run("GetDeckFnはDeckWithCardsResponseを返せる", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()

			srv.GetDeckFn = func(_ string) (int, any) {
				return http.StatusOK, apicardserverfake.DeckWithCardsResponse{
					Deck:  &apicard.Deck{DeckID: 7, DeckName: "pvp"},
					Cards: []apicard.DeckCard{{CardID: "card-1"}},
				}
			}

			resp, err := http.Get(srv.URL() + "/api/v1/cards/decks/7")
			require.NoError(t, err)
			defer resp.Body.Close()

			var decoded apicardserverfake.DeckWithCardsResponse
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&decoded))
			require.NotNil(t, decoded.Deck)
			assert.Equal(t, int64(7), decoded.Deck.DeckID)
			require.Len(t, decoded.Cards, 1)
			assert.Equal(t, "card-1", decoded.Cards[0].CardID)
		})
	})
}
