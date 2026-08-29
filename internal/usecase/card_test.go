package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	gamedesign "github.com/kenyamaneko/overload-party-common/packages/game-design-constants"
)

// newCardFixture は fakeCardRepo / fakePlayerCardRepo を差し替え可能な CardInteractor を返す。
func newCardFixture() (*CardInteractor, *fakeCardRepo, *fakePlayerCardRepo) {
	cardRepo := &fakeCardRepo{}
	playerCardRepo := newFakePlayerCardRepo()
	return NewCardInteractor(cardRepo, playerCardRepo), cardRepo, playerCardRepo
}

func TestListCardsWithOwnership(t *testing.T) {
	t.Run("[カード]カード一覧への所持状態付与", func(t *testing.T) {
		t.Run("カード定義の取得元がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, cardRepo, _ := newCardFixture()
			injected := errors.New("card repo unavailable")
			cardRepo.findAllErr = injected

			_, err := interactor.ListCardsWithOwnership(context.Background(), fxPlayerID)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("カード定義の取得が成功し、プレイヤー所持カードの取得元がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, cardRepo, playerCardRepo := newCardFixture()
			cardRepo.cards = []*domain.Card{{CardID: "TST-0001", Faction: fxFaction, Restriction: gamedesign.RestrictionUnlimited, Stats: json.RawMessage(`{}`)}}
			injected := errors.New("player card repo unavailable")
			playerCardRepo.getErr = injected

			_, err := interactor.ListCardsWithOwnership(context.Background(), fxPlayerID)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("カード定義が0件のとき、空の一覧を返す", func(t *testing.T) {
			interactor, _, _ := newCardFixture()

			result, err := interactor.ListCardsWithOwnership(context.Background(), fxPlayerID)

			require.NoError(t, err)
			assert.Empty(t, result)
		})

		t.Run("カードTST-0001をプレイヤーがどのアートNo変体も所持していないとき、返る一覧のTST-0001はis_ownedがfalseになる", func(t *testing.T) {
			interactor, cardRepo, _ := newCardFixture()
			cardRepo.cards = []*domain.Card{{CardID: "TST-0001", Faction: fxFaction, Restriction: gamedesign.RestrictionUnlimited, Stats: json.RawMessage(`{}`)}}

			result, err := interactor.ListCardsWithOwnership(context.Background(), fxPlayerID)

			require.NoError(t, err)
			require.Len(t, result, 1)
			assert.False(t, result[0].IsOwned)
		})

		t.Run("カードTST-0001のいずれかのアートNo変体をプレイヤーが1件以上所持しているとき、返る一覧のTST-0001はis_ownedがtrueになる", func(t *testing.T) {
			interactor, cardRepo, playerCardRepo := newCardFixture()
			cardRepo.cards = []*domain.Card{{CardID: "TST-0001", Faction: fxFaction, Restriction: gamedesign.RestrictionUnlimited, Stats: json.RawMessage(`{}`)}}
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 2, Count: 1})

			result, err := interactor.ListCardsWithOwnership(context.Background(), fxPlayerID)

			require.NoError(t, err)
			require.Len(t, result, 1)
			assert.True(t, result[0].IsOwned)
		})

		t.Run("返る一覧の各カードのcard_name/faction/card_type/restriction等の内容は、カード定義取得元が返した内容と一致する", func(t *testing.T) {
			interactor, cardRepo, _ := newCardFixture()
			subtype := "VM"
			effectText := "draw a card"
			cardRepo.cards = []*domain.Card{{
				CardID:        "TST-0001",
				CardName:      "Test Compute",
				ResourceLabel: "vCPU",
				Faction:       fxFaction,
				CardType:      gamedesign.CardTypeCompute,
				Subtype:       &subtype,
				Resizable:     true,
				Elastic:       true,
				Stats:         json.RawMessage(`{"tp":1}`),
				EffectText:    &effectText,
				Restriction:   gamedesign.RestrictionUnlimited,
				IsActive:      true,
			}}

			result, err := interactor.ListCardsWithOwnership(context.Background(), fxPlayerID)

			require.NoError(t, err)
			require.Len(t, result, 1)
			got := result[0]
			assert.Equal(t, "TST-0001", got.CardID)
			assert.Equal(t, "Test Compute", got.CardName)
			assert.Equal(t, "vCPU", got.ResourceLabel)
			assert.Equal(t, fxFaction, got.Faction)
			assert.Equal(t, gamedesign.CardTypeCompute, got.CardType)
			assert.Equal(t, &subtype, got.Subtype)
			assert.True(t, got.Resizable)
			assert.True(t, got.Elastic)
			assert.Equal(t, json.RawMessage(`{"tp":1}`), got.Stats)
			assert.Equal(t, &effectText, got.EffectText)
			assert.Equal(t, gamedesign.RestrictionUnlimited, got.Restriction)
			assert.True(t, got.IsActive)
		})
	})
}

func TestListCards(t *testing.T) {
	t.Run("[カード]カード定義一覧取得", func(t *testing.T) {
		t.Run("カード定義の取得元がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, cardRepo, _ := newCardFixture()
			injected := errors.New("card repo unavailable")
			cardRepo.findAllErr = injected

			_, err := interactor.ListCards(context.Background())

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("カード定義が0件のとき、空の一覧を返す", func(t *testing.T) {
			interactor, _, _ := newCardFixture()

			result, err := interactor.ListCards(context.Background())

			require.NoError(t, err)
			assert.Empty(t, result)
		})

		t.Run("返る一覧の内容は、カード定義取得元が返した内容と一致する", func(t *testing.T) {
			interactor, cardRepo, _ := newCardFixture()
			cardRepo.cards = []*domain.Card{{
				CardID:        "TST-0001",
				CardName:      "Test Compute",
				ResourceLabel: "vCPU",
				Faction:       fxFaction,
				CardType:      gamedesign.CardTypeCompute,
				Restriction:   gamedesign.RestrictionUnlimited,
				IsActive:      true,
				Stats:         json.RawMessage(`{}`),
			}}

			result, err := interactor.ListCards(context.Background())

			require.NoError(t, err)
			require.Len(t, result, 1)
			want := &apicard.CardDefinition{
				CardID:        "TST-0001",
				CardName:      "Test Compute",
				ResourceLabel: "vCPU",
				Faction:       fxFaction,
				CardType:      gamedesign.CardTypeCompute,
				Restriction:   gamedesign.RestrictionUnlimited,
				IsActive:      true,
				Stats:         json.RawMessage(`{}`),
			}
			assert.Equal(t, want, result[0])
		})
	})
}
