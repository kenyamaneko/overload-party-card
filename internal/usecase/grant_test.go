package usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/usecase/usecasetest"
)

func newPurchaseEvent(packID string) domain.CardPackPurchasedEvent {
	return domain.CardPackPurchasedEvent{
		EventID:    "TST-EVENT-0001",
		EventType:  "card_pack_purchased",
		PlayerID:   usecasetest.PlayerA,
		CardPackID: packID,
	}
}

func newOnboardedEvent(initialFaction string) domain.PlayerOnboardedEvent {
	return domain.PlayerOnboardedEvent{
		EventID:          "TST-EVENT-0002",
		EventType:        "player_onboarded",
		PlayerID:         usecasetest.PlayerA,
		InitialFactionID: initialFaction,
	}
}

func newPurchaseGrantHarness() *grantHarness {
	return newGrantHarness(
		newPack("TST-0100", true, newPackCard("TST-0001", 3), newPackCard("TST-0002", 2)),
		newPack("TST-0101", false, newPackCard("TST-0001", 1)),
		newPack("TST-0102", true),
		newPack("TST-0103", true, newPackCard("TST-0001", 1)),
	)
}

func newInitialGrantHarness() *grantHarness {
	return newGrantHarness(
		newPack("basic", true, newPackCard("TST-0001", 1), newPackCard("TST-0002", 2)),
		newPack("faction_set_she", true, newPackCard("TST-0003", 3)),
	)
}

