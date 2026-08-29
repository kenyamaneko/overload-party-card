//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

func TestCardRepoFindAll(t *testing.T) {
	t.Run("[カードリポジトリ] カード定義一覧取得", func(t *testing.T) {
		t.Run("有効なカード定義が0件のとき、空の一覧を返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgCardRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Empty(t, got)
		})

		t.Run("is_active=trueのカード定義があるとき、それらを返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCard(t, cardSeed{CardID: "TST-0001", CardName: "テストカードA", Faction: "SHE", CardType: "Resource", Restriction: "unlimited", IsActive: true})
			repo := repository.NewPgCardRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.Equal(t, "TST-0001", got[0].CardID)
		})

		t.Run("is_active=falseのカード定義は、返る一覧に含まれない", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCard(t, cardSeed{CardID: "TST-0001", CardName: "テストカードA", Faction: "SHE", CardType: "Resource", Restriction: "unlimited", IsActive: false})
			repo := repository.NewPgCardRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Empty(t, got)
		})

		t.Run("is_active=trueのカード定義が複数件あるとき、card_idの昇順で返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCards(t, []cardSeed{
				{CardID: "TST-0003", CardName: "テストカードC", Faction: "SHE", CardType: "Resource", Restriction: "unlimited", IsActive: true},
				{CardID: "TST-0001", CardName: "テストカードA", Faction: "SHE", CardType: "Resource", Restriction: "unlimited", IsActive: true},
				{CardID: "TST-0002", CardName: "テストカードB", Faction: "SHE", CardType: "Resource", Restriction: "unlimited", IsActive: true},
			})
			repo := repository.NewPgCardRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 3)
			require.Equal(t, []string{"TST-0001", "TST-0002", "TST-0003"},
				[]string{got[0].CardID, got[1].CardID, got[2].CardID})
		})

		t.Run("保存したcard_name/resource_label/faction/card_type/resizable/elastic/restriction/is_active/created_at/updated_atの値が、そのまま返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			createdAt := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
			updatedAt := time.Date(2026, 1, 15, 12, 30, 0, 0, time.UTC)
			seedFullCard(t, fullCardSeed{
				CardID: "TST-0001", CardName: "テストカードA", ResourceLabel: "CPU",
				Faction: "SHE", CardType: "Resource", Subtype: "VM",
				Resizable: true, Elastic: false, Stats: `{"cpu":2}`,
				EffectText: "テスト効果", Effects: `[{"op":"add"}]`,
				Restriction: "limited", IsActive: true,
				CreatedAt: createdAt, UpdatedAt: updatedAt,
			})
			repo := repository.NewPgCardRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 1)
			c := got[0]
			require.Equal(t, "テストカードA", c.CardName)
			require.Equal(t, "CPU", c.ResourceLabel)
			require.Equal(t, "SHE", c.Faction)
			require.Equal(t, "Resource", c.CardType)
			require.True(t, c.Resizable)
			require.False(t, c.Elastic)
			require.Equal(t, "limited", c.Restriction)
			require.True(t, c.IsActive)
			require.True(t, createdAt.Equal(c.CreatedAt))
			require.True(t, updatedAt.Equal(c.UpdatedAt))
		})

		t.Run("保存したstats(JSON)の内容が、そのまま返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedFullCard(t, fullCardSeed{
				CardID: "TST-0001", CardName: "テストカードA", ResourceLabel: "CPU",
				Faction: "SHE", CardType: "Resource", Subtype: "VM",
				Resizable: true, Elastic: false, Stats: `{"cpu":2,"mem":4}`,
				EffectText: "テスト効果", Effects: `[]`,
				Restriction: "unlimited", IsActive: true,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			})
			repo := repository.NewPgCardRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.JSONEq(t, `{"cpu":2,"mem":4}`, string(got[0].Stats))
		})

		t.Run("subtypeがNULLのカード定義を取得すると、返り値のsubtypeも未設定になる", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCard(t, cardSeed{CardID: "TST-0001", CardName: "テストカードA", Faction: "SHE", CardType: "Resource", Restriction: "unlimited", IsActive: true})
			repo := repository.NewPgCardRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.Nil(t, got[0].Subtype)
		})

		t.Run("subtypeが設定されたカード定義を取得すると、その値が返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedFullCard(t, fullCardSeed{
				CardID: "TST-0001", CardName: "テストカードA", ResourceLabel: "CPU",
				Faction: "SHE", CardType: "Resource", Subtype: "Container",
				Resizable: false, Elastic: false, Stats: `{}`,
				EffectText: "e", Effects: `[]`,
				Restriction: "unlimited", IsActive: true,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			})
			repo := repository.NewPgCardRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.NotNil(t, got[0].Subtype)
			require.Equal(t, "Container", *got[0].Subtype)
		})

		t.Run("effect_textがNULLのカード定義を取得すると、返り値のeffect_textも未設定になる", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCard(t, cardSeed{CardID: "TST-0001", CardName: "テストカードA", Faction: "SHE", CardType: "Resource", Restriction: "unlimited", IsActive: true})
			repo := repository.NewPgCardRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.Nil(t, got[0].EffectText)
		})

		t.Run("effect_textが設定されたカード定義を取得すると、その値が返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedFullCard(t, fullCardSeed{
				CardID: "TST-0001", CardName: "テストカードA", ResourceLabel: "CPU",
				Faction: "SHE", CardType: "Resource", Subtype: "VM",
				Resizable: false, Elastic: false, Stats: `{}`,
				EffectText: "効果テキストです", Effects: `[]`,
				Restriction: "unlimited", IsActive: true,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			})
			repo := repository.NewPgCardRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.NotNil(t, got[0].EffectText)
			require.Equal(t, "効果テキストです", *got[0].EffectText)
		})

		t.Run("effectsがNULLのカード定義を取得すると、返り値のeffectsも未設定になる", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCard(t, cardSeed{CardID: "TST-0001", CardName: "テストカードA", Faction: "SHE", CardType: "Resource", Restriction: "unlimited", IsActive: true})
			repo := repository.NewPgCardRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.Nil(t, got[0].Effects)
		})

		t.Run("effects(JSON)が設定されたカード定義を取得すると、その内容がそのまま返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedFullCard(t, fullCardSeed{
				CardID: "TST-0001", CardName: "テストカードA", ResourceLabel: "CPU",
				Faction: "SHE", CardType: "Resource", Subtype: "VM",
				Resizable: false, Elastic: false, Stats: `{}`,
				EffectText: "e", Effects: `[{"op":"draw","value":1}]`,
				Restriction: "unlimited", IsActive: true,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			})
			repo := repository.NewPgCardRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.JSONEq(t, `[{"op":"draw","value":1}]`, string(got[0].Effects))
		})
	})
}
