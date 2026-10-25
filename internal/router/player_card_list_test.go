package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/router/routertest"
)

func newHarnessOwningArtTwoCard(t *testing.T) *routertest.Harness {
	h := routertest.NewHarness(t)
	h.DB.SetOwned(routertest.PlayerA, "TST-0001", 2, 3)
	return h
}

func requestOwnedSecondCard(t *testing.T, mutate func(first, second *domain.Card)) map[string]any {
	masterData := routertest.NewMasterData()
	first := routertest.NewTestCard("TST-0001", "SHE", "unlimited")
	second := routertest.NewTestCard("TST-0002", "SHE", "unlimited")
	mutate(first, second)
	masterData.Cards = []*domain.Card{first, second}
	h := routertest.NewHarnessWithMasterData(t, masterData)
	h.DB.SetOwned(routertest.PlayerA, "TST-0002", 0, 1)
	return routertest.ExtractSingleObject(t, h.GetPlayerCards())
}

func TestPlayerCardList(t *testing.T) {
	t.Run("[所持カード]プレイヤーの所持カード一覧の取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("プレイヤーAがTST-0001のアート2を3枚所持しているとき、プレイヤーAのトークンで要求すると", func(t *testing.T) {
				t.Run("応答のカードのcard_idはTST-0001になる", func(t *testing.T) {
					card := routertest.ExtractSingleObject(t, newHarnessOwningArtTwoCard(t).GetPlayerCards())

					assert.Equal(t, "TST-0001", card["card_id"])
				})
				t.Run("応答のカードのart_noは2になる", func(t *testing.T) {
					card := routertest.ExtractSingleObject(t, newHarnessOwningArtTwoCard(t).GetPlayerCards())

					assert.EqualValues(t, 2, card["art_no"])
				})
				t.Run("応答のカードのcountは3になる", func(t *testing.T) {
					card := routertest.ExtractSingleObject(t, newHarnessOwningArtTwoCard(t).GetPlayerCards())

					assert.EqualValues(t, 3, card["count"])
				})
			})
			t.Run("カード定義にTST-0001とTST-0002があり、プレイヤーAがTST-0002だけを所持しているとき", func(t *testing.T) {
				t.Run("TST-0001のcard_nameがテストカードA、TST-0002のcard_nameがテストカードBだと、応答のカードのcard_nameはテストカードBになる", func(t *testing.T) {
					card := requestOwnedSecondCard(t, func(first, second *domain.Card) {
						first.CardName = "テストカードA"
						second.CardName = "テストカードB"
					})

					assert.Equal(t, "テストカードB", card["card_name"])
				})
				t.Run("TST-0001のfactionがSHE、TST-0002のfactionがTenkiだと、応答のカードのfactionはTenkiになる", func(t *testing.T) {
					card := requestOwnedSecondCard(t, func(first, second *domain.Card) {
						first.Faction = "SHE"
						second.Faction = "Tenki"
					})

					assert.Equal(t, "Tenki", card["faction"])
				})
				t.Run("TST-0001のcard_typeがCompute、TST-0002のcard_typeがStrategyだと、応答のカードのcard_typeはStrategyになる", func(t *testing.T) {
					card := requestOwnedSecondCard(t, func(first, second *domain.Card) {
						first.CardType = "Compute"
						second.CardType = "Strategy"
					})

					assert.Equal(t, "Strategy", card["card_type"])
				})
				t.Run("TST-0001のrestrictionがunlimited、TST-0002のrestrictionがlimitedだと、応答のカードのrestrictionはlimitedになる", func(t *testing.T) {
					card := requestOwnedSecondCard(t, func(first, second *domain.Card) {
						first.Restriction = "unlimited"
						second.Restriction = "limited"
					})

					assert.Equal(t, "limited", card["restriction"])
				})
				t.Run("TST-0001の効果テキストがテスト効果文A、TST-0002の効果テキストがテスト効果文Bだと、応答のカードのeffect_textはテスト効果文Bになる", func(t *testing.T) {
					card := requestOwnedSecondCard(t, func(first, second *domain.Card) {
						first.EffectText = routertest.NewPointer("テスト効果文A")
						second.EffectText = routertest.NewPointer("テスト効果文B")
					})

					assert.Equal(t, "テスト効果文B", card["effect_text"])
				})
				t.Run("TST-0001の効果テキストがテスト効果文A、TST-0002の効果テキストが未設定だと、応答のカードにeffect_textの項目が含まれない", func(t *testing.T) {
					card := requestOwnedSecondCard(t, func(first, second *domain.Card) {
						first.EffectText = routertest.NewPointer("テスト効果文A")
						second.EffectText = nil
					})

					assert.NotContains(t, card, "effect_text")
				})
			})
			t.Run("プレイヤーAがTST-0001のアート0とアート1を所持しているとき、応答のカードはart_noが0のものとart_noが1のものの2件になる", func(t *testing.T) {
				h := routertest.NewHarness(t)
				h.DB.SetOwned(routertest.PlayerA, "TST-0001", 0, 1)
				h.DB.SetOwned(routertest.PlayerA, "TST-0001", 1, 1)

				cards := routertest.DecodeArray(t, h.GetPlayerCards())

				assert.Len(t, cards, 2)
				assert.ElementsMatch(t, []any{float64(0), float64(1)}, routertest.CollectValues(t, cards, "art_no"))
			})
			t.Run("所持カードが0件のとき、応答は空の配列になる", func(t *testing.T) {
				h := routertest.NewHarness(t)

				recorder := h.GetPlayerCards()

				assert.Equal(t, http.StatusOK, recorder.Code)
				assert.Equal(t, "[]", recorder.Body.String())
			})
		})
		t.Run("異常系", func(t *testing.T) {
			t.Run("所持カード一覧を要求したとき", func(t *testing.T) {
				t.Run("所持カードの取得がplayer-card-find-failedのエラーで失敗していると", func(t *testing.T) {
					requestWithFailure := func(t *testing.T) (*routertest.Harness, *httptest.ResponseRecorder) {
						h := newHarnessOwningArtTwoCard(t)
						h.DB.ErrGetPlayerCards = routertest.ErrPlayerCardFindFailed
						return h, h.GetPlayerCards()
					}

					t.Run("内部エラーを示す500の応答になる", func(t *testing.T) {
						_, recorder := requestWithFailure(t)

						assert.Equal(t, http.StatusInternalServerError, recorder.Code)
					})
					t.Run("500の応答本文のerror項目は固定の文言になる", func(t *testing.T) {
						_, recorder := requestWithFailure(t)

						assert.Equal(t, "internal server error", routertest.DecodeObject(t, recorder)["error"])
					})
					t.Run("Errorレベルのログのpath項目は/api/v1/cards/cardsになる", func(t *testing.T) {
						h, _ := requestWithFailure(t)

						h.RequireErrorLogEquals(t, "path", "/api/v1/cards/cards")
					})
					t.Run("Errorレベルのログのmethod項目はGETになる", func(t *testing.T) {
						h, _ := requestWithFailure(t)

						h.RequireErrorLogEquals(t, "method", "GET")
					})
					t.Run("Errorレベルのログのerror項目はplayer-card-find-failedを含む", func(t *testing.T) {
						h, _ := requestWithFailure(t)

						h.RequireErrorLogContaining(t, "error", "player-card-find-failed")
					})
				})
				t.Run("プレイヤーAが所持しているTST-0009のカード定義が存在しないと", func(t *testing.T) {
					requestWithMissingDefinition := func(t *testing.T) (*routertest.Harness, *httptest.ResponseRecorder) {
						h := routertest.NewHarness(t)
						h.DB.SetOwned(routertest.PlayerA, "TST-0009", 0, 1)
						return h, h.GetPlayerCards()
					}

					t.Run("内部エラーを示す500の応答になる", func(t *testing.T) {
						_, recorder := requestWithMissingDefinition(t)

						assert.Equal(t, http.StatusInternalServerError, recorder.Code)
					})
					t.Run("Errorレベルのログのerror項目はTST-0009を含む", func(t *testing.T) {
						h, _ := requestWithMissingDefinition(t)

						h.RequireErrorLogContaining(t, "error", "TST-0009")
					})
				})
			})
		})
	})
}
