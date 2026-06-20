package account

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListPlayerFactions は、内部エンドポイントへ正しいパスで GET し、
// レスポンスの factions を返すことを検証します。
func TestListPlayerFactions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/v1/players/p-1/factions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"factions":["SHE","Tenki"]}`))
	}))
	defer srv.Close()

	got, err := NewClient(srv.URL).ListPlayerFactions(context.Background(), "p-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"SHE", "Tenki"}, got)
}

// TestListPlayerFactions_NonOKStatus は、非 200 応答をエラーにすることを検証します。
func TestListPlayerFactions_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL).ListPlayerFactions(context.Background(), "p-1")

	require.Error(t, err)
}
