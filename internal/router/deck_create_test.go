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

func requestCreatedDeck(t *testing.T, body routertest.DeckRequestBody) map[string]any {
	return routertest.DecodeObject(t, routertest.NewHarnessOwningValidDeckCards(t).CreateDeck(t, body))
}

func TestDeckCreate(t *testing.T) {
	t.Run("[デッキ]デッキの作成", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("プレイヤーAが陣営SHEを所持し、陣営SHE、プロダクトTST-0001、ルーチン施策TST-0011、スペシャル施策TST-0012で、SHEのカードTST-0101からTST-0105とNeutralのカードTST-0111からTST-0115のアート0を各3枚(所持各3枚)の合計30枚のデッキを作成するとき", func(t *testing.T) {
				t.Run("デッキを作成すると、201の応答になる", func(t *testing.T) {
					h := routertest.NewHarnessOwningValidDeckCards(t)

					recorder := h.CreateDeck(t, routertest.NewValidDeckRequestBody())

					assert.Equal(t, http.StatusCreated, recorder.Code)
				})
				t.Run("プレイヤーAのトークンでデッキを作成すると、応答のplayer_idは00000000-0000-0000-0000-000000000001になる", func(t *testing.T) {
					deck := requestCreatedDeck(t, routertest.NewValidDeckRequestBody())

					assert.Equal(t, "00000000-0000-0000-0000-000000000001", deck["player_id"])
				})
				t.Run("デッキのデータベースがデッキID42を採番すると、応答のdeck_idは42になる", func(t *testing.T) {
					deck := requestCreatedDeck(t, routertest.NewValidDeckRequestBody())

					assert.EqualValues(t, 42, deck["deck_id"])
				})
				t.Run("デッキ名をテストデッキAにして作成すると", func(t *testing.T) {
					requestDeckNamedA := func(t *testing.T) map[string]any {
						body := routertest.NewValidDeckRequestBody()
						body.DeckName = "テストデッキA"
						return requestCreatedDeck(t, body)
					}

					t.Run("応答のdeck_nameはテストデッキAになる", func(t *testing.T) {
						assert.Equal(t, "テストデッキA", requestDeckNamedA(t)["deck_name"])
					})
					t.Run("応答のfactionはSHEになる", func(t *testing.T) {
						assert.Equal(t, "SHE", requestDeckNamedA(t)["faction"])
					})
					t.Run("応答のproduct_idはTST-0001になる", func(t *testing.T) {
						assert.Equal(t, "TST-0001", requestDeckNamedA(t)["product_id"])
					})
					t.Run("応答のroutine_idはTST-0011になる", func(t *testing.T) {
						assert.Equal(t, "TST-0011", requestDeckNamedA(t)["routine_id"])
					})
					t.Run("応答のspecial_idはTST-0012になる", func(t *testing.T) {
						assert.Equal(t, "TST-0012", requestDeckNamedA(t)["special_id"])
					})
				})
				t.Run("構成カードをSHEのカードTST-0001のアート2の3枚(所持3枚)だけにして、プレイヤーAのトークンで作成すると", func(t *testing.T) {
					requestDeckCard := func(t *testing.T) map[string]any {
						h := routertest.NewHarness(t)
						h.DB.SetOwned(routertest.PlayerA, "TST-0001", 2, 3)
						body := routertest.NewDeckRequestBody([]routertest.CardCount{{CardID: "TST-0001", ArtNo: 2, Count: 3}})
						return routertest.ExtractSingleElement(t, routertest.DecodeObject(t, h.CreateDeck(t, body))["deck_cards"])
					}

					t.Run("応答のdeck_cardsの1件のplayer_idは00000000-0000-0000-0000-000000000001になる", func(t *testing.T) {
						assert.Equal(t, "00000000-0000-0000-0000-000000000001", requestDeckCard(t)["player_id"])
					})
					t.Run("デッキのデータベースがデッキID42を採番すると、応答のdeck_cardsの1件のdeck_idは42になる", func(t *testing.T) {
						assert.EqualValues(t, 42, requestDeckCard(t)["deck_id"])
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
				t.Run("デッキを作成すると", func(t *testing.T) {
					t.Run("応答のcreated_atは、要求を送った時刻から応答を受け取った時刻までの間の日時になる", func(t *testing.T) {
						h := routertest.NewHarnessOwningValidDeckCards(t)
						sentAt := time.Now()

						recorder := h.CreateDeck(t, routertest.NewValidDeckRequestBody())

						receivedAt := time.Now()
						createdAt, err := time.Parse(time.RFC3339Nano, fmt.Sprint(routertest.DecodeObject(t, recorder)["created_at"]))
						require.NoError(t, err)
						assert.WithinRange(t, createdAt, sentAt, receivedAt)
					})
					t.Run("応答のupdated_atは応答のcreated_atの値になる", func(t *testing.T) {
						deck := requestCreatedDeck(t, routertest.NewValidDeckRequestBody())

						assert.Equal(t, deck["created_at"], deck["updated_at"])
					})
					t.Run("応答のis_validはtrueになる", func(t *testing.T) {
						deck := requestCreatedDeck(t, routertest.NewValidDeckRequestBody())

						assert.Equal(t, true, deck["is_valid"])
					})
				})
				t.Run("TST-0115のアート0を2枚に減らして合計29枚にして作成すると", func(t *testing.T) {
					createTwentyNineCardDeck := func(t *testing.T) *httptest.ResponseRecorder {
						body := routertest.NewValidDeckRequestBody()
						routertest.ReduceToTwentyNineCards(&body)
						return routertest.NewHarnessOwningValidDeckCards(t).CreateDeck(t, body)
					}

					t.Run("201の応答になる", func(t *testing.T) {
						assert.Equal(t, http.StatusCreated, createTwentyNineCardDeck(t).Code)
					})
					t.Run("応答のis_validはfalseになる", func(t *testing.T) {
						assert.Equal(t, false, routertest.DecodeObject(t, createTwentyNineCardDeck(t))["is_valid"])
					})
				})
				t.Run("プレイマット番号5を指定して作成すると、応答のplaymat_noは5になる", func(t *testing.T) {
					body := routertest.NewValidDeckRequestBody()
					body.PlaymatNo = routertest.NewPointer(int64(5))

					deck := requestCreatedDeck(t, body)

					assert.EqualValues(t, 5, deck["playmat_no"])
				})
				t.Run("プレイマット番号を指定せずに作成すると、応答のデッキにplaymat_noの項目が含まれない", func(t *testing.T) {
					body := routertest.NewValidDeckRequestBody()
					body.PlaymatNo = nil

					deck := requestCreatedDeck(t, body)

					assert.NotContains(t, deck, "playmat_no")
				})
				t.Run("スリーブ番号6を指定して作成すると、応答のsleeve_noは6になる", func(t *testing.T) {
					body := routertest.NewValidDeckRequestBody()
					body.SleeveNo = routertest.NewPointer(int64(6))

					deck := requestCreatedDeck(t, body)

					assert.EqualValues(t, 6, deck["sleeve_no"])
				})
				t.Run("スリーブ番号を指定せずに作成すると、応答のデッキにsleeve_noの項目が含まれない", func(t *testing.T) {
					body := routertest.NewValidDeckRequestBody()
					body.SleeveNo = nil

					deck := requestCreatedDeck(t, body)

					assert.NotContains(t, deck, "sleeve_no")
				})
				t.Run("デッキのデータベースがデッキID42を採番したあとにデッキ一覧を取得すると、一覧の応答のdeck_idは42だけになる", func(t *testing.T) {
					h := routertest.NewHarnessOwningValidDeckCards(t)
					h.CreateDeck(t, routertest.NewValidDeckRequestBody())

					decks := routertest.DecodeArray(t, h.GetDecks())

					assert.Equal(t, []any{float64(42)}, routertest.CollectValues(t, decks, "deck_id"))
				})
			})
		})
		t.Run("異常系", func(t *testing.T) {
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
						h := routertest.NewHarnessOwningValidDeckCards(t)

						recorder := h.SendRequestWithToken(http.MethodPost, routertest.DecksPath, routertest.TokenA, tc.body)

						assert.Equal(t, http.StatusBadRequest, recorder.Code)
						assert.JSONEq(t, routertest.MalformedRequestBodyBody, recorder.Body.String())
					})
				}
			})
			t.Run("プレイヤーAが陣営SHEを所持し、陣営SHE、プロダクトTST-0001、ルーチン施策TST-0011、スペシャル施策TST-0012で、SHEのカードTST-0101からTST-0105とNeutralのカードTST-0111からTST-0115のアート0を各3枚(所持各3枚)の合計30枚のデッキを作成するとき", func(t *testing.T) {
				t.Run("NeutralのカードTST-0116のアート0を1枚(所持1枚)足して合計31枚にすると、400の応答になり、本文のerror項目はデッキの枚数が上限を超えていることを示す", func(t *testing.T) {
					h := routertest.NewHarnessOwningValidDeckCards(t)
					body := routertest.NewValidDeckRequestBody()
					routertest.AddThirtyFirstCard(h, &body)

					recorder := h.CreateDeck(t, body)

					assert.Equal(t, http.StatusBadRequest, recorder.Code)
					assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "cannot exceed")
				})
				t.Run("TST-0101のアート0の所持を2枚に減らすと、403の応答になり、本文のerror項目は所持枚数が足りないことを示す", func(t *testing.T) {
					h := routertest.NewHarnessOwningValidDeckCards(t)
					routertest.ReduceOwnedFirstCard(h)

					recorder := h.CreateDeck(t, routertest.NewValidDeckRequestBody())

					assert.Equal(t, http.StatusForbidden, recorder.Code)
					assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "not enough owned")
				})
				t.Run("陣営をNeutralにすると、400の応答になり、本文のerror項目は陣営が選択できないことを示す", func(t *testing.T) {
					h := routertest.NewHarnessOwningValidDeckCards(t)
					body := routertest.NewValidDeckRequestBody()
					routertest.DeclareNeutral(&body)

					recorder := h.CreateDeck(t, body)

					assert.Equal(t, http.StatusBadRequest, recorder.Code)
					assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "is not selectable")
				})
				t.Run("TST-0115のアート0の3枚を、制限区分がlimitedのカードTST-0003のアート0の2枚(所持2枚)に置き換えると、400の応答になり、本文のerror項目は制限枚数を超えていることを示す", func(t *testing.T) {
					h := routertest.NewHarnessOwningValidDeckCards(t)
					body := routertest.NewValidDeckRequestBody()
					routertest.SwapInLimitedCards(h, &body)

					recorder := h.CreateDeck(t, body)

					assert.Equal(t, http.StatusBadRequest, recorder.Code)
					assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "exceeds restriction limit")
				})
			})
			t.Run("プレイヤーAがデッキID7を持ち、構成カードがSHEのカードTST-0101からTST-0105とNeutralのカードTST-0111からTST-0115のアート0を各3枚(所持各3枚)にNeutralのカードTST-0116のアート0を1枚(所持1枚)足した合計31枚のデッキの作成が400で拒否されたあとにデッキ一覧を取得すると、一覧の応答のdeck_idは7だけになる", func(t *testing.T) {
				h := routertest.NewHarnessOwningValidDeckCards(t)
				h.StoreValidDeck(7)
				body := routertest.NewValidDeckRequestBody()
				routertest.AddThirtyFirstCard(h, &body)
				h.CreateDeck(t, body)

				decks := routertest.DecodeArray(t, h.GetDecks())

				assert.Equal(t, []any{float64(7)}, routertest.CollectValues(t, decks, "deck_id"))
			})
			requestBody := routertest.EncodeJSON(t, routertest.NewValidDeckRequestBody())
			routertest.RunInternalErrorCases(t, routertest.NewHarnessOwningValidDeckCards, func(h *routertest.Harness) *httptest.ResponseRecorder {
				return h.SendRequestWithToken(http.MethodPost, routertest.DecksPath, routertest.TokenA, requestBody)
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
