package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/router/routertest"
)

func newHarnessWithStoredValidDeck(t *testing.T) *routertest.Harness {
	h := routertest.NewHarness(t)
	h.StoreValidDeck(7)
	return h
}

func TestDeckBattleValidation(t *testing.T) {
	t.Run("[デッキ]デッキのバトル使用検証", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("プレイヤーAが、陣営SHE、プロダクトTST-0001、ルーチン施策TST-0011、スペシャル施策TST-0012で、SHEのカードTST-0101からTST-0105とNeutralのカードTST-0111からTST-0115のアート0を各3枚(所持各3枚)の合計30枚のデッキID7を持つとき、本文のない200の応答になる", func(t *testing.T) {
				h := newHarnessWithStoredValidDeck(t)

				recorder := h.ValidateDeckForBattle(routertest.BuildDeckPath(7))

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, 0, recorder.Body.Len())
			})
		})
		t.Run("異常系", func(t *testing.T) {
			t.Run("デッキIDがabcのとき、400の応答になり、本文のerror項目はデッキIDが不正であることを示す", func(t *testing.T) {
				h := newHarnessWithStoredValidDeck(t)

				recorder := h.ValidateDeckForBattle(routertest.DecksPath + "/abc")

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.JSONEq(t, routertest.InvalidDeckIDBody, recorder.Body.String())
			})
			t.Run("デッキID7のデッキが存在しないとき、404の応答になり、本文のerror項目はデッキが見つからないことを示す", func(t *testing.T) {
				h := routertest.NewHarness(t)

				recorder := h.ValidateDeckForBattle(routertest.BuildDeckPath(7))

				assert.Equal(t, http.StatusNotFound, recorder.Code)
				assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "not found")
			})
			t.Run("プレイヤーAが、陣営SHE、プロダクトTST-0001、ルーチン施策TST-0011、スペシャル施策TST-0012で、SHEのカードTST-0101からTST-0105とNeutralのカードTST-0111からTST-0115のアート0を各3枚(所持各3枚)の合計30枚のデッキID7を持つとき", func(t *testing.T) {
				t.Run("TST-0115のアート0を2枚に減らして合計29枚にすると、400の応答になり、本文のerror項目は枚数が30枚に足りないことを示す", func(t *testing.T) {
					h := routertest.NewHarness(t)
					composition := routertest.ChangeCompositionCount(routertest.ValidDeckComposition, "TST-0115", 2)
					h.DB.StoreDeck(routertest.NewStoredDeck(7), routertest.BuildDeckCards(routertest.PlayerA, 7, composition))
					h.DB.SetOwnedComposition(routertest.PlayerA, routertest.ValidDeckComposition)

					recorder := h.ValidateDeckForBattle(routertest.BuildDeckPath(7))

					assert.Equal(t, http.StatusBadRequest, recorder.Code)
					assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "need exactly")
				})
				t.Run("TST-0101のアート0の所持を2枚に減らすと、403の応答になり、本文のerror項目は所持枚数が足りないことを示す", func(t *testing.T) {
					h := newHarnessWithStoredValidDeck(t)
					routertest.ReduceOwnedFirstCard(h)

					recorder := h.ValidateDeckForBattle(routertest.BuildDeckPath(7))

					assert.Equal(t, http.StatusForbidden, recorder.Code)
					assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "not enough owned")
				})
			})
			routertest.RunInternalErrorCases(t, newHarnessWithStoredValidDeck, func(h *routertest.Harness) *httptest.ResponseRecorder {
				return h.ValidateDeckForBattle(routertest.BuildDeckPath(7))
			}, []routertest.InternalErrorCase{
				{
					Given:   "デッキの取得がdeck-find-failedのエラーで失敗したとき",
					Cause:   "deck-find-failed",
					Arrange: func(h *routertest.Harness) { h.DB.ErrFindDeck = routertest.ErrDeckFindFailed },
				},
				{
					Given:   "デッキの構成カードの取得がdeck-card-find-failedのエラーで失敗したとき",
					Cause:   "deck-card-find-failed",
					Arrange: func(h *routertest.Harness) { h.DB.ErrGetDeckCards = routertest.ErrDeckCardFindFailed },
				},
				{
					Given:   "所持カードの取得がplayer-card-find-failedのエラーで失敗したとき",
					Cause:   "player-card-find-failed",
					Arrange: func(h *routertest.Harness) { h.DB.ErrGetPlayerCards = routertest.ErrPlayerCardFindFailed },
				},
			})
		})
	})
}
