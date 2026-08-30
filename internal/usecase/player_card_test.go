package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	gamedesign "github.com/kenyamaneko/overload-party-common/packages/game-design-constants"
)

// newPlayerCardFixture は fakePlayerCardRepo / cache.CardCache を差し替え可能な
// PlayerCardInteractor を返す。
func newPlayerCardFixture() (*PlayerCardInteractor, *fakePlayerCardRepo, *cache.CardCache) {
	playerCardRepo := newFakePlayerCardRepo()
	cc := cache.NewCardCache()
	return NewPlayerCardInteractor(playerCardRepo, cc), playerCardRepo, cc
}

func TestGetPlayerCards(t *testing.T) {
	t.Run("[所持カード]所持カード一覧取得", func(t *testing.T) {
		t.Run("プレイヤー所持カードの取得元がエラーを返すとき、そのエラーをそのまま返す", func(t *testing.T) {
			interactor, playerCardRepo, _ := newPlayerCardFixture()
			injected := errors.New("player card repo unavailable")
			playerCardRepo.getErr = injected

			_, err := interactor.GetPlayerCards(context.Background(), fxPlayerID)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("プレイヤーが所持カードを1件も持たないとき、空の一覧を返す", func(t *testing.T) {
			interactor, _, _ := newPlayerCardFixture()

			result, err := interactor.GetPlayerCards(context.Background(), fxPlayerID)

			require.NoError(t, err)
			assert.Empty(t, result)
		})

		t.Run("プレイヤーが所持するカードのカードIDがカード定義キャッシュに存在しないとき、キャッシュ不整合を示すエラーを返す", func(t *testing.T) {
			interactor, playerCardRepo, _ := newPlayerCardFixture()
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-UNDEFINED", ArtNo: 1, Count: 1})

			_, err := interactor.GetPlayerCards(context.Background(), fxPlayerID)

			require.Error(t, err)
			assert.ErrorContains(t, err, "cache")
		})

		t.Run("プレイヤーが所持するカードがすべてカード定義キャッシュに存在するとき、返る一覧の各要素は所持カードの内容とカード定義キャッシュ上の対応するカード定義の内容を組み合わせたものになる", func(t *testing.T) {
			interactor, playerCardRepo, cc := newPlayerCardFixture()
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 3, Count: 2})
			cc.InjectForTest("TST-0001", &domain.Card{
				CardID:        "TST-0001",
				CardName:      "Test Compute",
				ResourceLabel: "vCPU",
				Faction:       fxFaction,
				CardType:      gamedesign.CardTypeCompute,
				Resizable:     true,
				Elastic:       true,
				Stats:         json.RawMessage(`{"tp":1}`),
				Restriction:   gamedesign.RestrictionUnlimited,
			})

			result, err := interactor.GetPlayerCards(context.Background(), fxPlayerID)

			require.NoError(t, err)
			require.Len(t, result, 1)
			want := &apicard.PlayerCardWithDef{
				CardID:        "TST-0001",
				ArtNo:         3,
				Count:         2,
				CardName:      "Test Compute",
				ResourceLabel: "vCPU",
				Faction:       fxFaction,
				CardType:      gamedesign.CardTypeCompute,
				Resizable:     true,
				Elastic:       true,
				Stats:         json.RawMessage(`{"tp":1}`),
				Restriction:   gamedesign.RestrictionUnlimited,
			}
			assert.Equal(t, want, result[0])
		})
	})
}
