package usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/usecase"
	"github.com/kenyamaneko/overload-party-card/internal/usecase/usecasetest"
)

func TestPlayerCardList(t *testing.T) {
	t.Run("[カード]所持カード一覧", func(t *testing.T) {
		t.Run("プレイヤーの所持カードをカード定義つきで取得する", func(t *testing.T) {
			t.Run("正常系", func(t *testing.T) {
				t.Run("所持カードが0件のとき、空の一覧が返る", func(t *testing.T) {
					interactor := usecase.NewPlayerCardInteractor(usecasetest.NewFakePlayerCardRepository(), usecasetest.LoadCardCache(t, usecasetest.NewUnlimitedSheCards(1)))

					cards, err := interactor.GetPlayerCards(t.Context(), usecasetest.PlayerA)

					assert.NoError(t, err)
					assert.NotNil(t, cards)
					assert.Empty(t, cards)
				})
			})

			t.Run("異常系", func(t *testing.T) {
				t.Run("プレイヤーがTST-9999を所持していて、カード定義にTST-9999が無いとき、エラー内容にTST-9999を含み、カード定義が無いことを示すエラーになる", func(t *testing.T) {
					playerCards := usecasetest.NewFakePlayerCardRepository()
					playerCards.SetCount(usecasetest.PlayerA, "TST-9999", 1, 1)
					interactor := usecase.NewPlayerCardInteractor(playerCards, usecasetest.LoadCardCache(t, usecasetest.NewUnlimitedSheCards(1)))

					_, err := interactor.GetPlayerCards(t.Context(), usecasetest.PlayerA)

					assert.Error(t, err)
					assert.ErrorContains(t, err, "TST-9999")
				})

				t.Run("所持カードのデータベースが「所持カード一覧時の所持カード取得の失敗」のエラーを返したとき、エラーの原因が「所持カード一覧時の所持カード取得の失敗」のエラーになる", func(t *testing.T) {
					failure := errors.New("所持カード一覧時の所持カード取得の失敗")
					playerCards := usecasetest.NewFakePlayerCardRepository()
					playerCards.GetErr = failure
					interactor := usecase.NewPlayerCardInteractor(playerCards, usecasetest.LoadCardCache(t, usecasetest.NewUnlimitedSheCards(1)))

					_, err := interactor.GetPlayerCards(t.Context(), usecasetest.PlayerA)

					assert.ErrorIs(t, err, failure)
				})
			})
		})
	})
}
