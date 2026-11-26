package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/router/routertest"
)

func TestPlayerOnboardedEventReceive(t *testing.T) {
	t.Run("[カード配布]プレイヤー登録完了イベントの受信", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("陣営SHEのプレイヤーAのプレイヤー登録完了イベントをbase64にして送ると、本文のない200の応答になる", func(t *testing.T) {
				h := routertest.NewHarness(t)

				recorder := h.SendPlayerOnboarded(routertest.BuildEnvelope(routertest.BuildPlayerOnboardedEventJSON("SHE")))

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, 0, recorder.Body.Len())
			})
			t.Run("基本パックbasicにTST-0001が3枚、陣営SHEのパックfaction_set_sheにTST-0002が2枚、陣営Tenkiのパックfaction_set_tenkiにTST-0003が1枚あるとき、陣営SHEのプレイヤーAのプレイヤー登録完了イベントを送ったあとに、プレイヤーAのカード一覧を取得すると", func(t *testing.T) {
				requestPlayerCardsAfterEvent := func(t *testing.T) []any {
					h := routertest.NewHarness(t)
					h.SendPlayerOnboarded(routertest.BuildEnvelope(routertest.BuildPlayerOnboardedEventJSON("SHE")))
					return routertest.DecodeArray(t, h.GetPlayerCards())
				}

				t.Run("応答のcard_idはTST-0001とTST-0002だけになる", func(t *testing.T) {
					cards := requestPlayerCardsAfterEvent(t)

					assert.ElementsMatch(t, []any{"TST-0001", "TST-0002"}, routertest.CollectValues(t, cards, "card_id"))
				})
				t.Run("応答のカードTST-0001のcountは3になる", func(t *testing.T) {
					cards := requestPlayerCardsAfterEvent(t)

					assert.EqualValues(t, 3, routertest.FindObject(t, cards, "card_id", "TST-0001")["count"])
				})
				t.Run("応答のカードTST-0002のcountは2になる", func(t *testing.T) {
					cards := requestPlayerCardsAfterEvent(t)

					assert.EqualValues(t, 2, routertest.FindObject(t, cards, "card_id", "TST-0002")["count"])
				})
			})
		})
		t.Run("異常系", func(t *testing.T) {
			t.Run("受信内容が読めないとき", func(t *testing.T) {
				malformedTests := []struct {
					name string
					body string
				}{
					{"ボディがJSONでない文字列だと、400の応答になり、本文のerror項目は受信内容が不正であることを示す", "not-json"},
					{"ボディが空だと、400の応答になり、本文のerror項目は受信内容が不正であることを示す", ""},
					{"messageが無いと、400の応答になり、本文のerror項目は受信内容が不正であることを示す", `{}`},
					{"messageのdataが無いと、400の応答になり、本文のerror項目は受信内容が不正であることを示す", `{"message":{}}`},
					{"messageのdataが空文字だと、400の応答になり、本文のerror項目は受信内容が不正であることを示す", `{"message":{"data":""}}`},
				}
				for _, tc := range malformedTests {
					t.Run(tc.name, func(t *testing.T) {
						h := routertest.NewHarness(t)

						recorder := h.SendPlayerOnboarded(tc.body)

						assert.Equal(t, http.StatusBadRequest, recorder.Code)
						assert.JSONEq(t, routertest.MalformedEnvelopeBody, recorder.Body.String())
					})
				}
			})
			t.Run("messageのdataが!!!のとき、400の応答になり、本文のerror項目はdataのbase64が不正であることを示す", func(t *testing.T) {
				h := routertest.NewHarness(t)

				recorder := h.SendPlayerOnboarded(`{"message":{"data":"!!!"}}`)

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.JSONEq(t, routertest.MalformedBase64Body, recorder.Body.String())
			})
			t.Run("messageのdataがnot-jsonをbase64にした文字列のとき", func(t *testing.T) {
				sendNotJSON := func(t *testing.T) (*routertest.Harness, *httptest.ResponseRecorder) {
					h := routertest.NewHarness(t)
					return h, h.SendPlayerOnboarded(routertest.BuildEnvelope("not-json"))
				}

				t.Run("内部エラーを示す500の応答になる", func(t *testing.T) {
					_, recorder := sendNotJSON(t)

					assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				})
				t.Run("Errorレベルのログのerror項目は、イベントの内容が読めなかったことを表すエラーの文言を含む", func(t *testing.T) {
					h, _ := sendNotJSON(t)

					h.RequireErrorLogContaining(t, "error", "bad payload")
				})
			})
			t.Run("プレイヤーAがカードパックTST-0021を購入したイベントを、プレイヤー登録完了の通知の受信へ送ったとき", func(t *testing.T) {
				sendPurchaseEvent := func(t *testing.T) (*routertest.Harness, *httptest.ResponseRecorder) {
					h := routertest.NewHarness(t)
					return h, h.SendPlayerOnboarded(routertest.BuildEnvelope(routertest.BuildCardPackPurchasedEventJSON("TST-0021")))
				}

				t.Run("内部エラーを示す500の応答になる", func(t *testing.T) {
					_, recorder := sendPurchaseEvent(t)

					assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				})
				t.Run("Errorレベルのログのerror項目は、イベントの種別が想定と異なることを表すエラーの文言を含む", func(t *testing.T) {
					h, _ := sendPurchaseEvent(t)

					h.RequireErrorLogContaining(t, "error", "unexpected event_type")
				})
			})
			t.Run("陣営SHEのパックfaction_set_sheの所持カードの保存に失敗してカードを配布できなかったイベントが再配信されたとき", func(t *testing.T) {
				newHarnessAfterFailedDelivery := func(t *testing.T) *routertest.Harness {
					h := routertest.NewHarness(t)
					h.DB.FailAddingCard("TST-0002", routertest.ErrPlayerCardSaveFailed)
					firstAttempt := h.SendPlayerOnboarded(routertest.BuildEnvelope(routertest.BuildPlayerOnboardedEventJSON("SHE")))
					require.Equal(t, http.StatusInternalServerError, firstAttempt.Code)
					h.DB.RecoverAddingCards()
					return h
				}

				t.Run("同じイベントを再度送ると、本文のない200の応答になる", func(t *testing.T) {
					h := newHarnessAfterFailedDelivery(t)

					recorder := h.SendPlayerOnboarded(routertest.BuildEnvelope(routertest.BuildPlayerOnboardedEventJSON("SHE")))

					assert.Equal(t, http.StatusOK, recorder.Code)
					assert.Equal(t, 0, recorder.Body.Len())
				})
				t.Run("基本パックbasicにTST-0001が3枚、陣営SHEのパックfaction_set_sheにTST-0002が2枚、陣営Tenkiのパックfaction_set_tenkiにTST-0003が1枚あり、陣営SHEのプレイヤーAのプレイヤー登録完了イベントを同じ内容で再度送ったあとに、プレイヤーAのカード一覧を取得すると", func(t *testing.T) {
					requestPlayerCardsAfterRedelivery := func(t *testing.T) []any {
						h := newHarnessAfterFailedDelivery(t)
						h.SendPlayerOnboarded(routertest.BuildEnvelope(routertest.BuildPlayerOnboardedEventJSON("SHE")))
						return routertest.DecodeArray(t, h.GetPlayerCards())
					}

					t.Run("応答のcard_idはTST-0001とTST-0002だけになる", func(t *testing.T) {
						cards := requestPlayerCardsAfterRedelivery(t)

						assert.ElementsMatch(t, []any{"TST-0001", "TST-0002"}, routertest.CollectValues(t, cards, "card_id"))
					})
					t.Run("応答のカードTST-0001のcountは3になる", func(t *testing.T) {
						cards := requestPlayerCardsAfterRedelivery(t)

						assert.EqualValues(t, 3, routertest.FindObject(t, cards, "card_id", "TST-0001")["count"])
					})
					t.Run("応答のカードTST-0002のcountは2になる", func(t *testing.T) {
						cards := requestPlayerCardsAfterRedelivery(t)

						assert.EqualValues(t, 2, routertest.FindObject(t, cards, "card_id", "TST-0002")["count"])
					})
				})
			})
		})
	})
}
