package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/router/routertest"
)

func TestCardPackPurchasedEventReceive(t *testing.T) {
	t.Run("[カード配布]カードパック購入イベントの受信", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("プレイヤーAがカードパックTST-0021を購入したイベントをbase64にして送ると、本文のない200の応答になる", func(t *testing.T) {
				h := routertest.NewHarness(t)

				recorder := h.SendCardPackPurchased(routertest.BuildEnvelope(routertest.BuildCardPackPurchasedEventJSON("TST-0021")))

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, 0, recorder.Body.Len())
			})
			t.Run("カードパックTST-0021にTST-0001が3枚、カードパックTST-0022にTST-0002が2枚あるとき、プレイヤーAがTST-0021を購入したイベントを送ったあとに、プレイヤーAのカード一覧を取得すると", func(t *testing.T) {
				requestPlayerCardsAfterEvent := func(t *testing.T) []any {
					h := routertest.NewHarness(t)
					h.SendCardPackPurchased(routertest.BuildEnvelope(routertest.BuildCardPackPurchasedEventJSON("TST-0021")))
					return routertest.DecodeArray(t, h.GetPlayerCards())
				}

				t.Run("応答のcard_idはTST-0001だけになる", func(t *testing.T) {
					cards := requestPlayerCardsAfterEvent(t)

					assert.Equal(t, []any{"TST-0001"}, routertest.CollectValues(t, cards, "card_id"))
				})
				t.Run("応答のカードTST-0001のcountは3になる", func(t *testing.T) {
					cards := requestPlayerCardsAfterEvent(t)

					assert.EqualValues(t, 3, routertest.FindObject(t, cards, "card_id", "TST-0001")["count"])
				})
			})
		})
		t.Run("異常系", func(t *testing.T) {
			t.Run("messageのdataがnot-jsonをbase64にした文字列のとき", func(t *testing.T) {
				sendNotJSON := func(t *testing.T) (*routertest.Harness, *httptest.ResponseRecorder) {
					h := routertest.NewHarness(t)
					return h, h.SendCardPackPurchased(routertest.BuildEnvelope("not-json"))
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
			t.Run("陣営SHEのプレイヤーAのプレイヤー登録完了イベントを、カードパック購入の通知の受信へ送ったとき", func(t *testing.T) {
				sendOnboardedEvent := func(t *testing.T) (*routertest.Harness, *httptest.ResponseRecorder) {
					h := routertest.NewHarness(t)
					return h, h.SendCardPackPurchased(routertest.BuildEnvelope(routertest.BuildPlayerOnboardedEventJSON("SHE")))
				}

				t.Run("内部エラーを示す500の応答になる", func(t *testing.T) {
					_, recorder := sendOnboardedEvent(t)

					assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				})
				t.Run("Errorレベルのログのerror項目は、イベントの種別が想定と異なることを表すエラーの文言を含む", func(t *testing.T) {
					h, _ := sendOnboardedEvent(t)

					h.RequireErrorLogContaining(t, "error", "unexpected event_type")
				})
			})
			t.Run("プレイヤーAが存在しないカードパックTST-0029を購入したイベントを送ったとき", func(t *testing.T) {
				sendUnknownPackEvent := func(t *testing.T) (*routertest.Harness, *httptest.ResponseRecorder) {
					h := routertest.NewHarness(t)
					return h, h.SendCardPackPurchased(routertest.BuildEnvelope(routertest.BuildCardPackPurchasedEventJSON("TST-0029")))
				}

				t.Run("内部エラーを示す500の応答になる", func(t *testing.T) {
					_, recorder := sendUnknownPackEvent(t)

					assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				})
				t.Run("Errorレベルのログのerror項目はTST-0029を含む", func(t *testing.T) {
					h, _ := sendUnknownPackEvent(t)

					h.RequireErrorLogContaining(t, "error", "TST-0029")
				})
			})
			t.Run("所持カードの保存に失敗してカードを配布できなかったイベントが再配信されたとき", func(t *testing.T) {
				newHarnessAfterFailedDelivery := func(t *testing.T) *routertest.Harness {
					h := routertest.NewHarness(t)
					h.DB.FailAddingCard("TST-0001", routertest.ErrPlayerCardSaveFailed)
					firstAttempt := h.SendCardPackPurchased(routertest.BuildEnvelope(routertest.BuildCardPackPurchasedEventJSON("TST-0021")))
					require.Equal(t, http.StatusInternalServerError, firstAttempt.Code)
					h.DB.RecoverAddingCards()
					return h
				}

				t.Run("同じイベントを再度送ると、本文のない200の応答になる", func(t *testing.T) {
					h := newHarnessAfterFailedDelivery(t)

					recorder := h.SendCardPackPurchased(routertest.BuildEnvelope(routertest.BuildCardPackPurchasedEventJSON("TST-0021")))

					assert.Equal(t, http.StatusOK, recorder.Code)
					assert.Equal(t, 0, recorder.Body.Len())
				})
				t.Run("カードパックTST-0021にTST-0001が3枚、カードパックTST-0022にTST-0002が2枚あり、プレイヤーAがTST-0021を購入したイベントを同じ内容で再度送ったあとに、プレイヤーAのカード一覧を取得すると", func(t *testing.T) {
					requestPlayerCardsAfterRedelivery := func(t *testing.T) []any {
						h := newHarnessAfterFailedDelivery(t)
						h.SendCardPackPurchased(routertest.BuildEnvelope(routertest.BuildCardPackPurchasedEventJSON("TST-0021")))
						return routertest.DecodeArray(t, h.GetPlayerCards())
					}

					t.Run("応答のcard_idはTST-0001だけになる", func(t *testing.T) {
						cards := requestPlayerCardsAfterRedelivery(t)

						assert.Equal(t, []any{"TST-0001"}, routertest.CollectValues(t, cards, "card_id"))
					})
					t.Run("応答のカードTST-0001のcountは3になる", func(t *testing.T) {
						cards := requestPlayerCardsAfterRedelivery(t)

						assert.EqualValues(t, 3, routertest.FindObject(t, cards, "card_id", "TST-0001")["count"])
					})
				})
			})
		})
	})
}
