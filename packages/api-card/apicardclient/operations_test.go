package apicardclient_test

import (
	"net/http"
	"testing"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	"github.com/kenyamaneko/overload-party-card/packages/api-card/apicardclient"
	"github.com/kenyamaneko/overload-party-card/packages/api-card/apicardclient/apicardclienttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	t.Run("[カードAPI死活監視]ヘルスチェック", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("ヘルスチェックを呼ぶと、カードAPIにメソッドGETでパス/healthの要求が届く", func(t *testing.T) {
				recorder := &apicardclienttest.RequestRecorder{}
				client := apicardclienttest.NewClientFor(t, recorder.Record(apicardclienttest.RespondJSON(http.StatusOK, `{"status":"ok"}`)))

				_, err := client.GetHealth(t.Context())

				require.NoError(t, err)
				assert.Equal(t, []apicardclienttest.ReceivedRequest{{Method: "GET", Path: "/health"}}, recorder.Requests())
			})

			t.Run("カードAPIがステータス200で本文のstatusがokの応答を返したとき、ヘルスチェックが返す応答のstatusはokになる", func(t *testing.T) {
				client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondJSON(http.StatusOK, `{"status":"ok"}`))

				health, err := client.GetHealth(t.Context())

				require.NoError(t, err)
				assert.Equal(t, "ok", health.Status)
			})
		})
	})
}

func TestPlayerCardList(t *testing.T) {
	t.Run("[所持カード]所持カード一覧の取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("所持カード一覧を取得すると、カードAPIにメソッドGETでパス/api/v1/cards/cardsの要求が届く", func(t *testing.T) {
				recorder := &apicardclienttest.RequestRecorder{}
				client := apicardclienttest.NewClientFor(t, recorder.Record(apicardclienttest.RespondJSON(http.StatusOK, `[]`)))

				_, err := client.ListPlayerCards(t.Context())

				require.NoError(t, err)
				assert.Equal(t, []apicardclienttest.ReceivedRequest{{Method: "GET", Path: "/api/v1/cards/cards"}}, recorder.Requests())
			})

			t.Run("カードAPIが所持枚数3のTST-0001と所持枚数1のTST-0002を返したとき", func(t *testing.T) {
				body := `[{"card_id":"TST-0001","count":3},{"card_id":"TST-0002","count":1}]`

				t.Run("返った一覧のTST-0001の所持枚数は3になる", func(t *testing.T) {
					client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondJSON(http.StatusOK, body))

					cards, err := client.ListPlayerCards(t.Context())

					require.NoError(t, err)
					assert.Equal(t, 3, apicardclienttest.CollectCountsByCardID(cards)["TST-0001"])
				})

				t.Run("返った一覧のTST-0002の所持枚数は1になる", func(t *testing.T) {
					client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondJSON(http.StatusOK, body))

					cards, err := client.ListPlayerCards(t.Context())

					require.NoError(t, err)
					assert.Equal(t, 1, apicardclienttest.CollectCountsByCardID(cards)["TST-0002"])
				})
			})
		})
	})
}

func TestCardOwnershipList(t *testing.T) {
	t.Run("[カード所持状況]所持状態つきカード定義一覧の取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("所持状態つきカード定義一覧を取得すると、カードAPIにメソッドGETでパス/api/v1/cards/cards/with-ownershipの要求が届く", func(t *testing.T) {
				recorder := &apicardclienttest.RequestRecorder{}
				client := apicardclienttest.NewClientFor(t, recorder.Record(apicardclienttest.RespondJSON(http.StatusOK, `[]`)))

				_, err := client.ListCardsWithOwnership(t.Context())

				require.NoError(t, err)
				assert.Equal(t, []apicardclienttest.ReceivedRequest{{Method: "GET", Path: "/api/v1/cards/cards/with-ownership"}}, recorder.Requests())
			})

			t.Run("カードAPIが所持済みのTST-0001と未所持のTST-0002を返したとき", func(t *testing.T) {
				body := `[{"card_id":"TST-0001","is_owned":true},{"card_id":"TST-0002","is_owned":false}]`

				t.Run("返った一覧のTST-0001の所持状態は所持済みになる", func(t *testing.T) {
					client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondJSON(http.StatusOK, body))

					cards, err := client.ListCardsWithOwnership(t.Context())

					require.NoError(t, err)
					assert.True(t, apicardclienttest.CollectOwnershipByCardID(cards)["TST-0001"])
				})

				t.Run("返った一覧のTST-0002の所持状態は未所持になる", func(t *testing.T) {
					client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondJSON(http.StatusOK, body))

					cards, err := client.ListCardsWithOwnership(t.Context())

					require.NoError(t, err)
					isOwned, hasEntry := apicardclienttest.CollectOwnershipByCardID(cards)["TST-0002"]
					assert.True(t, hasEntry)
					assert.False(t, isOwned)
				})
			})
		})
	})
}

