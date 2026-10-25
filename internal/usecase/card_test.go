package usecase_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"
	"github.com/kenyamaneko/overload-party-card/internal/usecase/usecasetest"
)

type cardHarness struct {
	interactor  *usecase.CardInteractor
	cardRepo    *usecasetest.FakeCardRepository
	playerCards *usecasetest.FakePlayerCardRepository
}

func newCardHarness(cards []*domain.Card) *cardHarness {
	cardRepo := &usecasetest.FakeCardRepository{Cards: cards}
	playerCards := usecasetest.NewFakePlayerCardRepository()
	return &cardHarness{
		interactor:  usecase.NewCardInteractor(cardRepo, playerCards),
		cardRepo:    cardRepo,
		playerCards: playerCards,
	}
}

func collectOwnershipByCardID(cards []domain.CardWithOwnership) map[string]bool {
	ownership := map[string]bool{}
	for _, c := range cards {
		ownership[c.Card.CardID] = c.IsOwned
	}
	return ownership
}

func countCardsByCardID(cards []domain.CardWithOwnership, cardID string) int {
	matched := slices.DeleteFunc(slices.Clone(cards), func(c domain.CardWithOwnership) bool { return c.Card.CardID != cardID })
	return len(matched)
}

func TestCardListWithOwnership(t *testing.T) {
	t.Run("[カード]所持状況つきカード一覧", func(t *testing.T) {
		t.Run("プレイヤーの所持状況つきでカード一覧を取得する", func(t *testing.T) {
			t.Run("正常系", func(t *testing.T) {
				t.Run("カードマスターにTST-0001とTST-0002があり、プレイヤーがTST-0001だけを所持しているとき", func(t *testing.T) {
					t.Run("一覧のTST-0001のis_ownedがtrueになる", func(t *testing.T) {
						h := newCardHarness(usecasetest.NewUnlimitedSheCards(2))
						h.playerCards.SetCount(usecasetest.PlayerA, "TST-0001", 1, 1)

						cards, err := h.interactor.ListCardsWithOwnership(t.Context(), usecasetest.PlayerA)

						assert.NoError(t, err)
						ownership := collectOwnershipByCardID(cards)
						assert.Contains(t, ownership, "TST-0001")
						assert.True(t, ownership["TST-0001"])
					})

					t.Run("一覧のTST-0002のis_ownedがfalseになる", func(t *testing.T) {
						h := newCardHarness(usecasetest.NewUnlimitedSheCards(2))
						h.playerCards.SetCount(usecasetest.PlayerA, "TST-0001", 1, 1)

						cards, err := h.interactor.ListCardsWithOwnership(t.Context(), usecasetest.PlayerA)

						assert.NoError(t, err)
						ownership := collectOwnershipByCardID(cards)
						assert.Contains(t, ownership, "TST-0002")
						assert.False(t, ownership["TST-0002"])
					})
				})

				t.Run("カードマスターにTST-0001があり、プレイヤーがTST-0001のアート1とアート2を所持しているとき", func(t *testing.T) {
					t.Run("一覧にTST-0001が1件だけ含まれる", func(t *testing.T) {
						h := newCardHarness(usecasetest.NewUnlimitedSheCards(1))
						h.playerCards.SetCount(usecasetest.PlayerA, "TST-0001", 1, 1)
						h.playerCards.SetCount(usecasetest.PlayerA, "TST-0001", 2, 1)

						cards, err := h.interactor.ListCardsWithOwnership(t.Context(), usecasetest.PlayerA)

						assert.NoError(t, err)
						assert.Equal(t, 1, countCardsByCardID(cards, "TST-0001"))
					})

					t.Run("一覧のTST-0001のis_ownedがtrueになる", func(t *testing.T) {
						h := newCardHarness(usecasetest.NewUnlimitedSheCards(1))
						h.playerCards.SetCount(usecasetest.PlayerA, "TST-0001", 1, 1)
						h.playerCards.SetCount(usecasetest.PlayerA, "TST-0001", 2, 1)

						cards, err := h.interactor.ListCardsWithOwnership(t.Context(), usecasetest.PlayerA)

						assert.NoError(t, err)
						ownership := collectOwnershipByCardID(cards)
						assert.Contains(t, ownership, "TST-0001")
						assert.True(t, ownership["TST-0001"])
					})
				})

				t.Run("カードマスターにTST-0001とTST-0002があり、プレイヤーの所持カードが0件のとき、一覧のTST-0001とTST-0002のis_ownedがどちらもfalseになる", func(t *testing.T) {
					h := newCardHarness(usecasetest.NewUnlimitedSheCards(2))

					cards, err := h.interactor.ListCardsWithOwnership(t.Context(), usecasetest.PlayerA)

					assert.NoError(t, err)
					assert.Equal(t, map[string]bool{"TST-0001": false, "TST-0002": false}, collectOwnershipByCardID(cards))
				})

				t.Run("カードマスターが0件のとき、空の一覧が返る", func(t *testing.T) {
					h := newCardHarness(nil)

					cards, err := h.interactor.ListCardsWithOwnership(t.Context(), usecasetest.PlayerA)

					assert.NoError(t, err)
					assert.NotNil(t, cards)
					assert.Empty(t, cards)
				})
			})

			t.Run("異常系", func(t *testing.T) {
				t.Run("依存先がエラーを返したとき", func(t *testing.T) {
					tests := []struct {
						name    string
						failure error
						inject  func(h *cardHarness, failure error)
					}{
						{
							name:    "カードマスターのデータベースが「所持状況つき一覧時のカードマスター取得の失敗」のエラーを返したとき、エラーの原因が「所持状況つき一覧時のカードマスター取得の失敗」のエラーになる",
							failure: errors.New("所持状況つき一覧時のカードマスター取得の失敗"),
							inject:  func(h *cardHarness, failure error) { h.cardRepo.Err = failure },
						},
						{
							name:    "所持カードのデータベースが「所持状況つき一覧時の所持カード取得の失敗」のエラーを返したとき、エラーの原因が「所持状況つき一覧時の所持カード取得の失敗」のエラーになる",
							failure: errors.New("所持状況つき一覧時の所持カード取得の失敗"),
							inject:  func(h *cardHarness, failure error) { h.playerCards.GetErr = failure },
						},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							h := newCardHarness(usecasetest.NewUnlimitedSheCards(1))
							tt.inject(h, tt.failure)

							_, err := h.interactor.ListCardsWithOwnership(t.Context(), usecasetest.PlayerA)

							assert.ErrorIs(t, err, tt.failure)
						})
					}
				})
			})
		})
	})
}