func TestPurchasedCardPackGrant(t *testing.T) {
	t.Run("[カードパック]購入されたカードパックの配布", func(t *testing.T) {
		t.Run("カードパック購入イベントを受け取って、カードパックを配布する", func(t *testing.T) {
			t.Run("正常系", func(t *testing.T) {
				t.Run("パックTST-0100がTST-0001を3枚、TST-0002を2枚内包するとき", func(t *testing.T) {
					t.Run("プレイヤーAがパックTST-0100を購入したイベントを受け取ると", func(t *testing.T) {
						t.Run("配布した合計枚数が5になる", func(t *testing.T) {
							h := newPurchaseGrantHarness()

							granted, err := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0100"))

							assert.NoError(t, err)
							assert.Equal(t, 5, granted)
						})

						t.Run("プレイヤーAの所持カードが、TST-0001のアート0が3枚、TST-0002のアート0が2枚になる", func(t *testing.T) {
							h := newPurchaseGrantHarness()

							_, err := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0100"))

							assert.NoError(t, err)
							assert.ElementsMatch(t,
								[]domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 0, 3), usecasetest.NewOwnedCard("TST-0002", 0, 2)},
								h.playerCards.FetchOwnedCards(t, usecasetest.PlayerA),
							)
						})
					})

					t.Run("プレイヤーAがパックTST-0100を購入した同じイベントを続けて2回受け取ったとき", func(t *testing.T) {
						t.Run("2回目の配布した合計枚数が0になる", func(t *testing.T) {
							h := newPurchaseGrantHarness()
							_, firstErr := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0100"))
							require.NoError(t, firstErr)

							granted, _ := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0100"))

							assert.Equal(t, 0, granted)
						})

						t.Run("2回目はエラーにならない", func(t *testing.T) {
							h := newPurchaseGrantHarness()
							_, firstErr := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0100"))
							require.NoError(t, firstErr)

							_, err := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0100"))

							assert.NoError(t, err)
						})

						t.Run("2回目のあと、プレイヤーAの所持カードが、TST-0001のアート0が3枚、TST-0002のアート0が2枚だけのままになる", func(t *testing.T) {
							h := newPurchaseGrantHarness()
							_, firstErr := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0100"))
							require.NoError(t, firstErr)

							_, _ = h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0100"))

							assert.ElementsMatch(t,
								[]domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 0, 3), usecasetest.NewOwnedCard("TST-0002", 0, 2)},
								h.playerCards.FetchOwnedCards(t, usecasetest.PlayerA),
							)
						})
					})
				})

				t.Run("パックTST-0103がTST-0001を1枚だけ内包するとき、プレイヤーAがパックTST-0103を購入したイベントを受け取ると、配布した合計枚数が1になる", func(t *testing.T) {
					h := newPurchaseGrantHarness()

					granted, err := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0103"))

					assert.NoError(t, err)
					assert.Equal(t, 1, granted)
				})
			})

			t.Run("異常系", func(t *testing.T) {
				t.Run("存在しないパックTST-9999を購入したとき、見つからないことを示すエラーになる", func(t *testing.T) {
					h := newPurchaseGrantHarness()

					_, err := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-9999"))

					assert.ErrorIs(t, err, port.ErrNotFound)
				})

				t.Run("運用停止中のパックTST-0101を購入したとき、運用停止中であることを示すエラーになる", func(t *testing.T) {
					h := newPurchaseGrantHarness()

					_, err := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0101"))

					assert.ErrorIs(t, err, port.ErrPackInactive)
				})

				t.Run("内包カードが0件のパックTST-0102を購入したとき、内包カードが無いことを示すエラーになる", func(t *testing.T) {
					h := newPurchaseGrantHarness()

					_, err := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0102"))

					assert.ErrorIs(t, err, port.ErrEmptyPack)
				})

				t.Run("依存先がエラーを返したとき", func(t *testing.T) {
					tests := []struct {
						name    string
						failure error
						inject  func(h *grantHarness, failure error)
					}{
						{
							name:    "処理済みイベントのデータベースが記録時に「配布時のイベント記録の失敗」のエラーを返したとき、エラーの原因が「配布時のイベント記録の失敗」のエラーになる",
							failure: errors.New("配布時のイベント記録の失敗"),
							inject:  func(h *grantHarness, failure error) { h.processedEvents.insertErr = failure },
						},
						{
							name:    "パックのデータベースがパック取得時に、存在しない以外のエラーとして「配布時のパック取得の失敗」のエラーを返したとき、エラーの原因が「配布時のパック取得の失敗」のエラーになる",
							failure: errors.New("配布時のパック取得の失敗"),
							inject:  func(h *grantHarness, failure error) { h.cardPacks.getErr = failure },
						},
						{
							name:    "所持カードのデータベースが付与時に「配布時の所持カード付与の失敗」のエラーを返したとき、エラーの原因が「配布時の所持カード付与の失敗」のエラーになる",
							failure: errors.New("配布時の所持カード付与の失敗"),
							inject: func(h *grantHarness, failure error) {
								h.playerCards.FailAddingOnce("TST-0001", failure)
							},
						},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							h := newPurchaseGrantHarness()
							tt.inject(h, tt.failure)

							_, err := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0100"))

							assert.ErrorIs(t, err, tt.failure)
						})
					}
				})

				t.Run("プレイヤーAがTST-0001のアート1を2枚所持していて、配布に失敗したとき", func(t *testing.T) {
					tests := []struct {
						name   string
						packID string
					}{
						{name: "存在しないパックTST-9999を購入したとき、プレイヤーAの所持カードが、TST-0001のアート1が2枚だけのままになる", packID: "TST-9999"},
						{name: "運用停止中のパックTST-0101を購入したとき、プレイヤーAの所持カードが、TST-0001のアート1が2枚だけのままになる", packID: "TST-0101"},
						{name: "内包カードが0件のパックTST-0102を購入したとき、プレイヤーAの所持カードが、TST-0001のアート1が2枚だけのままになる", packID: "TST-0102"},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							h := newPurchaseGrantHarness()
							h.playerCards.SetCount(usecasetest.PlayerA, "TST-0001", 1, 2)

							_, grantErr := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent(tt.packID))

							assert.Error(t, grantErr)
							assert.Equal(t, []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 2)}, h.playerCards.FetchOwnedCards(t, usecasetest.PlayerA))
						})
					}
				})

				t.Run("パックTST-0100がTST-0001を3枚、TST-0002を2枚内包し、プレイヤーAの所持カードが無いとき、所持カードのデータベースが付与時に一度だけ「配布時の所持カード付与の失敗」のエラーを返したあと、プレイヤーAがパックTST-0100を購入した同じイベントを再度受け取ると", func(t *testing.T) {
					t.Run("配布した合計枚数が5になる", func(t *testing.T) {
						h := newPurchaseGrantHarness()
						h.playerCards.FailAddingOnce("TST-0001", errors.New("配布時の所持カード付与の失敗"))
						_, firstErr := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0100"))
						require.Error(t, firstErr)

						granted, err := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0100"))

						assert.NoError(t, err)
						assert.Equal(t, 5, granted)
					})

					t.Run("プレイヤーAの所持カードが、TST-0001のアート0が3枚、TST-0002のアート0が2枚になる", func(t *testing.T) {
						h := newPurchaseGrantHarness()
						h.playerCards.FailAddingOnce("TST-0001", errors.New("配布時の所持カード付与の失敗"))
						_, firstErr := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0100"))
						require.Error(t, firstErr)

						_, err := h.interactor.GrantPurchasedPack(t.Context(), newPurchaseEvent("TST-0100"))

						assert.NoError(t, err)
						assert.ElementsMatch(t,
							[]domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 0, 3), usecasetest.NewOwnedCard("TST-0002", 0, 2)},
							h.playerCards.FetchOwnedCards(t, usecasetest.PlayerA),
						)
					})
				})
			})
		})
	})
}