func TestDeckList(t *testing.T) {
	t.Run("[デッキ]デッキ一覧の取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("デッキ一覧を取得すると、カードAPIにメソッドGETでパス/api/v1/cards/decksの要求が届く", func(t *testing.T) {
				recorder := &apicardclienttest.RequestRecorder{}
				client := apicardclienttest.NewClientFor(t, recorder.Record(apicardclienttest.RespondJSON(http.StatusOK, `[]`)))

				_, err := client.ListDecks(t.Context())

				require.NoError(t, err)
				assert.Equal(t, []apicardclienttest.ReceivedRequest{{Method: "GET", Path: "/api/v1/cards/decks"}}, recorder.Requests())
			})

			t.Run("カードAPIがデッキID42とデッキID7のデッキをこの順に返したとき、返った一覧のデッキIDの並びは42、7の順になる", func(t *testing.T) {
				client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondJSON(http.StatusOK, `[{"deck_id":42},{"deck_id":7}]`))

				decks, err := client.ListDecks(t.Context())

				require.NoError(t, err)
				assert.Equal(t, []int64{42, 7}, apicardclienttest.ListDeckIDs(decks))
			})
		})
	})
}

func TestDeckGet(t *testing.T) {
	t.Run("[デッキ]デッキ詳細の取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("デッキID42でデッキ詳細を取得すると、カードAPIにメソッドGETでパス/api/v1/cards/decks/42の要求が届く", func(t *testing.T) {
				recorder := &apicardclienttest.RequestRecorder{}
				client := apicardclienttest.NewClientFor(t, recorder.Record(apicardclienttest.RespondJSON(http.StatusOK, `{"deck_id":42}`)))

				_, err := client.GetDeck(t.Context(), 42)

				require.NoError(t, err)
				assert.Equal(t, []apicardclienttest.ReceivedRequest{{Method: "GET", Path: "/api/v1/cards/decks/42"}}, recorder.Requests())
			})

			t.Run("カードAPIがTST-0001を2枚とTST-0002を1枚含むカード構成つきのデッキを返したとき", func(t *testing.T) {
				body := `{"deck_id":42,"deck_cards":[{"card_id":"TST-0001","count":2},{"card_id":"TST-0002","count":1}]}`

				t.Run("返ったデッキで、deck_cardsに含まれるTST-0001のcountは2になる", func(t *testing.T) {
					client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondJSON(http.StatusOK, body))

					deck, err := client.GetDeck(t.Context(), 42)

					require.NoError(t, err)
					require.NotNil(t, deck.DeckCards)
					assert.Equal(t, 2, apicardclienttest.CollectDeckCardCounts(deck)["TST-0001"])
				})

				t.Run("返ったデッキで、deck_cardsに含まれるTST-0002のcountは1になる", func(t *testing.T) {
					client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondJSON(http.StatusOK, body))

					deck, err := client.GetDeck(t.Context(), 42)

					require.NoError(t, err)
					require.NotNil(t, deck.DeckCards)
					assert.Equal(t, 1, apicardclienttest.CollectDeckCardCounts(deck)["TST-0002"])
				})
			})
		})
	})
}

func TestDeckCreate(t *testing.T) {
	t.Run("[デッキ]デッキの作成", func(t *testing.T) {
		createRequest := apicard.DeckCreateRequest{DeckName: "要求デッキ", Faction: "Tenki"}

		t.Run("正常系", func(t *testing.T) {
			t.Run("デッキを作成すると、カードAPIにメソッドPOSTでパス/api/v1/cards/decksの要求が届く", func(t *testing.T) {
				recorder := &apicardclienttest.RequestRecorder{}
				client := apicardclienttest.NewClientFor(t, recorder.Record(apicardclienttest.RespondJSON(http.StatusCreated, `{"deck_id":1}`)))

				_, err := client.CreateDeck(t.Context(), createRequest)

				require.NoError(t, err)
				assert.Equal(t, []apicardclienttest.ReceivedRequest{{Method: "POST", Path: "/api/v1/cards/decks"}}, recorder.Requests())
			})

			t.Run("カードAPIがステータス201で、デッキ名が応答デッキのデッキを返したとき、デッキを作成すると、返ったデッキのdeck_nameは応答デッキになる", func(t *testing.T) {
				client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondJSON(http.StatusCreated, `{"deck_id":1,"deck_name":"応答デッキ"}`))

				deck, err := client.CreateDeck(t.Context(), createRequest)

				require.NoError(t, err)
				assert.Equal(t, "応答デッキ", deck.DeckName)
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("カードAPIが作成成功のステータス201ではなくステータス200を返したとき、予期しない応答を示すエラーになる", func(t *testing.T) {
				client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondJSON(http.StatusOK, `{"deck_id":1,"deck_name":"応答デッキ"}`))

				_, err := client.CreateDeck(t.Context(), createRequest)

				apicardclienttest.AssertUnexpectedResponseError(t, err)
			})
		})
	})
}

