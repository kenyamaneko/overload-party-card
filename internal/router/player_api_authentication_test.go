package router_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/router/routertest"
)

func TestPlayerAPIAuthentication(t *testing.T) {
	t.Run("[サービス間認証]プレイヤー向けAPIの認証", func(t *testing.T) {
		t.Run("異常系", func(t *testing.T) {
			t.Run("X-Internal-Authヘッダを付けないとき", func(t *testing.T) {
				missingHeaderTests := []struct {
					name   string
					method string
					path   string
				}{
					{"所持カード一覧の取得を要求すると、401の応答になり、本文のerror項目はヘッダが無いことを示す", http.MethodGet, routertest.PlayerCardsPath},
					{"所持状態付きカード一覧の取得を要求すると、401の応答になり、本文のerror項目はヘッダが無いことを示す", http.MethodGet, routertest.CardOwnershipPath},
					{"プロダクト一覧の取得を要求すると、401の応答になり、本文のerror項目はヘッダが無いことを示す", http.MethodGet, routertest.ProductsPath},
					{"デッキ一覧の取得を要求すると、401の応答になり、本文のerror項目はヘッダが無いことを示す", http.MethodGet, routertest.DecksPath},
					{"デッキ詳細の取得を要求すると、401の応答になり、本文のerror項目はヘッダが無いことを示す", http.MethodGet, routertest.BuildDeckPath(7)},
					{"デッキの作成を要求すると、401の応答になり、本文のerror項目はヘッダが無いことを示す", http.MethodPost, routertest.DecksPath},
					{"デッキの更新を要求すると、401の応答になり、本文のerror項目はヘッダが無いことを示す", http.MethodPut, routertest.BuildDeckPath(7)},
					{"デッキの削除を要求すると、401の応答になり、本文のerror項目はヘッダが無いことを示す", http.MethodDelete, routertest.BuildDeckPath(7)},
					{"デッキのバトル使用検証を要求すると、401の応答になり、本文のerror項目はヘッダが無いことを示す", http.MethodPost, routertest.BuildDeckPath(7) + "/validate-for-battle"},
				}
				for _, tc := range missingHeaderTests {
					t.Run(tc.name, func(t *testing.T) {
						h := routertest.NewHarness(t)

						recorder := h.SendRequestWithoutToken(tc.method, tc.path, "")

						assert.Equal(t, http.StatusUnauthorized, recorder.Code)
						assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "X-Internal-Auth header is required")
					})
				}
			})
			t.Run("X-Internal-Authヘッダの値がinvalid-tokenのとき、デッキ一覧の取得を要求すると、401の応答になり、本文のerror項目はトークンが無効であることを示す", func(t *testing.T) {
				h := routertest.NewHarness(t)

				recorder := h.SendRequestWithToken(http.MethodGet, routertest.DecksPath, routertest.InvalidToken, "")

				assert.Equal(t, http.StatusUnauthorized, recorder.Code)
				assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "invalid internal auth token")
			})
		})
	})
}