func TestInitialCardPackGrant(t *testing.T) {
	t.Run("[カードパック]初期カードパックの配布", func(t *testing.T) {
		t.Run("プレイヤー登録完了イベントを受け取って、初期カードパックを配布する", func(t *testing.T) {
			t.Run("正常系", func(t *testing.T) {
				t.Run("基本パックがTST-0001を1枚、TST-0002を2枚、faction_set_sheがTST-0003を3枚内包するとき", func(t *testing.T) {
					t.Run("初期陣営がSHEのプレイヤーAの登録完了イベントを受け取ると", func(t *testing.T) {
						t.Run("配布した合計枚数が6になる", func(t *testing.T) {
							h := newInitialGrantHarness()

							granted, err := h.interactor.GrantInitialPacks(t.Context(), newOnboardedEvent("SHE"))

							assert.NoError(t, err)
							assert.Equal(t, 6, granted)
						})

						t.Run("プレイヤーAの所持カードに、基本パックのTST-0001のアート0が1枚、TST-0002のアート0が2枚含まれる", func(t *testing.T) {
							h := newInitialGrantHarness()

							_, err := h.interactor.GrantInitialPacks(t.Context(), newOnboardedEvent("SHE"))

							assert.NoError(t, err)
							owned := h.playerCards.FetchOwnedCards(t, usecasetest.PlayerA)
							assert.Contains(t, owned, usecasetest.NewOwnedCard("TST-0001", 0, 1))
							assert.Contains(t, owned, usecasetest.NewOwnedCard("TST-0002", 0, 2))
						})

						t.Run("プレイヤーAの所持カードに、faction_set_sheのTST-0003のアート0が3枚含まれる", func(t *testing.T) {
							h := newInitialGrantHarness()

							_, err := h.interactor.GrantInitialPacks(t.Context(), newOnboardedEvent("SHE"))

							assert.NoError(t, err)
							assert.Contains(t, h.playerCards.FetchOwnedCards(t, usecasetest.PlayerA), usecasetest.NewOwnedCard("TST-0003", 0, 3))
						})
					})
				})
			})

			t.Run("異常系", func(t *testing.T) {
				t.Run("基本パックがTST-0001を1枚、TST-0002を2枚、faction_set_sheがTST-0003を3枚内包し、プレイヤーAの所持カードが無いとき", func(t *testing.T) {
					t.Run("所持カードのデータベースがfaction_set_sheの付与時に「初期配布時の陣営セット付与の失敗」のエラーを返すとき、初期陣営がSHEのプレイヤーAの登録完了イベントを受け取ると", func(t *testing.T) {
						t.Run("エラーの原因が「初期配布時の陣営セット付与の失敗」のエラーになる", func(t *testing.T) {
							h := newInitialGrantHarness()
							failure := errors.New("初期配布時の陣営セット付与の失敗")
							h.playerCards.FailAddingOnce("TST-0003", failure)

							_, err := h.interactor.GrantInitialPacks(t.Context(), newOnboardedEvent("SHE"))

							assert.ErrorIs(t, err, failure)
						})

						t.Run("プレイヤーAの所持カードが0件のままになる", func(t *testing.T) {
							h := newInitialGrantHarness()
							h.playerCards.FailAddingOnce("TST-0003", errors.New("初期配布時の陣営セット付与の失敗"))

							_, grantErr := h.interactor.GrantInitialPacks(t.Context(), newOnboardedEvent("SHE"))

							assert.Error(t, grantErr)
							assert.Empty(t, h.playerCards.FetchOwnedCards(t, usecasetest.PlayerA))
						})
					})

					t.Run("所持カードのデータベースがfaction_set_sheの付与時に一度だけ「初期配布時の陣営セット付与の失敗」のエラーを返したあと、初期陣営がSHEのプレイヤーAの同じ登録完了イベントを再度受け取ると", func(t *testing.T) {
						t.Run("配布した合計枚数が6になる", func(t *testing.T) {
							h := newInitialGrantHarness()
							h.playerCards.FailAddingOnce("TST-0003", errors.New("初期配布時の陣営セット付与の失敗"))
							_, firstErr := h.interactor.GrantInitialPacks(t.Context(), newOnboardedEvent("SHE"))
							require.Error(t, firstErr)

							granted, err := h.interactor.GrantInitialPacks(t.Context(), newOnboardedEvent("SHE"))

							assert.NoError(t, err)
							assert.Equal(t, 6, granted)
						})

						t.Run("プレイヤーAの所持カードの、基本パックのTST-0001のアート0が1枚、TST-0002のアート0が2枚になる", func(t *testing.T) {
							h := newInitialGrantHarness()
							h.playerCards.FailAddingOnce("TST-0003", errors.New("初期配布時の陣営セット付与の失敗"))
							_, firstErr := h.interactor.GrantInitialPacks(t.Context(), newOnboardedEvent("SHE"))
							require.Error(t, firstErr)

							_, err := h.interactor.GrantInitialPacks(t.Context(), newOnboardedEvent("SHE"))

							assert.NoError(t, err)
							owned := h.playerCards.FetchOwnedCards(t, usecasetest.PlayerA)
							assert.Contains(t, owned, usecasetest.NewOwnedCard("TST-0001", 0, 1))
							assert.Contains(t, owned, usecasetest.NewOwnedCard("TST-0002", 0, 2))
						})

						t.Run("プレイヤーAの所持カードの、faction_set_sheのTST-0003のアート0が3枚になる", func(t *testing.T) {
							h := newInitialGrantHarness()
							h.playerCards.FailAddingOnce("TST-0003", errors.New("初期配布時の陣営セット付与の失敗"))
							_, firstErr := h.interactor.GrantInitialPacks(t.Context(), newOnboardedEvent("SHE"))
							require.Error(t, firstErr)

							_, err := h.interactor.GrantInitialPacks(t.Context(), newOnboardedEvent("SHE"))

							assert.NoError(t, err)
							assert.Contains(t, h.playerCards.FetchOwnedCards(t, usecasetest.PlayerA), usecasetest.NewOwnedCard("TST-0003", 0, 3))
						})
					})
				})
			})
		})
	})
}
