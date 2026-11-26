package router_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/router/routertest"
)

func requestOnlyDeck(t *testing.T, mutate func(deck *domain.Deck)) map[string]any {
	h := routertest.NewHarness(t)
	deck := routertest.NewStoredDeck(7)
	mutate(&deck)
	h.DB.StoreDeck(deck, nil)
	return routertest.ExtractSingleObject(t, h.GetDecks())
}

func TestDeckList(t *testing.T) {
	t.Run("[デッキ]プレイヤーのデッキ一覧の取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("プレイヤーAがデッキを持たないとき、応答は空の配列になる", func(t *testing.T) {
				h := routertest.NewHarness(t)

				recorder := h.GetDecks()

				assert.Equal(t, "[]", recorder.Body.String())
			})
			t.Run("プレイヤーAのデッキが1件あるとき", func(t *testing.T) {
				t.Run("応答のデッキのplayer_idは00000000-0000-0000-0000-000000000001になる", func(t *testing.T) {
					deck := requestOnlyDeck(t, func(deck *domain.Deck) { deck.PlayerID = routertest.PlayerA })

					assert.Equal(t, "00000000-0000-0000-0000-000000000001", deck["player_id"])
				})
				t.Run("deck_idが7だと、応答のデッキのdeck_idは7になる", func(t *testing.T) {
					deck := requestOnlyDeck(t, func(deck *domain.Deck) { deck.DeckID = 7 })

					assert.EqualValues(t, 7, deck["deck_id"])
				})
				t.Run("deck_nameがテストデッキAだと、応答のデッキのdeck_nameはテストデッキAになる", func(t *testing.T) {
					deck := requestOnlyDeck(t, func(deck *domain.Deck) { deck.DeckName = "テストデッキA" })

					assert.Equal(t, "テストデッキA", deck["deck_name"])
				})
				t.Run("factionがSHEだと、応答のデッキのfactionはSHEになる", func(t *testing.T) {
					deck := requestOnlyDeck(t, func(deck *domain.Deck) { deck.Faction = "SHE" })

					assert.Equal(t, "SHE", deck["faction"])
				})
				t.Run("product_idがTST-0001だと、応答のデッキのproduct_idはTST-0001になる", func(t *testing.T) {
					deck := requestOnlyDeck(t, func(deck *domain.Deck) { deck.ProductID = "TST-0001" })

					assert.Equal(t, "TST-0001", deck["product_id"])
				})
				t.Run("routine_idがTST-0011だと、応答のデッキのroutine_idはTST-0011になる", func(t *testing.T) {
					deck := requestOnlyDeck(t, func(deck *domain.Deck) { deck.RoutineID = "TST-0011" })

					assert.Equal(t, "TST-0011", deck["routine_id"])
				})
				t.Run("special_idがTST-0012だと、応答のデッキのspecial_idはTST-0012になる", func(t *testing.T) {
					deck := requestOnlyDeck(t, func(deck *domain.Deck) { deck.SpecialID = "TST-0012" })

					assert.Equal(t, "TST-0012", deck["special_id"])
				})
				t.Run("created_atが2026-01-02T03:04:05Zだと、応答のデッキのcreated_atは2026-01-02T03:04:05Zになる", func(t *testing.T) {
					deck := requestOnlyDeck(t, func(deck *domain.Deck) { deck.CreatedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) })

					assert.Equal(t, "2026-01-02T03:04:05Z", deck["created_at"])
				})
				t.Run("updated_atが2026-02-03T04:05:06Zだと、応答のデッキのupdated_atは2026-02-03T04:05:06Zになる", func(t *testing.T) {
					deck := requestOnlyDeck(t, func(deck *domain.Deck) { deck.UpdatedAt = time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC) })

					assert.Equal(t, "2026-02-03T04:05:06Z", deck["updated_at"])
				})
			})
			t.Run("プレイヤーAのデッキID1の構成カードがTST-0001、デッキID2の構成カードがTST-0002のとき", func(t *testing.T) {
				requestDecks := func(t *testing.T) []any {
					h := routertest.NewHarness(t)
					h.StoreDeckWithCard(1, routertest.CardCount{CardID: "TST-0001", ArtNo: 0, Count: 1})
					h.StoreDeckWithCard(2, routertest.CardCount{CardID: "TST-0002", ArtNo: 0, Count: 1})
					return routertest.DecodeArray(t, h.GetDecks())
				}

				t.Run("デッキID1のdeck_cardsのcard_idはTST-0001だけになる", func(t *testing.T) {
					decks := requestDecks(t)

					deck := routertest.FindObject(t, decks, "deck_id", float64(1))

					assert.Equal(t, []any{"TST-0001"}, routertest.CollectValues(t, routertest.ToArray(t, deck["deck_cards"]), "card_id"))
				})
				t.Run("デッキID2のdeck_cardsのcard_idはTST-0002だけになる", func(t *testing.T) {
					decks := requestDecks(t)

					deck := routertest.FindObject(t, decks, "deck_id", float64(2))

					assert.Equal(t, []any{"TST-0002"}, routertest.CollectValues(t, routertest.ToArray(t, deck["deck_cards"]), "card_id"))
				})
			})
			t.Run("デッキにプレイマット番号5が設定されているとき、応答のデッキのplaymat_noは5になる", func(t *testing.T) {
				deck := requestOnlyDeck(t, func(deck *domain.Deck) { deck.PlaymatNo = routertest.NewPointer(int64(5)) })

				assert.EqualValues(t, 5, deck["playmat_no"])
			})
			t.Run("デッキにプレイマット番号が未設定のとき、応答のデッキにplaymat_noの項目が含まれない", func(t *testing.T) {
				deck := requestOnlyDeck(t, func(deck *domain.Deck) { deck.PlaymatNo = nil })

				assert.NotContains(t, deck, "playmat_no")
			})
			t.Run("デッキにスリーブ番号6が設定されているとき、応答のデッキのsleeve_noは6になる", func(t *testing.T) {
				deck := requestOnlyDeck(t, func(deck *domain.Deck) { deck.SleeveNo = routertest.NewPointer(int64(6)) })

				assert.EqualValues(t, 6, deck["sleeve_no"])
			})
			t.Run("デッキにスリーブ番号が未設定のとき、応答のデッキにsleeve_noの項目が含まれない", func(t *testing.T) {
				deck := requestOnlyDeck(t, func(deck *domain.Deck) { deck.SleeveNo = nil })

				assert.NotContains(t, deck, "sleeve_no")
			})
			t.Run("デッキの有効判定", func(t *testing.T) {
				t.Run("プレイヤーAが、陣営SHE、プロダクトTST-0001、ルーチン施策TST-0011、スペシャル施策TST-0012で、SHEのカードTST-0101からTST-0105とNeutralのカードTST-0111からTST-0115のアート0を各3枚(所持各3枚)の合計30枚のデッキを持つとき", func(t *testing.T) {
					t.Run("応答のデッキのis_validはtrueになる", func(t *testing.T) {
						h := routertest.NewHarness(t)
						h.StoreValidDeck(7)

						deck := routertest.ExtractSingleObject(t, h.GetDecks())

						assert.Equal(t, true, deck["is_valid"])
					})

					invalidTests := []struct {
						name    string
						arrange func(h *routertest.Harness)
					}{
						{
							"TST-0115のアート0を2枚に減らして合計29枚にすると、応答のデッキのis_validはfalseになる",
							func(h *routertest.Harness) {
								composition := routertest.ChangeCompositionCount(routertest.ValidDeckComposition, "TST-0115", 2)
								h.DB.StoreDeck(routertest.NewStoredDeck(7), routertest.BuildDeckCards(routertest.PlayerA, 7, composition))
								h.DB.SetOwnedComposition(routertest.PlayerA, routertest.ValidDeckComposition)
							},
						},
						{
							"TST-0101のアート0を所持していないと、応答のデッキのis_validはfalseになる",
							func(h *routertest.Harness) {
								h.DB.StoreDeck(routertest.NewStoredDeck(7), routertest.BuildDeckCards(routertest.PlayerA, 7, routertest.ValidDeckComposition))
								h.DB.SetOwnedComposition(routertest.PlayerA, routertest.RemoveFromComposition(routertest.ValidDeckComposition, "TST-0101"))
							},
						},
					}
					for _, tc := range invalidTests {
						t.Run(tc.name, func(t *testing.T) {
							h := routertest.NewHarness(t)
							tc.arrange(h)

							deck := routertest.ExtractSingleObject(t, h.GetDecks())

							assert.Equal(t, false, deck["is_valid"])
						})
					}
				})
			})
		})
		t.Run("異常系", func(t *testing.T) {
			t.Run("デッキ一覧を要求したとき", func(t *testing.T) {
				routertest.RunInternalErrorCases(t, routertest.NewHarnessWithStoredDeck, func(h *routertest.Harness) *httptest.ResponseRecorder {
					return h.GetDecks()
				}, []routertest.InternalErrorCase{
					{
						Given:   "デッキの取得がdeck-find-failedのエラーで失敗していると",
						Cause:   "deck-find-failed",
						Arrange: func(h *routertest.Harness) { h.DB.ErrFindDeck = routertest.ErrDeckFindFailed },
					},
					{
						Given:   "デッキの構成カードの取得がdeck-card-find-failedのエラーで失敗していると",
						Cause:   "deck-card-find-failed",
						Arrange: func(h *routertest.Harness) { h.DB.ErrGetDeckCards = routertest.ErrDeckCardFindFailed },
					},
					{
						Given:   "所持カードの取得がplayer-card-find-failedのエラーで失敗していると",
						Cause:   "player-card-find-failed",
						Arrange: func(h *routertest.Harness) { h.DB.ErrGetPlayerCards = routertest.ErrPlayerCardFindFailed },
					},
				})
			})
		})
	})
}
