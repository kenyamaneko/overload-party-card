package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/router/routertest"
)

func requestOnlyCard(t *testing.T, mutate func(card *domain.Card)) map[string]any {
	masterData := routertest.NewMasterData()
	card := routertest.NewTestCard("TST-0001", "SHE", "unlimited")
	mutate(card)
	masterData.Cards = []*domain.Card{card}
	h := routertest.NewHarnessWithMasterData(t, masterData)
	return routertest.ExtractSingleObject(t, h.SendGetRequest(routertest.CardOwnershipPath))
}

func TestCardOwnershipList(t *testing.T) {
	t.Run("[カード定義]所持状態付きカード定義一覧の取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("カード定義にTST-0001とTST-0002があり、プレイヤーAのトークンで要求したとき", func(t *testing.T) {
				t.Run("プレイヤーAがTST-0001だけを所持していると", func(t *testing.T) {
					ownershipTests := []struct {
						name        string
						cardID      string
						wantIsOwned bool
					}{
						{"応答のカードTST-0001のis_ownedはtrueになる", "TST-0001", true},
						{"応答のカードTST-0002のis_ownedはfalseになる", "TST-0002", false},
					}
					for _, tc := range ownershipTests {
						t.Run(tc.name, func(t *testing.T) {
							masterData := routertest.NewMasterData()
							masterData.Cards = []*domain.Card{routertest.NewTestCard("TST-0001", "SHE", "unlimited"), routertest.NewTestCard("TST-0002", "SHE", "unlimited")}
							h := routertest.NewHarnessWithMasterData(t, masterData)
							h.DB.SetOwned(routertest.PlayerA, "TST-0001", 0, 1)

							cards := routertest.DecodeArray(t, h.SendGetRequest(routertest.CardOwnershipPath))

							assert.Equal(t, tc.wantIsOwned, routertest.FindObject(t, cards, "card_id", tc.cardID)["is_owned"])
						})
					}
				})
			})
			t.Run("プレイヤーAがTST-0001のアート0とアート1を所持しているとき、応答のcard_idがTST-0001のカードは1件になる", func(t *testing.T) {
				h := routertest.NewHarness(t)
				h.DB.SetOwned(routertest.PlayerA, "TST-0001", 0, 1)
				h.DB.SetOwned(routertest.PlayerA, "TST-0001", 1, 1)

				cards := routertest.DecodeArray(t, h.SendGetRequest(routertest.CardOwnershipPath))

				assert.Equal(t, 1, routertest.CountValues(routertest.CollectValues(t, cards, "card_id"), "TST-0001"))
			})
			t.Run("任意項目", func(t *testing.T) {
				t.Run("サブタイプがVMのとき、応答のカードのsubtypeはVMになる", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.Subtype = routertest.NewPointer("VM") })

					assert.Equal(t, "VM", card["subtype"])
				})
				t.Run("サブタイプが未設定のとき、応答のカードにsubtypeの項目が含まれない", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.Subtype = nil })

					assert.NotContains(t, card, "subtype")
				})
				t.Run("効果テキストがテスト効果文Aのとき、応答のカードのeffect_textはテスト効果文Aになる", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.EffectText = routertest.NewPointer("テスト効果文A") })

					assert.Equal(t, "テスト効果文A", card["effect_text"])
				})
				t.Run("効果テキストが未設定のとき、応答のカードにeffect_textの項目が含まれない", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.EffectText = nil })

					assert.NotContains(t, card, "effect_text")
				})
				t.Run(`効果定義が[{"trigger":"on_deploy"}]のJSONの配列のとき、応答のカードのeffectsは[{"trigger":"on_deploy"}]と同じ内容になる`, func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.Effects = []map[string]any{{"trigger": "on_deploy"}} })

					assert.JSONEq(t, `[{"trigger":"on_deploy"}]`, routertest.EncodeJSON(t, card["effects"]))
				})
				t.Run("効果定義が空の配列のとき、応答のカードのeffectsは空の配列になる", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.Effects = []map[string]any{} })

					assert.Equal(t, []any{}, card["effects"])
				})
				t.Run("効果定義が未設定のとき、応答のカードにeffectsの項目が含まれない", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.Effects = nil })

					assert.NotContains(t, card, "effects")
				})
			})
			t.Run("カード定義が1件保存されているとき", func(t *testing.T) {
				t.Run("card_idがTST-0001だと、応答のカードのcard_idはTST-0001になる", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.CardID = "TST-0001" })

					assert.Equal(t, "TST-0001", card["card_id"])
				})
				t.Run("card_nameがテストカードAだと、応答のカードのcard_nameはテストカードAになる", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.CardName = "テストカードA" })

					assert.Equal(t, "テストカードA", card["card_name"])
				})
				t.Run("resource_labelがテストラベルAだと、応答のカードのresource_labelはテストラベルAになる", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.ResourceLabel = "テストラベルA" })

					assert.Equal(t, "テストラベルA", card["resource_label"])
				})
				t.Run("factionがSHEだと、応答のカードのfactionはSHEになる", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.Faction = "SHE" })

					assert.Equal(t, "SHE", card["faction"])
				})
				t.Run("card_typeがComputeだと、応答のカードのcard_typeはComputeになる", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.CardType = "Compute" })

					assert.Equal(t, "Compute", card["card_type"])
				})
				t.Run("resizableがtrueでelasticがfalseだと", func(t *testing.T) {
					requestResizableCard := func(t *testing.T) map[string]any {
						return requestOnlyCard(t, func(card *domain.Card) {
							card.IsResizable = true
							card.IsElastic = false
						})
					}

					t.Run("応答のカードのresizableはtrueになる", func(t *testing.T) {
						card := requestResizableCard(t)

						assert.Equal(t, true, card["resizable"])
					})
					t.Run("応答のカードのelasticはfalseになる", func(t *testing.T) {
						card := requestResizableCard(t)

						assert.Equal(t, false, card["elastic"])
					})
				})
				t.Run(`statsが{"throughput":700,"availability":1400}だと、応答のカードのstatsは{"throughput":700,"availability":1400}と同じ内容になる`, func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) {
						card.Stats = domain.CardStats{Throughput: routertest.NewPointer(int64(700)), Availability: routertest.NewPointer(int64(1400))}
					})

					assert.JSONEq(t, `{"throughput":700,"availability":1400}`, routertest.EncodeJSON(t, card["stats"]))
				})
				t.Run("restrictionがsemi_limitedだと、応答のカードのrestrictionはsemi_limitedになる", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.Restriction = "semi_limited" })

					assert.Equal(t, "semi_limited", card["restriction"])
				})
				t.Run("is_activeがtrueだと、応答のカードのis_activeはtrueになる", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.IsActive = true })

					assert.Equal(t, true, card["is_active"])
				})
				t.Run("is_activeがfalseだと、応答のカードのis_activeはfalseになる", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.IsActive = false })

					assert.Equal(t, false, card["is_active"])
				})
				t.Run("created_atが2026-01-02T03:04:05Zだと、応答のカードのcreated_atは2026-01-02T03:04:05Zになる", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.CreatedAt = routertest.StoredCreatedAt })

					assert.Equal(t, "2026-01-02T03:04:05Z", card["created_at"])
				})
				t.Run("updated_atが2026-02-03T04:05:06Zだと、応答のカードのupdated_atは2026-02-03T04:05:06Zになる", func(t *testing.T) {
					card := requestOnlyCard(t, func(card *domain.Card) { card.UpdatedAt = routertest.StoredUpdatedAt })

					assert.Equal(t, "2026-02-03T04:05:06Z", card["updated_at"])
				})
			})
			t.Run("カード定義が0件のとき、応答は空の配列になる", func(t *testing.T) {
				h := routertest.NewHarness(t)
				h.DB.Cards = nil

				recorder := h.SendGetRequest(routertest.CardOwnershipPath)

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, "[]", recorder.Body.String())
			})
		})
		t.Run("異常系", func(t *testing.T) {
			t.Run("所持状態付きカード定義一覧を要求したとき", func(t *testing.T) {
				routertest.RunInternalErrorCases(t, routertest.NewHarness, func(h *routertest.Harness) *httptest.ResponseRecorder {
					return h.SendGetRequest(routertest.CardOwnershipPath)
				}, []routertest.InternalErrorCase{
					{
						Given:   "カード定義の取得がcard-find-all-failedのエラーで失敗していると",
						Cause:   "card-find-all-failed",
						Arrange: func(h *routertest.Harness) { h.DB.ErrFindAllCards = routertest.ErrCardFindAllFailed },
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
