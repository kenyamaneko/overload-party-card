package apicardclienttest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	"github.com/kenyamaneko/overload-party-card/packages/api-card/apicardclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Operation は apicardclient の公開操作を 1 つ呼び出す。
type Operation struct {
	Name string
	Call func(ctx context.Context, client *apicardclient.Client) error
}

// AllOperations は apicardclient の公開操作すべて。
var AllOperations = []Operation{
	{
		Name: "GetHealth",
		Call: func(ctx context.Context, client *apicardclient.Client) error {
			_, err := client.GetHealth(ctx)
			return err
		},
	},
	{
		Name: "ListPlayerCards",
		Call: func(ctx context.Context, client *apicardclient.Client) error {
			_, err := client.ListPlayerCards(ctx)
			return err
		},
	},
	{
		Name: "ListCardsWithOwnership",
		Call: func(ctx context.Context, client *apicardclient.Client) error {
			_, err := client.ListCardsWithOwnership(ctx)
			return err
		},
	},
	{
		Name: "ListDecks",
		Call: func(ctx context.Context, client *apicardclient.Client) error {
			_, err := client.ListDecks(ctx)
			return err
		},
	},
	{
		Name: "GetDeck",
		Call: func(ctx context.Context, client *apicardclient.Client) error {
			_, err := client.GetDeck(ctx, 42)
			return err
		},
	},
	{
		Name: "CreateDeck",
		Call: func(ctx context.Context, client *apicardclient.Client) error {
			_, err := client.CreateDeck(ctx, apicard.DeckCreateRequest{DeckName: "TSTデッキ", Faction: "Tenki"})
			return err
		},
	},
	{
		Name: "UpdateDeck",
		Call: func(ctx context.Context, client *apicardclient.Client) error {
			_, err := client.UpdateDeck(ctx, 7, apicard.DeckUpdateRequest{DeckName: "TSTデッキ", Faction: "Tenki"})
			return err
		},
	},
	{
		Name: "DeleteDeck",
		Call: func(ctx context.Context, client *apicardclient.Client) error {
			return client.DeleteDeck(ctx, 7)
		},
	},
	{
		Name: "ValidateDeckForBattle",
		Call: func(ctx context.Context, client *apicardclient.Client) error {
			return client.ValidateDeckForBattle(ctx, 7)
		},
	},
}

// ReceivedRequest は偽のカードAPIに届いた要求のメソッドとパス。
type ReceivedRequest struct {
	Method string
	Path   string
}

// RequestRecorder は偽のカードAPIに届いた要求を届いた順に記録する。
type RequestRecorder struct {
	mu       sync.Mutex
	received []ReceivedRequest
}

// Record は要求を記録してから next に処理を渡すハンドラを返す。
func (r *RequestRecorder) Record(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		r.received = append(r.received, ReceivedRequest{Method: req.Method, Path: req.URL.Path})
		r.mu.Unlock()
		next(w, req)
	}
}

// Requests は記録した要求を届いた順に返す。
func (r *RequestRecorder) Requests() []ReceivedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ReceivedRequest(nil), r.received...)
}

// FailingHTTPClient は常に Err を返す HTTP クライアント。
type FailingHTTPClient struct {
	Err error
}

// Do は要求を送らず Err を返す。
func (f FailingHTTPClient) Do(*http.Request) (*http.Response, error) {
	return nil, f.Err
}

// RespondJSON は本文が JSON の応答を返すハンドラを作る。
func RespondJSON(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

// RespondText は本文が text/plain の応答を返すハンドラを作る。
func RespondText(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

// RespondEmpty は本文なしの応答を返すハンドラを作る。
func RespondEmpty(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	}
}

// NewClientFor は handler を処理する偽のカードAPIに接続するクライアントを作る。
func NewClientFor(t *testing.T, handler http.HandlerFunc) *apicardclient.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := apicardclient.New(server.URL)
	require.NoError(t, err)
	return client
}

// AssertUnexpectedResponseError は err が予期しない応答を示すエラーであることを確かめる。
func AssertUnexpectedResponseError(t *testing.T, err error, msgAndArgs ...any) {
	t.Helper()
	assert.ErrorIs(t, err, apicardclient.ErrUnexpectedResponse, msgAndArgs...)
}

// CollectCountsByCardID は所持カード一覧をカードIDごとの所持枚数にまとめる。
func CollectCountsByCardID(cards []apicard.PlayerCardWithDef) map[string]int {
	counts := map[string]int{}
	for _, card := range cards {
		counts[card.CardID] = card.Count
	}
	return counts
}

// CollectOwnershipByCardID は所持状態つきカード定義一覧をカードIDごとの所持状態にまとめる。
func CollectOwnershipByCardID(cards []apicard.CardWithOwnership) map[string]bool {
	ownership := map[string]bool{}
	for _, card := range cards {
		ownership[card.CardID] = card.IsOwned
	}
	return ownership
}

// ListDeckIDs はデッキ一覧のデッキIDを一覧の順に返す。
func ListDeckIDs(decks []apicard.Deck) []int64 {
	ids := []int64{}
	for _, deck := range decks {
		ids = append(ids, deck.DeckID)
	}
	return ids
}

// CollectDeckCardCounts はデッキのカード構成をカードIDごとの枚数にまとめる。
func CollectDeckCardCounts(deck *apicard.Deck) map[string]int {
	counts := map[string]int{}
	for _, deckCard := range *deck.DeckCards {
		counts[deckCard.CardID] = deckCard.Count
	}
	return counts
}
