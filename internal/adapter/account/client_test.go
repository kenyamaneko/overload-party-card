package account

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListPlayerFactions(t *testing.T) {
	t.Run("プレイヤー所持ファクションの取得", func(t *testing.T) {
		t.Run("200応答のとき、内部エンドポイントへGETしfactionsを返す", func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/internal/v1/players/p-1/factions", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"factions":["SHE","Tenki"]}`))
			}))
			defer srv.Close()

			got, err := NewClient(srv.URL).ListPlayerFactions(context.Background(), "p-1")

			require.NoError(t, err)
			assert.Equal(t, []string{"SHE", "Tenki"}, got)
		})

		t.Run("非200応答 (500)のとき、エラーになる", func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer srv.Close()

			_, err := NewClient(srv.URL).ListPlayerFactions(context.Background(), "p-1")

			require.Error(t, err)
		})
	})
}
