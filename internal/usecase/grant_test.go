package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// newGrantFixture は fakeCardPackRepo / fakePlayerCardRepo を差し替え可能な
// GrantInteractor を返す。
func newGrantFixture() (*GrantInteractor, *fakeCardPackRepo, *fakePlayerCardRepo) {
	cardPackRepo := &fakeCardPackRepo{}
	playerCardRepo := newFakePlayerCardRepo()
	return NewGrantInteractor(cardPackRepo, playerCardRepo), cardPackRepo, playerCardRepo
}

func TestGrantPack(t *testing.T) {
	t.Run("[パック配布]カードパック配布", func(t *testing.T) {
		t.Run("指定したパックIDがパックマスターに存在しないとき、パック不在を表すエラーを返す", func(t *testing.T) {
			interactor, cardPackRepo, _ := newGrantFixture()
			cardPackRepo.getErr = port.ErrNotFound

			_, err := interactor.GrantPack(context.Background(), fxPlayerID, "PK-0001")

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrNotFound)
		})

		t.Run("パックマスターの取得元がその他のエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, cardPackRepo, _ := newGrantFixture()
			injected := errors.New("card pack repo unavailable")
			cardPackRepo.getErr = injected

			_, err := interactor.GrantPack(context.Background(), fxPlayerID, "PK-0001")

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("取得したパックが運用停止中(is_activeがfalse)のとき、運用停止を表すエラーを返す", func(t *testing.T) {
			interactor, cardPackRepo, _ := newGrantFixture()
			cardPackRepo.pack = &domain.CardPack{
				PackID:   "PK-0001",
				IsActive: false,
				Cards:    []domain.CardPackCard{{CardID: "TST-0001", Copies: 1}},
			}

			_, err := interactor.GrantPack(context.Background(), fxPlayerID, "PK-0001")

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrPackInactive)
		})

		t.Run("取得したパックが運用中で、内包カードが0件のとき、内包カード無しを表すエラーを返す", func(t *testing.T) {
			interactor, cardPackRepo, _ := newGrantFixture()
			cardPackRepo.pack = &domain.CardPack{PackID: "PK-0001", IsActive: true, Cards: nil}

			_, err := interactor.GrantPack(context.Background(), fxPlayerID, "PK-0001")

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrEmptyPack)
		})

		t.Run("取得したパックが運用中で内包カードが1件以上あり、所持カード追加処理がエラーを返すとき、そのエラーをそのまま返す", func(t *testing.T) {
			interactor, cardPackRepo, playerCardRepo := newGrantFixture()
			cardPackRepo.pack = &domain.CardPack{
				PackID:   "PK-0001",
				IsActive: true,
				Cards:    []domain.CardPackCard{{CardID: "TST-0001", Copies: 1}},
			}
			injected := errors.New("player card repo unavailable")
			playerCardRepo.addErr = injected

			_, err := interactor.GrantPack(context.Background(), fxPlayerID, "PK-0001")

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("取得したパックが運用中で内包カードが1件以上あるとき、そのパックの内包カード構成がそのまま所持カード追加処理に渡される", func(t *testing.T) {
			interactor, cardPackRepo, playerCardRepo := newGrantFixture()
			packCards := []domain.CardPackCard{{CardID: "TST-0001", Copies: 2}, {CardID: "TST-0002", Copies: 3}}
			cardPackRepo.pack = &domain.CardPack{PackID: "PK-0001", IsActive: true, Cards: packCards}

			_, err := interactor.GrantPack(context.Background(), fxPlayerID, "PK-0001")

			require.NoError(t, err)
			assert.Equal(t, fxPlayerID, playerCardRepo.addedPlayerID)
			assert.Equal(t, packCards, playerCardRepo.addedCards)
		})

		t.Run("所持カード追加処理が成功したとき、その処理が返した加算コピー総数がそのまま返る", func(t *testing.T) {
			interactor, cardPackRepo, playerCardRepo := newGrantFixture()
			cardPackRepo.pack = &domain.CardPack{
				PackID:   "PK-0001",
				IsActive: true,
				Cards:    []domain.CardPackCard{{CardID: "TST-0001", Copies: 2}, {CardID: "TST-0002", Copies: 3}},
			}
			playerCardRepo.addResult = 99

			got, err := interactor.GrantPack(context.Background(), fxPlayerID, "PK-0001")

			require.NoError(t, err)
			assert.Equal(t, 99, got)
		})
	})
}
