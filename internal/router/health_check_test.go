package router_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/router/routertest"
)

func TestHealthCheck(t *testing.T) {
	t.Run("[死活監視]ヘルスチェック", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("X-Internal-Authヘッダを付けずに要求すると、応答のstatusはokになる", func(t *testing.T) {
				h := routertest.NewHarness(t)

				recorder := h.SendRequestWithoutToken(http.MethodGet, "/health", "")

				assert.Equal(t, "ok", routertest.DecodeObject(t, recorder)["status"])
			})
		})
	})
}
