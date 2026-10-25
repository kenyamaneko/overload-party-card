package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/router/routertest"
)

func requestDeckDetail(t *testing.T, mutate func(deck *domain.Deck)) map[string]any {
	h := routertest.NewHarness(t)
	deck := routertest.NewStoredDeck(7)
	mutate(&deck)
	h.DB.StoreDeck(deck, nil)
	return routertest.DecodeObject(t, h.GetDeck(7))
}

func TestDeckDetail(t *testing.T) {
	t.Run("[デッキ]デッキ詳細の取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("デッキID7のデッキが存在するとき、デッキID7で要求すると", func(t *testing.T) {
				t.Run("デッキの持ち主のplayer_idが00000000-0000-0000-0000-000000000001だと、応答のplayer_idは00000000-0000-0000-0000-000000000001になる", func(t *testing.T) {
					deck := requestDeckDetail(t, func(deck *domain.Deck) { deck.PlayerID = routertest.PlayerA })

					assert.Equal(t, "00000000-0000-0000-0000-000000000001", deck["player_id"])
				})
				t.Run("応答のdeck_idは7になる", func(t *testing.T) {
					deck := requestDeckDetail(t, func(deck *domain.Deck) {})

					assert.EqualValues(t, 7, deck["deck_id"])
				})
				t.Run("deck_nameがテストデッキAだと、応答のdeck_nameはテストデッキAになる", func(t *testing.T) {
					deck := requestDeckDetail(t, func(deck *domain.Deck) { deck.DeckName = "テストデッキA" })

					assert.Equal(t, "テストデッキA", deck["deck_name"])
				})
				t.Run("factionがSHEだと、応答のfactionはSHEになる", func(t *testing.T) {
					deck := requestDeckDetail(t, func(deck *domain.Deck) { deck.Faction = "SHE" })

					assert.Equal(t, "SHE", deck["faction"])
				})
				t.Run("product_idがTST-0001だと、応答のproduct_idはTST-0001になる", func(t *testing.T) {
					deck := requestDeckDetail(t, func(deck *domain.Deck) { deck.ProductID = "TST-0001" })

					assert.Equal(t, "TST-0001", deck["product_id"])
				})
				t.Run("routine_idがTST-0011だと、応答のroutine_idはTST-0011になる", func(t *testing.T) {
					deck := requestDeckDetail(t, func(deck *domain.Deck) { deck.RoutineID = "TST-0011" })

					assert.Equal(t, "TST-0011", deck["routine_id"])
				})
				t.Run("special_idがTST-0012だと、応答のspecial_idはTST-0012になる", func(t *testing.T) {
					deck := requestDeckDetail(t, func(deck *domain.Deck) { deck.SpecialID = "TST-0012" })

					assert.Equal(t, "TST-0012", deck["special_id"])
				})
				t.Run("created_atが2026-01-02T03:04:05Zだと、応答のcreated_atは2026-01-02T03:04:05Zになる", func(t *testing.T) {
					deck := requestDeckDetail(t, func(deck *domain.Deck) { deck.CreatedAt = routertest.StoredCreatedAt })

					assert.Equal(t, "2026-01-02T03:04:05Z", deck["created_at"])
				})
				t.Run("updated_atが2026-02-03T04:05:06Zだと、応答のupdated_atは2026-02-03T04:05:06Zになる", func(t *testing.T) {
					deck := requestDeckDetail(t, func(deck *domain.Deck) { deck.UpdatedAt = routertest.StoredUpdatedAt })

					assert.Equal(t, "2026-02-03T04:05:06Z", deck["updated_at"])
				})
				t.Run("構成カードがプレイヤーAのTST-0001のアート2の3枚だけだと", func(t *testing.T) {
					requestDeckCard := func(t *testing.T) map[string]any {
						h := routertest.NewHarnessWithStoredDeck(t)
						return routertest.ExtractSingleElement(t, routertest.DecodeObject(t, h.GetDeck(7))["deck_cards"])
					}

					t.Run("応答のdeck_cardsの1件のplayer_idは00000000-0000-0000-0000-000000000001になる", func(t *testing.T) {
						card := requestDeckCard(t)

						assert.Equal(t, "00000000-0000-0000-0000-000000000001", card["player_id"])
					})
					t.Run("応答のdeck_cardsの1件のdeck_idは7になる", func(t *testing.T) {
						card := requestDeckCard(t)

						assert.EqualValues(t, 7, card["deck_id"])
					})
					t.Run("応答のdeck_cardsの1件のcard_idはTST-0001になる", func(t *testing.T) {
						card := requestDeckCard(t)

						assert.Equal(t, "TST-0001", card["card_id"])
					})
					t.Run("応答のdeck_cardsの1件のart_noは2になる", func(t *testing.T) {
						card := requestDeckCard(t)

						assert.EqualValues(t, 2, card["art_no"])
					})
					t.Run("応答のdeck_cardsの1件のcountは3になる", func(t *testing.T) {
						card := requestDeckCard(t)

						assert.EqualValues(t, 3, card["count"])
					})
				})
			})
			t.Run("デッキの有効判定", func(t *testing.T) {
				t.Run("プレイヤーAが、陣営SHE、プロダクトTST-0001、ルーチン施策TST-0011、スペシャル施策TST-0012で、SHEのカードTST-0101からTST-0105とNeutralのカードTST-0111からTST-0115のアート0を各3枚(所持各3枚)の合計30枚のデッキID7を持つとき", func(t *testing.T) {
					t.Run("応答のis_validはtrueになる", func(t *testing.T) {
						h := routertest.NewHarness(t)
						h.StoreValidDeck(7)

						deck := routertest.DecodeObject(t, h.GetDeck(7))

						assert.Equal(t, true, deck["is_valid"])
					})
					t.Run("TST-0115のアート0を2枚に減らして合計29枚にすると、応答のis_validはfalseになる", func(t *testing.T) {
						h := routertest.NewHarness(t)
						composition := routertest.ChangeCompositionCount(routertest.ValidDeckComposition, "TST-0115", 2)
						h.DB.StoreDeck(routertest.NewStoredDeck(7), routertest.BuildDeckCards(routertest.PlayerA, 7, composition))
						h.DB.SetOwnedComposition(routertest.PlayerA, routertest.ValidDeckComposition)

						deck := routertest.DecodeObject(t, h.GetDeck(7))

						assert.Equal(t, false, deck["is_valid"])
					})
				})
			})
		})
		t.Run("異常系", func(t *testing.T) {
			t.Run("デッキIDが整数として読めない、または範囲外のとき", func(t *testing.T) {
				invalidTests := []struct {
					name   string
					deckID string
				}{
					{"デッキIDがabcだと、400の応答になり、本文のerror項目はデッキIDが不正であることを示す", "abc"},
					{"デッキIDが1.5だと、400の応答になり、本文のerror項目はデッキIDが不正であることを示す", "1.5"},
					{"デッキIDが0x10だと、400の応答になり、本文のerror項目はデッキIDが不正であることを示す", "0x10"},
					{"デッキIDが9223372036854775808だと、400の応答になり、本文のerror項目はデッキIDが不正であることを示す", "9223372036854775808"},
				}
				for _, tc := range invalidTests {
					t.Run(tc.name, func(t *testing.T) {
						h := routertest.NewHarness(t)

						recorder := h.SendGetRequest(routertest.DecksPath + "/" + tc.deckID)

						assert.Equal(t, http.StatusBadRequest, recorder.Code)
						assert.JSONEq(t, routertest.InvalidDeckIDBody, recorder.Body.String())
					})
				}
			})
			t.Run("指定したデッキIDのデッキが存在しないとき", func(t *testing.T) {
				notFoundTests := []struct {
					name   string
					deckID string
				}{
					{"デッキIDが7だと、404の応答になり、本文のerror項目はデッキが見つからないことを示す", "7"},
					{"デッキIDが9223372036854775807だと、404の応答になり、本文のerror項目はデッキが見つからないことを示す", "9223372036854775807"},
				}
				for _, tc := range notFoundTests {
					t.Run(tc.name, func(t *testing.T) {
						h := routertest.NewHarness(t)

						recorder := h.SendGetRequest(routertest.DecksPath + "/" + tc.deckID)

						assert.Equal(t, http.StatusNotFound, recorder.Code)
						assert.Contains(t, routertest.DecodeObject(t, recorder)["error"], "not found")
					})
				}
			})
			routertest.RunInternalErrorCases(t, routertest.NewHarnessWithStoredDeck, func(h *routertest.Harness) *httptest.ResponseRecorder {
				return h.GetDeck(7)
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
