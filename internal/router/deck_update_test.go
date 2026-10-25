package router_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/router/routertest"
)

func newHarnessWithStoredDeckAndOwnedCards(t *testing.T) *routertest.Harness {
	h := routertest.NewHarnessOwningValidDeckCards(t)
	h.StoreDeckWithCard(7, routertest.CardCount{CardID: "TST-0001", ArtNo: 2, Count: 3})
	h.DB.SetOwned(routertest.PlayerA, "TST-0001", 2, 3)
	h.DB.SetOwned(routertest.PlayerA, "TST-0002", 1, 2)
	return h
}

func requestUpdatedDeck(t *testing.T, body routertest.DeckRequestBody) map[string]any {
	return routertest.DecodeObject(t, newHarnessWithStoredDeckAndOwnedCards(t).UpdateDeck(t, 7, body))
}

func TestDeckUpdate(t *testing.T) {
	t.Run("[デッキ]デッキの更新", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("デッキID7のデッキが存在し、プレイヤーAが陣営SHEを所持し、陣営SHE、プロダクトTST-0001、ルーチン施策TST-0011、スペシャル施策TST-0012で、SHEのカードTST-0101からTST-0105とNeutralのカードTST-0111からTST-0115のアート0を各3枚(所持各3枚)の合計30枚のデッキとして更新するとき", func(t *testing.T) {
				t.Run("デッキID7を更新すると", func(t *testing.T) {
					t.Run("200の応答になる", func(t *testing.T) {
						recorder := newHarnessWithStoredDeckAndOwnedCards(t).UpdateDeck(t, 7, routertest.NewValidDeckRequestBody())

						assert.Equal(t, http.StatusOK, recorder.Code)
					})
					t.Run("応答のdeck_idは7になる", func(t *testing.T) {
						deck := requestUpdatedDeck(t, routertest.NewValidDeckRequestBody())

						assert.EqualValues(t, 7, deck["deck_id"])
					})
				})
				t.Run("プレイヤーAのトークンでデッキを更新すると、応答のplayer_idは00000000-0000-0000-0000-000000000001になる", func(t *testing.T) {
					deck := requestUpdatedDeck(t, routertest.NewValidDeckRequestBody())

					assert.Equal(t, "00000000-0000-0000-0000-000000000001", deck["player_id"])
				})
				t.Run("デッキ名をテストデッキBにして更新すると", func(t *testing.T) {
					requestDeckNamedB := func(t *testing.T) map[string]any {
						body := routertest.NewValidDeckRequestBody()
						body.DeckName = "テストデッキB"
						return requestUpdatedDeck(t, body)
					}

					t.Run("応答のdeck_nameはテストデッキBになる", func(t *testing.T) {
						assert.Equal(t, "テストデッキB", requestDeckNamedB(t)["deck_name"])
					})
					t.Run("応答のfactionはSHEになる", func(t *testing.T) {
						assert.Equal(t, "SHE", requestDeckNamedB(t)["faction"])
					})
					t.Run("応答のproduct_idはTST-0001になる", func(t *testing.T) {
						assert.Equal(t, "TST-0001", requestDeckNamedB(t)["product_id"])
					})
					t.Run("応答のroutine_idはTST-0011になる", func(t *testing.T) {
						assert.Equal(t, "TST-0011", requestDeckNamedB(t)["routine_id"])
					})
					t.Run("応答のspecial_idはTST-0012になる", func(t *testing.T) {
						assert.Equal(t, "TST-0012", requestDeckNamedB(t)["special_id"])
					})
				})
				t.Run("構成カードをSHEのカードTST-0001のアート2の3枚(所持3枚)だけにして、プレイヤーAのトークンでデッキID7を更新すると", func(t *testing.T) {
					requestDeckCard := func(t *testing.T) map[string]any {
						body := routertest.NewDeckRequestBody([]routertest.CardCount{{CardID: "TST-0001", ArtNo: 2, Count: 3}})
						return routertest.ExtractSingleElement(t, requestUpdatedDeck(t, body)["deck_cards"])
					}

					t.Run("応答のdeck_cardsの1件のplayer_idは00000000-0000-0000-0000-000000000001になる", func(t *testing.T) {
						assert.Equal(t, "00000000-0000-0000-0000-000000000001", requestDeckCard(t)["player_id"])
					})
					t.Run("応答のdeck_cardsの1件のdeck_idは7になる", func(t *testing.T) {
						assert.EqualValues(t, 7, requestDeckCard(t)["deck_id"])
					})
					t.Run("応答のdeck_cardsの1件のcard_idはTST-0001になる", func(t *testing.T) {
						assert.Equal(t, "TST-0001", requestDeckCard(t)["card_id"])
					})
					t.Run("応答のdeck_cardsの1件のart_noは2になる", func(t *testing.T) {
						assert.EqualValues(t, 2, requestDeckCard(t)["art_no"])
					})
					t.Run("応答のdeck_cardsの1件のcountは3になる", func(t *testing.T) {
						assert.EqualValues(t, 3, requestDeckCard(t)["count"])
					})
				})
				t.Run("デッキを更新すると", func(t *testing.T) {
					t.Run("応答のupdated_atは、要求を送った時刻から応答を受け取った時刻までの間の日時になる", func(t *testing.T) {
						h := newHarnessWithStoredDeckAndOwnedCards(t)
						sentAt := time.Now()

						recorder := h.UpdateDeck(t, 7, routertest.NewValidDeckRequestBody())

						receivedAt := time.Now()
						updatedAt, err := time.Parse(time.RFC3339Nano, fmt.Sprint(routertest.DecodeObject(t, recorder)["updated_at"]))
						require.NoError(t, err)
						assert.WithinRange(t, updatedAt, sentAt, receivedAt)
					})
					t.Run("応答のis_validはtrueになる", func(t *testing.T) {
						deck := requestUpdatedDeck(t, routertest.NewValidDeckRequestBody())

						assert.Equal(t, true, deck["is_valid"])
					})
				})
				t.Run("デッキの作成日時が2026-01-02T03:04:05Zで保存されているとき、デッキを更新すると、応答のcreated_atは2026-01-02T03:04:05Zになる", func(t *testing.T) {
					deck := requestUpdatedDeck(t, routertest.NewValidDeckRequestBody())

					assert.Equal(t, "2026-01-02T03:04:05Z", deck["created_at"])
				})
				t.Run("TST-0115のアート0を2枚に減らして合計29枚にして更新すると", func(t *testing.T) {
					updateToTwentyNineCards := func(t *testing.T) *httptest.ResponseRecorder {
						body := routertest.NewValidDeckRequestBody()
						routertest.ReduceToTwentyNineCards(&body)
						return newHarnessWithStoredDeckAndOwnedCards(t).UpdateDeck(t, 7, body)
					}

					t.Run("200の応答になる", func(t *testing.T) {
						assert.Equal(t, http.StatusOK, updateToTwentyNineCards(t).Code)
					})
					t.Run("応答のis_validはfalseになる", func(t *testing.T) {
						assert.Equal(t, false, routertest.DecodeObject(t, updateToTwentyNineCards(t))["is_valid"])
					})
				})
				t.Run("プレイマット番号5を指定して更新すると、応答のplaymat_noは5になる", func(t *testing.T) {
					body := routertest.NewValidDeckRequestBody()
					body.PlaymatNo = routertest.NewPointer(int64(5))

					deck := requestUpdatedDeck(t, body)

					assert.EqualValues(t, 5, deck["playmat_no"])
				})
				t.Run("プレイマット番号を指定せずに更新すると、応答のデッキにplaymat_noの項目が含まれない", func(t *testing.T) {
					body := routertest.NewValidDeckRequestBody()
					body.PlaymatNo = nil

					deck := requestUpdatedDeck(t, body)

					assert.NotContains(t, deck, "playmat_no")
				})
				t.Run("スリーブ番号6を指定して更新すると、応答のsleeve_noは6になる", func(t *testing.T) {
					body := routertest.NewValidDeckRequestBody()
					body.SleeveNo = routertest.NewPointer(int64(6))

					deck := requestUpdatedDeck(t, body)

					assert.EqualValues(t, 6, deck["sleeve_no"])
				})
				t.Run("スリーブ番号を指定せずに更新すると、応答のデッキにsleeve_noの項目が含まれない", func(t *testing.T) {
					body := routertest.NewValidDeckRequestBody()
					body.SleeveNo = nil

					deck := requestUpdatedDeck(t, body)

					assert.NotContains(t, deck, "sleeve_no")
				})
				t.Run("デッキ名がテストデッキAで構成カードがTST-0001のアート2の3枚のデッキID7を、デッキ名をテストデッキB、構成カードをTST-0002のアート1の2枚にして更新したあとにデッキ詳細を取得すると", func(t *testing.T) {
					requestDeckDetailAfterUpdate := func(t *testing.T) map[string]any {
						h := newHarnessWithStoredDeckAndOwnedCards(t)
						body := routertest.NewDeckRequestBody([]routertest.CardCount{{CardID: "TST-0002", ArtNo: 1, Count: 2}})
						body.DeckName = "テストデッキB"
						h.UpdateDeck(t, 7, body)
						return routertest.DecodeObject(t, h.GetDeck(7))
					}

					t.Run("詳細の応答のdeck_nameはテストデッキBになる", func(t *testing.T) {
						assert.Equal(t, "テストデッキB", requestDeckDetailAfterUpdate(t)["deck_name"])
					})
					t.Run("詳細の応答のdeck_cardsの1件のcard_idはTST-0002になる", func(t *testing.T) {
						assert.Equal(t, "TST-0002", routertest.ExtractSingleElement(t, requestDeckDetailAfterUpdate(t)["deck_cards"])["card_id"])
					})
					t.Run("詳細の応答のdeck_cardsの1件のart_noは1になる", func(t *testing.T) {
						assert.EqualValues(t, 1, routertest.ExtractSingleElement(t, requestDeckDetailAfterUpdate(t)["deck_cards"])["art_no"])
					})
					t.Run("詳細の応答のdeck_cardsの1件のcountは2になる", func(t *testing.T) {
						assert.EqualValues(t, 2, routertest.ExtractSingleElement(t, requestDeckDetailAfterUpdate(t)["deck_cards"])["count"])
					})
				})
			})
		})
		t.Run("異常系", func(t *testing.T) {
			t.Run("デッキIDがabcのとき、400の応答になり、本文のerror項目はデッキIDが不正であることを示す", func(t *testing.T) {
				h := newHarnessWithStoredDeckAndOwnedCards(t)

				recorder := h.SendRequestWithToken(http.MethodPut, routertest.DecksPath+"/abc", routertest.TokenA, routertest.EncodeJSON(t, routertest.NewValidDeckRequestBody()))

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.JSONEq(t, routertest.InvalidDeckIDBody, recorder.Body.String())
			})
			t.Run("リクエストボディが読めないとき", func(t *testing.T) {
				malformedTests := []struct {
					name string
					body string
				}{
					{"ボディがJSONでない文字列だと、400の応答になり、本文のerror項目はリクエストボディが不正であることを示す", "not-json"},
					{"ボディが空だと、400の応答になり、本文のerror項目はリクエストボディが不正であることを示す", ""},
				}
				for _, tc := range malformedTests {
					t.Run(tc.name, func(t *testing.T) {
						h := newHarnessWithStoredDeckAndOwnedCards(t)

						recorder := h.SendRequestWithToken(http.MethodPut, routertest.BuildDeckPath(7), routertest.TokenA, tc.body)

						assert.Equal(t, http.StatusBadRequest, recorder.Code)
						assert.JSONEq(t, routertest.MalformedRequestBodyBody, recorder.Body.String())
					})
				}
			})
			t.Run("デッキID7のデッキ名がテストデッキAで構成カードがTST-0001のアート2の3枚のとき、デッキ名をテストデッキB、構成カードをSHEのカードTST-0101からTST-0105とNeutralのカードTST-0111からTST-0115のアート0を各3枚(所持各3枚)にNeutralのカードTST-0116のアート0を1枚(所持1枚)足した合計31枚にする更新が400で拒否されたあとにデッキ詳細を取得すると", func(t *testing.T) {
				requestDeckDetailAfterRejection := func(t *testing.T) map[string]any {
					h := newHarnessWithStoredDeckAndOwnedCards(t)
					body := routertest.NewValidDeckRequestBody()
					body.DeckName = "テストデッキB"
					routertest.AddThirtyFirstCard(h, &body)
					h.UpdateDeck(t, 7, body)
					return routertest.DecodeObject(t, h.GetDeck(7))
				}

				t.Run("詳細の応答のdeck_nameはテストデッキAのままになる", func(t *testing.T) {
					assert.Equal(t, "テストデッキA", requestDeckDetailAfterRejection(t)["deck_name"])
				})
				t.Run("詳細の応答のdeck_cardsの1件のcard_idはTST-0001のままになる", func(t *testing.T) {
					assert.Equal(t, "TST-0001", routertest.ExtractSingleElement(t, requestDeckDetailAfterRejection(t)["deck_cards"])["card_id"])
				})
				t.Run("詳細の応答のdeck_cardsの1件のart_noは2のままになる", func(t *testing.T) {
					assert.EqualValues(t, 2, routertest.ExtractSingleElement(t, requestDeckDetailAfterRejection(t)["deck_cards"])["art_no"])
				})
				t.Run("詳細の応答のdeck_cardsの1件のcountは3のままになる", func(t *testing.T) {
					assert.EqualValues(t, 3, routertest.ExtractSingleElement(t, requestDeckDetailAfterRejection(t)["deck_cards"])["count"])
				})
			})
			t.Run("更新が業務上の理由で拒否されたとき", func(t *testing.T) {
				t.Run("プレイヤーAが陣営SHEを所持し、陣営SHE、プロダクトTST-0001、ルーチン施策TST-0011、スペシャル施策TST-0012で、SHEのカードTST-0101からTST-0105とNeutralのカードTST-0111からTST-0115のアート0を各3枚(所持各3枚)の合計30枚のデッキとして更新を要求すると", func(t *testing.T) {
					t.Run("更新対象のデッキID7のデッキが存在しないと、404の応答になり、本文のerror項目はデッキが見つからないことを示す", func(t *testing.T) {
						h := routertest.NewHarnessOwningValidDeckCards(t)

						recorder := h.UpdateDeck(t, 7, routertest.NewValidDeckRequestBody())

						assert.Equal(t, http.StatusNotFound, recorder.Code)
						assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "not found")
					})
					t.Run("TST-0101のアート0の所持を2枚に減らすと、403の応答になり、本文のerror項目は所持枚数が足りないことを示す", func(t *testing.T) {
						h := newHarnessWithStoredDeckAndOwnedCards(t)
						routertest.ReduceOwnedFirstCard(h)

						recorder := h.UpdateDeck(t, 7, routertest.NewValidDeckRequestBody())

						assert.Equal(t, http.StatusForbidden, recorder.Code)
						assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "not enough owned")
					})
					t.Run("陣営をNeutralにすると、400の応答になり、本文のerror項目は陣営が選択できないことを示す", func(t *testing.T) {
						h := newHarnessWithStoredDeckAndOwnedCards(t)
						body := routertest.NewValidDeckRequestBody()
						routertest.DeclareNeutral(&body)

						recorder := h.UpdateDeck(t, 7, body)

						assert.Equal(t, http.StatusBadRequest, recorder.Code)
						assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "is not selectable")
					})
					t.Run("TST-0115のアート0の3枚を、制限区分がlimitedのカードTST-0003のアート0の2枚(所持2枚)に置き換えると、400の応答になり、本文のerror項目は制限枚数を超えていることを示す", func(t *testing.T) {
						h := newHarnessWithStoredDeckAndOwnedCards(t)
						body := routertest.NewValidDeckRequestBody()
						routertest.SwapInLimitedCards(h, &body)

						recorder := h.UpdateDeck(t, 7, body)

						assert.Equal(t, http.StatusBadRequest, recorder.Code)
						assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "exceeds restriction limit")
					})
					t.Run("NeutralのカードTST-0116のアート0を1枚(所持1枚)足して合計31枚にすると、400の応答になり、本文のerror項目はデッキの枚数が上限を超えていることを示す", func(t *testing.T) {
						h := newHarnessWithStoredDeckAndOwnedCards(t)
						body := routertest.NewValidDeckRequestBody()
						routertest.AddThirtyFirstCard(h, &body)

						recorder := h.UpdateDeck(t, 7, body)

						assert.Equal(t, http.StatusBadRequest, recorder.Code)
						assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "cannot exceed")
					})
				})
			})
			requestBody := routertest.EncodeJSON(t, routertest.NewValidDeckRequestBody())
			routertest.RunInternalErrorCases(t, newHarnessWithStoredDeckAndOwnedCards, func(h *routertest.Harness) *httptest.ResponseRecorder {
				return h.SendRequestWithToken(http.MethodPut, routertest.BuildDeckPath(7), routertest.TokenA, requestBody)
			}, []routertest.InternalErrorCase{
				{
					Given:   "所持カードの取得がplayer-card-find-failedのエラーで失敗したとき",
					Cause:   "player-card-find-failed",
					Arrange: func(h *routertest.Harness) { h.DB.ErrGetPlayerCards = routertest.ErrPlayerCardFindFailed },
				},
				{
					Given:   "陣営の所持の照会がaccount-faction-owned-failedのエラーで失敗したとき",
					Cause:   "account-faction-owned-failed",
					Arrange: func(h *routertest.Harness) { h.Factions.Err = routertest.ErrAccountFactionOwnedFailed },
				},
				{
					Given:   "デッキの保存がdeck-save-failedのエラーで失敗したとき",
					Cause:   "deck-save-failed",
					Arrange: func(h *routertest.Harness) { h.DB.ErrSaveDeck = routertest.ErrDeckSaveFailed },
				},
			})
		})
	})
}
