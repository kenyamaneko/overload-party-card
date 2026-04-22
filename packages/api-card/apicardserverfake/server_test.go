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

// Fn 未設定の endpoint は既定応答を返す (空配列 or 空 struct + 200、または 204)。
func TestServer_DefaultResponses(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		reqBody    string
		wantStatus int
	}{
		{name: "ListAllCards 既定は空配列", method: http.MethodGet, path: "/internal/v1/cards", wantStatus: http.StatusOK},
		{name: "ListCardsWithOwnership 既定は空配列", method: http.MethodGet, path: "/internal/v1/players/p-1/cards/with-ownership", wantStatus: http.StatusOK},
		{name: "ListPlayerCards 既定は空配列", method: http.MethodGet, path: "/internal/v1/players/p-1/cards", wantStatus: http.StatusOK},
		{name: "ListDecks 既定は空配列", method: http.MethodGet, path: "/internal/v1/players/p-1/decks", wantStatus: http.StatusOK},
		{name: "GetDeck 既定は空 DeckWithCardsResponse", method: http.MethodGet, path: "/internal/v1/players/p-1/decks/1", wantStatus: http.StatusOK},
		{name: "CreateDeck 既定は空 Deck", method: http.MethodPost, path: "/internal/v1/players/p-1/decks", reqBody: `{}`, wantStatus: http.StatusOK},
		{name: "UpdateDeck 既定は空 Deck", method: http.MethodPut, path: "/internal/v1/players/p-1/decks/1", reqBody: `{}`, wantStatus: http.StatusOK},
		{name: "DeleteDeck 既定は 204", method: http.MethodDelete, path: "/internal/v1/players/p-1/decks/1", wantStatus: http.StatusNoContent},
		{name: "ValidateDeckForBattle 既定は 200", method: http.MethodPost, path: "/internal/v1/players/p-1/decks/1/validate-for-battle", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()

			var body io.Reader
			if tt.reqBody != "" {
				body = bytes.NewReader([]byte(tt.reqBody))
			}
			req, _ := http.NewRequest(tt.method, srv.URL()+tt.path, body)
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)
		})
	}
}

// CreateDeckFn は typed request を受け取って typed response を返せる。
func TestServer_CreateDeckFn_RoundTrip(t *testing.T) {
	srv := apicardserverfake.NewServer()
	defer srv.Close()

	var gotPlayerID string
	var gotReq apicard.DeckCreateRequest
	srv.CreateDeckFn = func(playerID string, req apicard.DeckCreateRequest) (int, any) {
		gotPlayerID = playerID
		gotReq = req
		return http.StatusOK, apicard.Deck{DeckID: 42, DeckName: req.DeckName}
	}

	reqBody, _ := json.Marshal(apicard.DeckCreateRequest{DeckName: "my deck"})
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/internal/v1/players/p-1/decks", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "p-1", gotPlayerID)
	assert.Equal(t, "my deck", gotReq.DeckName)

	var decoded apicard.Deck
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&decoded))
	assert.Equal(t, int64(42), decoded.DeckID)
	assert.Equal(t, "my deck", decoded.DeckName)
}

// ValidateDeckForBattleFn で 400 ErrDeckInvalid を擬似できる。cardclient 側が
// status code + error body message から ErrDeckInvalid を生成するための契約を固定する。
func TestServer_ValidateDeckForBattleFn_CanReturn400(t *testing.T) {
	srv := apicardserverfake.NewServer()
	defer srv.Close()

	srv.ValidateDeckForBattleFn = func(_, _ string) (int, any) {
		return http.StatusBadRequest, map[string]string{"error": "deck must have 30 cards"}
	}

	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/internal/v1/players/p-1/decks/1/validate-for-battle", nil)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	var errBody struct {
		Error string `json:"error"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errBody))
	assert.Equal(t, "deck must have 30 cards", errBody.Error)
}

// GetDeckFn で DeckWithCardsResponse を返せる (wrapper 型の round trip)。
func TestServer_GetDeckFn_ReturnsWrappedResponse(t *testing.T) {
	srv := apicardserverfake.NewServer()
	defer srv.Close()

	srv.GetDeckFn = func(_, _ string) (int, any) {
		return http.StatusOK, apicardserverfake.DeckWithCardsResponse{
			Deck:  &apicard.Deck{DeckID: 7, DeckName: "pvp"},
			Cards: []apicard.DeckCard{{CardID: "card-1"}},
		}
	}

	resp, err := http.Get(srv.URL() + "/internal/v1/players/p-1/decks/7")
	require.NoError(t, err)
	defer resp.Body.Close()

	var decoded apicardserverfake.DeckWithCardsResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&decoded))
	require.NotNil(t, decoded.Deck)
	assert.Equal(t, int64(7), decoded.Deck.DeckID)
	require.Len(t, decoded.Cards, 1)
	assert.Equal(t, "card-1", decoded.Cards[0].CardID)
}
