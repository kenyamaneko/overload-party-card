package apicardserverfake_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	"github.com/kenyamaneko/overload-party-card/packages/api-card/apicardserverfake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	t.Run("サーバフェイク", func(t *testing.T) {
		t.Run("Fn 未設定の endpoint は既定応答を返す", func(t *testing.T) {
			tests := []struct {
				name       string
				method     string
				path       string
				reqBody    []byte
				wantStatus int
			}{
				{name: "ListAllCards (Fn 未設定) のとき、200 になる", method: http.MethodGet, path: "/internal/v1/cards", reqBody: nil, wantStatus: http.StatusOK},
				{name: "ListCardsWithOwnership (Fn 未設定) のとき、200 になる", method: http.MethodGet, path: "/api/v1/cards/cards/with-ownership", reqBody: nil, wantStatus: http.StatusOK},
				{name: "ListPlayerCards (Fn 未設定) のとき、200 になる", method: http.MethodGet, path: "/api/v1/cards/cards", reqBody: nil, wantStatus: http.StatusOK},
				{name: "ListDecks (Fn 未設定) のとき、200 になる", method: http.MethodGet, path: "/api/v1/cards/decks", reqBody: nil, wantStatus: http.StatusOK},
				{name: "GetDeck (Fn 未設定) のとき、200 になる", method: http.MethodGet, path: "/api/v1/cards/decks/1", reqBody: nil, wantStatus: http.StatusOK},
				{name: "CreateDeck (Fn 未設定) のとき、200 になる", method: http.MethodPost, path: "/api/v1/cards/decks", reqBody: []byte(`{}`), wantStatus: http.StatusOK},
				{name: "UpdateDeck (Fn 未設定) のとき、200 になる", method: http.MethodPut, path: "/api/v1/cards/decks/1", reqBody: []byte(`{}`), wantStatus: http.StatusOK},
				{name: "DeleteDeck (Fn 未設定) のとき、204 になる", method: http.MethodDelete, path: "/api/v1/cards/decks/1", reqBody: nil, wantStatus: http.StatusNoContent},
				{name: "ValidateDeckForBattle (Fn 未設定) のとき、200 になる", method: http.MethodPost, path: "/api/v1/cards/decks/1/validate-for-battle", reqBody: nil, wantStatus: http.StatusOK},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					srv := apicardserverfake.NewServer()
					defer srv.Close()

					req, _ := http.NewRequest(tt.method, srv.URL()+tt.path, bytes.NewReader(tt.reqBody))
					req.Header.Set("Content-Type", "application/json")
					resp, err := http.DefaultClient.Do(req)
					require.NoError(t, err)
					defer resp.Body.Close()

					assert.Equal(t, tt.wantStatus, resp.StatusCode)
				})
			}
		})

		t.Run("CreateDeckFn は typed request を受け取り typed response を返せる", func(t *testing.T) {
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

		t.Run("ValidateDeckForBattleFn は 400 と error body を返せる", func(t *testing.T) {
			// cardclient 側が status code + error body message から ErrDeckInvalid を生成するための契約を固定する。
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

		t.Run("GetDeckFn は DeckWithCardsResponse を返せる", func(t *testing.T) {
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
