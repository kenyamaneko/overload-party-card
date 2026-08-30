package apicardclient_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	"github.com/kenyamaneko/overload-party-card/packages/api-card/apicardclient"
	"github.com/kenyamaneko/overload-party-card/packages/api-card/apicardserverfake"
)

// stubDoer は apicard.HttpRequestDoer を実装し、常に固定レスポンスを返す。
type stubDoer struct {
	response *http.Response
}

func (d *stubDoer) Do(_ *http.Request) (*http.Response, error) {
	return d.response, nil
}

func jsonResponse(t *testing.T, status int, body any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	require.NoError(t, err)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(data)),
	}
}

func TestClient(t *testing.T) {
	t.Run("HTTP クライアントの差し替え", func(t *testing.T) {
		t.Run("差し替えたクライアントが返す応答が、そのまま呼び出し結果に反映される", func(t *testing.T) {
			want := apicard.HealthResponse{Status: "dummy-status"}
			doer := &stubDoer{response: jsonResponse(t, http.StatusOK, want)}

			client, err := apicardclient.New("http://dummy.invalid", apicardclient.WithHTTPClient(doer))
			require.NoError(t, err)

			got, err := client.GetHealth(context.Background())

			require.NoError(t, err)
			assert.Equal(t, want, *got)
		})
	})

	t.Run("送信リクエストを加工する RequestEditor の登録", func(t *testing.T) {
		t.Run("RequestEditorを登録すると、送信するリクエストにその加工が適用される", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()

			client, err := apicardclient.New(srv.URL(), apicardclient.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
				req.Header.Set("X-Test-Editor", "applied")
				return nil
			}))
			require.NoError(t, err)

			_, err = client.ListDecks(context.Background())

			require.NoError(t, err)
			assert.Equal(t, "applied", srv.LastRequestHeader().Get("X-Test-Editor"))
		})
	})

	t.Run("HTTP status から sentinel error への変換", func(t *testing.T) {
		tests := []struct {
			name    string
			status  int
			wantErr error
		}{
			{"status 400 が返るとき、ErrBadRequestに一致するエラーを返す", http.StatusBadRequest, apicardclient.ErrBadRequest},
			{"status 401 が返るとき、ErrUnauthorizedに一致するエラーを返す", http.StatusUnauthorized, apicardclient.ErrUnauthorized},
			{"status 403 が返るとき、ErrForbiddenに一致するエラーを返す", http.StatusForbidden, apicardclient.ErrForbidden},
			{"status 404 が返るとき、ErrNotFoundに一致するエラーを返す", http.StatusNotFound, apicardclient.ErrNotFound},
			{"status 500 が返るとき、ErrInternalServerに一致するエラーを返す", http.StatusInternalServerError, apicardclient.ErrInternalServer},
			{"status 599 が返るとき、ErrInternalServerに一致するエラーを返す", 599, apicardclient.ErrInternalServer},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				srv := apicardserverfake.NewServer()
				defer srv.Close()
				srv.ListDecksFn = func() (int, any) { return tt.status, nil }

				client, err := apicardclient.New(srv.URL())
				require.NoError(t, err)

				_, err = client.ListDecks(context.Background())

				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			})
		}

		t.Run("status 499 が返るとき、定義済みの一般エラー系 sentinel(ErrBadRequest/ErrUnauthorized/ErrForbidden/ErrNotFound/ErrInternalServer)のいずれにも一致せず、エラーの内容から応答のstatus code (499) を識別できるエラーを返す", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.ListDecksFn = func() (int, any) { return 499, nil }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			_, err = client.ListDecks(context.Background())

			require.Error(t, err)
			assert.False(t, errors.Is(err, apicardclient.ErrBadRequest))
			assert.False(t, errors.Is(err, apicardclient.ErrUnauthorized))
			assert.False(t, errors.Is(err, apicardclient.ErrForbidden))
			assert.False(t, errors.Is(err, apicardclient.ErrNotFound))
			assert.False(t, errors.Is(err, apicardclient.ErrInternalServer))
			assert.Contains(t, err.Error(), "499")
		})
	})

	t.Run("デッキ一覧取得", func(t *testing.T) {
		t.Run("status 200 が返るとき、応答本文のデッキ一覧をそのまま返す", func(t *testing.T) {
			want := []apicard.Deck{{DeckID: 1, DeckName: "dummy-deck", PlayerID: "dummy-player"}}
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.ListDecksFn = func() (int, any) { return http.StatusOK, want }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			got, err := client.ListDecks(context.Background())

			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	})

	t.Run("ヘルスチェック", func(t *testing.T) {
		t.Run("status 200 が返るとき、応答本文のstatusフィールドを含むHealthResponseを返す", func(t *testing.T) {
			want := apicard.HealthResponse{Status: "healthy"}
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.GetHealthFn = func() (int, any) { return http.StatusOK, want }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			got, err := client.GetHealth(context.Background())

			require.NoError(t, err)
			assert.Equal(t, want, *got)
		})
	})

	t.Run("カード定義一覧取得", func(t *testing.T) {
		t.Run("status 200 が返るとき、応答本文のカード定義一覧をそのまま返す", func(t *testing.T) {
			want := []apicard.CardDefinition{{CardID: "dummy-card-id", CardName: "dummy-card-name", Stats: json.RawMessage("{}")}}
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.ListAllCardsFn = func() (int, any) { return http.StatusOK, want }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			got, err := client.ListCards(context.Background())

			require.NoError(t, err)
			assert.Equal(t, want, got)
		})

		t.Run("status 500 が返るとき、ErrInternalServerに一致するエラーを返す", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.ListAllCardsFn = func() (int, any) { return http.StatusInternalServerError, nil }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			_, err = client.ListCards(context.Background())

			require.Error(t, err)
			assert.ErrorIs(t, err, apicardclient.ErrInternalServer)
		})
	})

	t.Run("所持カード一覧取得", func(t *testing.T) {
		t.Run("status 200 が返るとき、応答本文の所持カード一覧をそのまま返す", func(t *testing.T) {
			want := []apicard.PlayerCardWithDef{{CardID: "dummy-card-id", CardName: "dummy-card-name", Count: 1, Stats: json.RawMessage("{}")}}
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.ListPlayerCardsFn = func() (int, any) { return http.StatusOK, want }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			got, err := client.ListPlayerCards(context.Background())

			require.NoError(t, err)
			assert.Equal(t, want, got)
		})

		t.Run("status 401 が返るとき、ErrUnauthorizedに一致するエラーを返す", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.ListPlayerCardsFn = func() (int, any) { return http.StatusUnauthorized, nil }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			_, err = client.ListPlayerCards(context.Background())

			require.Error(t, err)
			assert.ErrorIs(t, err, apicardclient.ErrUnauthorized)
		})
	})

	t.Run("所持状態付きカード一覧取得", func(t *testing.T) {
		t.Run("status 200 が返るとき、応答本文の所持状態付きカード一覧をそのまま返す", func(t *testing.T) {
			want := []apicard.CardWithOwnership{{CardID: "dummy-card-id", CardName: "dummy-card-name", IsOwned: true, Stats: json.RawMessage("{}")}}
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.ListCardsWithOwnershipFn = func() (int, any) { return http.StatusOK, want }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			got, err := client.ListCardsWithOwnership(context.Background())

			require.NoError(t, err)
			assert.Equal(t, want, got)
		})

		t.Run("status 401 が返るとき、ErrUnauthorizedに一致するエラーを返す", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.ListCardsWithOwnershipFn = func() (int, any) { return http.StatusUnauthorized, nil }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			_, err = client.ListCardsWithOwnership(context.Background())

			require.Error(t, err)
			assert.ErrorIs(t, err, apicardclient.ErrUnauthorized)
		})
	})

	t.Run("デッキ取得", func(t *testing.T) {
		t.Run("status 200 が返るとき、応答本文のデッキ(デッキ構成カードを含む)をそのまま返す", func(t *testing.T) {
			deckCards := []apicard.DeckCard{{DeckID: 1, CardID: "dummy-card-id", Count: 2}}
			want := apicard.Deck{DeckID: 1, DeckName: "dummy-deck", PlayerID: "dummy-player", DeckCards: &deckCards}
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.GetDeckFn = func(_ string) (int, any) { return http.StatusOK, want }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			got, err := client.GetDeck(context.Background(), 1)

			require.NoError(t, err)
			assert.Equal(t, want, *got)
		})

		t.Run("status 404 が返るとき、ErrNotFoundに一致するエラーを返す", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.GetDeckFn = func(_ string) (int, any) { return http.StatusNotFound, nil }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			_, err = client.GetDeck(context.Background(), 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, apicardclient.ErrNotFound)
		})
	})

	t.Run("デッキ作成", func(t *testing.T) {
		t.Run("status 201 が返るとき、応答本文のデッキを返す", func(t *testing.T) {
			want := apicard.Deck{DeckID: 1, DeckName: "dummy-deck", PlayerID: "dummy-player"}
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.CreateDeckFn = func(_ apicard.DeckCreateRequest) (int, any) { return http.StatusCreated, want }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			got, err := client.CreateDeck(context.Background(), apicard.DeckCreateRequest{DeckName: "dummy-deck"})

			require.NoError(t, err)
			assert.Equal(t, want, *got)
		})

		t.Run("status 403 が返るとき、ErrForbiddenに一致するエラーを返す", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.CreateDeckFn = func(_ apicard.DeckCreateRequest) (int, any) { return http.StatusForbidden, nil }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			_, err = client.CreateDeck(context.Background(), apicard.DeckCreateRequest{DeckName: "dummy-deck"})

			require.Error(t, err)
			assert.ErrorIs(t, err, apicardclient.ErrForbidden)
		})
	})

	t.Run("デッキ更新", func(t *testing.T) {
		t.Run("status 200 が返るとき、応答本文のデッキを返す", func(t *testing.T) {
			want := apicard.Deck{DeckID: 1, DeckName: "dummy-deck-updated", PlayerID: "dummy-player"}
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.UpdateDeckFn = func(_ string, _ apicard.DeckUpdateRequest) (int, any) { return http.StatusOK, want }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			got, err := client.UpdateDeck(context.Background(), 1, apicard.DeckUpdateRequest{DeckName: "dummy-deck-updated"})

			require.NoError(t, err)
			assert.Equal(t, want, *got)
		})

		t.Run("status 400 が返るとき、ErrBadRequestに一致するエラーを返す", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.UpdateDeckFn = func(_ string, _ apicard.DeckUpdateRequest) (int, any) { return http.StatusBadRequest, nil }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			_, err = client.UpdateDeck(context.Background(), 1, apicard.DeckUpdateRequest{DeckName: "dummy-deck-updated"})

			require.Error(t, err)
			assert.ErrorIs(t, err, apicardclient.ErrBadRequest)
		})
	})

	t.Run("デッキ削除", func(t *testing.T) {
		t.Run("status 204 が返るとき、エラーを返さない", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.DeleteDeckFn = func(_ string) (int, any) { return http.StatusNoContent, nil }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			err = client.DeleteDeck(context.Background(), 1)

			assert.NoError(t, err)
		})

		t.Run("status 400 が返るとき、ErrBadRequestに一致するエラーを返す", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.DeleteDeckFn = func(_ string) (int, any) { return http.StatusBadRequest, nil }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			err = client.DeleteDeck(context.Background(), 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, apicardclient.ErrBadRequest)
		})
	})

	t.Run("デッキのバトル使用可否検証", func(t *testing.T) {
		t.Run("status 200 が返るとき、エラーを返さない", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.ValidateDeckForBattleFn = func(_ string) (int, any) { return http.StatusOK, nil }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			err = client.ValidateDeckForBattle(context.Background(), 1)

			assert.NoError(t, err)
		})

		t.Run("status 400 が返るとき、ErrDeckInvalidに一致するエラーを返し、そのエラーの内容には応答本文に設定した文字列が含まれる", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.ValidateDeckForBattleFn = func(_ string) (int, any) {
				return http.StatusBadRequest, "dummy-invalid-reason"
			}

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			err = client.ValidateDeckForBattle(context.Background(), 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, apicardclient.ErrDeckInvalid)
			assert.Contains(t, err.Error(), "dummy-invalid-reason")
		})

		t.Run("status 404 が返るとき、ErrNotFoundに一致するエラーを返す", func(t *testing.T) {
			srv := apicardserverfake.NewServer()
			defer srv.Close()
			srv.ValidateDeckForBattleFn = func(_ string) (int, any) { return http.StatusNotFound, nil }

			client, err := apicardclient.New(srv.URL())
			require.NoError(t, err)

			err = client.ValidateDeckForBattle(context.Background(), 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, apicardclient.ErrNotFound)
		})
	})
}
