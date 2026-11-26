package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/router/routertest"
)

func TestDeckDelete(t *testing.T) {
	t.Run("[デッキ]デッキの削除", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("デッキID7のデッキが存在するとき、デッキID7を削除すると、本文のない204の応答になる", func(t *testing.T) {
				h := routertest.NewHarnessWithStoredDeck(t)

				recorder := h.DeleteDeck(routertest.BuildDeckPath(7))

				assert.Equal(t, http.StatusNoContent, recorder.Code)
				assert.Equal(t, 0, recorder.Body.Len())
			})
			t.Run("プレイヤーAがデッキID7と8を持つとき、デッキID7を削除したあとにデッキ一覧を取得すると、一覧の応答のdeck_idは8だけになる", func(t *testing.T) {
				h := routertest.NewHarnessWithStoredDeck(t)
				h.StoreDeckWithCard(8, routertest.CardCount{CardID: "TST-0001", ArtNo: 0, Count: 1})
				h.DeleteDeck(routertest.BuildDeckPath(7))

				decks := routertest.DecodeArray(t, h.GetDecks())

				assert.Equal(t, []any{float64(8)}, routertest.CollectValues(t, decks, "deck_id"))
			})
			t.Run("デッキID7を削除したあとにデッキID7の詳細を取得すると、404の応答になり、本文のerror項目はデッキが見つからないことを示す", func(t *testing.T) {
				h := routertest.NewHarnessWithStoredDeck(t)
				h.DeleteDeck(routertest.BuildDeckPath(7))

				recorder := h.GetDeck(7)

				assert.Equal(t, http.StatusNotFound, recorder.Code)
				assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "not found")
			})
		})
		t.Run("異常系", func(t *testing.T) {
			t.Run("デッキIDがabcのとき、400の応答になり、本文のerror項目はデッキIDが不正であることを示す", func(t *testing.T) {
				h := routertest.NewHarnessWithStoredDeck(t)

				recorder := h.DeleteDeck(routertest.DecksPath + "/abc")

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.JSONEq(t, routertest.InvalidDeckIDBody, recorder.Body.String())
			})
			t.Run("デッキID7のデッキが存在しないとき、404の応答になり、本文のerror項目はデッキが見つからないことを示す", func(t *testing.T) {
				h := routertest.NewHarness(t)

				recorder := h.DeleteDeck(routertest.BuildDeckPath(7))

				assert.Equal(t, http.StatusNotFound, recorder.Code)
				assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "not found")
			})
			t.Run("デッキの削除がdeck-delete-failedのエラーで失敗したとき", func(t *testing.T) {
				requestWithFailure := func(t *testing.T) (*routertest.Harness, *httptest.ResponseRecorder) {
					h := routertest.NewHarnessWithStoredDeck(t)
					h.DB.ErrDeleteDeck = routertest.ErrDeckDeleteFailed
					return h, h.DeleteDeck(routertest.BuildDeckPath(7))
				}

				t.Run("内部エラーを示す500の応答になる", func(t *testing.T) {
					_, recorder := requestWithFailure(t)

					assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				})
				t.Run("Errorレベルのログのerror項目はdeck-delete-failedを含む", func(t *testing.T) {
					h, _ := requestWithFailure(t)

					h.RequireErrorLogContaining(t, "error", "deck-delete-failed")
				})
			})
		})
	})
}