func TestDeckUpdate(t *testing.T) {
	t.Run("[デッキ]デッキの更新", func(t *testing.T) {
		updateRequest := apicard.DeckUpdateRequest{DeckName: "更新後デッキ", Faction: "Tenki"}

		t.Run("正常系", func(t *testing.T) {
			t.Run("デッキID7でデッキを更新すると、カードAPIにメソッドPUTでパス/api/v1/cards/decks/7の要求が届く", func(t *testing.T) {
				recorder := &apicardclienttest.RequestRecorder{}
				client := apicardclienttest.NewClientFor(t, recorder.Record(apicardclienttest.RespondJSON(http.StatusOK, `{"deck_id":7}`)))

				_, err := client.UpdateDeck(t.Context(), 7, updateRequest)

				require.NoError(t, err)
				assert.Equal(t, []apicardclienttest.ReceivedRequest{{Method: "PUT", Path: "/api/v1/cards/decks/7"}}, recorder.Requests())
			})

			t.Run("カードAPIがステータス200で、デッキ名が応答デッキのデッキを返したとき、デッキを更新すると、返ったデッキのdeck_nameは応答デッキになる", func(t *testing.T) {
				client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondJSON(http.StatusOK, `{"deck_id":7,"deck_name":"応答デッキ"}`))

				deck, err := client.UpdateDeck(t.Context(), 7, updateRequest)

				require.NoError(t, err)
				assert.Equal(t, "応答デッキ", deck.DeckName)
			})
		})
	})
}

func TestDeckDelete(t *testing.T) {
	t.Run("[デッキ]デッキの削除", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("デッキID7でデッキを削除すると、カードAPIにメソッドDELETEでパス/api/v1/cards/decks/7の要求が届く", func(t *testing.T) {
				recorder := &apicardclienttest.RequestRecorder{}
				client := apicardclienttest.NewClientFor(t, recorder.Record(apicardclienttest.RespondEmpty(http.StatusNoContent)))

				err := client.DeleteDeck(t.Context(), 7)

				require.NoError(t, err)
				assert.Equal(t, []apicardclienttest.ReceivedRequest{{Method: "DELETE", Path: "/api/v1/cards/decks/7"}}, recorder.Requests())
			})

			t.Run("カードAPIがステータス204を返したとき、エラーにならない", func(t *testing.T) {
				client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondEmpty(http.StatusNoContent))

				err := client.DeleteDeck(t.Context(), 7)

				assert.NoError(t, err)
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("カードAPIが削除成功のステータス204ではなくステータス200を返したとき、予期しない応答を示すエラーになる", func(t *testing.T) {
				client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondEmpty(http.StatusOK))

				err := client.DeleteDeck(t.Context(), 7)

				apicardclienttest.AssertUnexpectedResponseError(t, err)
			})
		})
	})
}

func TestDeckBattleValidation(t *testing.T) {
	t.Run("[デッキ]バトル使用可否の検証", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("デッキID7でバトル使用可否を検証すると、カードAPIにメソッドPOSTでパス/api/v1/cards/decks/7/validate-for-battleの要求が届く", func(t *testing.T) {
				recorder := &apicardclienttest.RequestRecorder{}
				client := apicardclienttest.NewClientFor(t, recorder.Record(apicardclienttest.RespondEmpty(http.StatusOK)))

				err := client.ValidateDeckForBattle(t.Context(), 7)

				require.NoError(t, err)
				assert.Equal(t, []apicardclienttest.ReceivedRequest{{Method: "POST", Path: "/api/v1/cards/decks/7/validate-for-battle"}}, recorder.Requests())
			})

			t.Run("カードAPIが本文なしのステータス200を返したとき、エラーにならない", func(t *testing.T) {
				client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondEmpty(http.StatusOK))

				err := client.ValidateDeckForBattle(t.Context(), 7)

				assert.NoError(t, err)
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("カードAPIがステータス400を返したとき、デッキがバトルで使えないことを示すエラーになる", func(t *testing.T) {
				client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondEmpty(http.StatusBadRequest))

				err := client.ValidateDeckForBattle(t.Context(), 7)

				assert.ErrorIs(t, err, apicardclient.ErrDeckInvalid)
			})

			t.Run("カードAPIが検証成功のステータス200ではなくステータス204を返したとき、予期しない応答を示すエラーになる", func(t *testing.T) {
				client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondEmpty(http.StatusNoContent))

				err := client.ValidateDeckForBattle(t.Context(), 7)

				apicardclienttest.AssertUnexpectedResponseError(t, err)
			})
		})
	})
}
