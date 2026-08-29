package account_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/adapter/account"
)

func TestClientListPlayerFactions(t *testing.T) {
	t.Run("[account] 所持ファクション一覧の取得", func(t *testing.T) {
		t.Run("accountサービスが200とファクション一覧を含むJSONを返したとき、所持ファクション一覧の取得はそのファクション一覧をそのまま返す", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"factions":["SHE","Tenki"]}`))
			}))
			defer server.Close()
			client := account.NewClient(server.URL)

			got, err := client.ListPlayerFactions(context.Background(), "TST-0001")

			require.NoError(t, err)
			assert.Equal(t, []string{"SHE", "Tenki"}, got)
		})

		t.Run("accountサービスが200とfactionsが空配列のJSONを返したとき、所持ファクション一覧の取得は0件を返す", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"factions":[]}`))
			}))
			defer server.Close()
			client := account.NewClient(server.URL)

			got, err := client.ListPlayerFactions(context.Background(), "TST-0001")

			require.NoError(t, err)
			assert.Empty(t, got)
		})

		t.Run("accountサービスが200以外のステータス(404)を返したとき、所持ファクション一覧の取得はエラーになる", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()
			client := account.NewClient(server.URL)

			_, err := client.ListPlayerFactions(context.Background(), "TST-0001")

			assert.ErrorContains(t, err, "status 404")
		})

		t.Run("accountサービスが200とともにJSONとして解釈できないボディを返したとき、所持ファクション一覧の取得はエラーになる", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("not json"))
			}))
			defer server.Close()
			client := account.NewClient(server.URL)

			_, err := client.ListPlayerFactions(context.Background(), "TST-0001")

			assert.ErrorContains(t, err, "decode player factions")
		})

		t.Run("accountサービスへの接続自体に失敗したとき、所持ファクション一覧の取得はエラーになる", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			server.Close()
			client := account.NewClient(server.URL)

			_, err := client.ListPlayerFactions(context.Background(), "TST-0001")

			assert.ErrorContains(t, err, "call account")
		})

		t.Run("player_idにURLの予約文字を含む値を渡しても、accountサービスへのリクエストパスにはURLエンコードされた形で現れる", func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.EscapedPath()
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"factions":[]}`))
			}))
			defer server.Close()
			client := account.NewClient(server.URL)

			_, err := client.ListPlayerFactions(context.Background(), "TST-0001/TST-0002")

			require.NoError(t, err)
			assert.Equal(t, "/internal/v1/players/TST-0001%2FTST-0002/factions", gotPath)
		})
	})
}
