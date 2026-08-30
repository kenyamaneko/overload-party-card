//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

func TestInitiativeRepoFindAll(t *testing.T) {
	t.Run("[施策リポジトリ] 施策定義一覧取得", func(t *testing.T) {
		t.Run("有効な施策定義が0件のとき、空の一覧を返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgInitiativeRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Empty(t, got)
		})

		t.Run("有効な施策定義があるとき、それらを返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedProduct(t, productSeed{ProductID: "PD-TST01", Faction: "SHE", ProductName: "テストプロダクトA"})
			seedInitiative(t, initiativeSeed{InitiativeID: "IN-TST01", ProductID: "PD-TST01", Kind: "routine", Name: "テスト施策A", InsightCost: 1, EffectText: "効果A", Effect: `{}`})
			repo := repository.NewPgInitiativeRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.Equal(t, "IN-TST01", got[0].InitiativeID)
		})

		t.Run("無効な施策定義は、返る一覧に含まれない", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedProduct(t, productSeed{ProductID: "PD-TST01", Faction: "SHE", ProductName: "テストプロダクトA"})
			seedInactiveInitiative(t, initiativeSeed{InitiativeID: "IN-TST01", ProductID: "PD-TST01", Kind: "routine", Name: "テスト施策A", InsightCost: 1, EffectText: "効果A", Effect: `{}`})
			repo := repository.NewPgInitiativeRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Empty(t, got)
		})

		t.Run("有効な施策定義が複数件あるとき、initiative_idの昇順で返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedProduct(t, productSeed{ProductID: "PD-TST01", Faction: "SHE", ProductName: "テストプロダクトA"})
			seedInitiative(t, initiativeSeed{InitiativeID: "IN-TST03", ProductID: "PD-TST01", Kind: "routine", Name: "テスト施策C", InsightCost: 1, EffectText: "効果C", Effect: `{}`})
			seedInitiative(t, initiativeSeed{InitiativeID: "IN-TST01", ProductID: "PD-TST01", Kind: "routine", Name: "テスト施策A", InsightCost: 1, EffectText: "効果A", Effect: `{}`})
			seedInitiative(t, initiativeSeed{InitiativeID: "IN-TST02", ProductID: "PD-TST01", Kind: "routine", Name: "テスト施策B", InsightCost: 1, EffectText: "効果B", Effect: `{}`})
			repo := repository.NewPgInitiativeRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 3)
			require.Equal(t, []string{"IN-TST01", "IN-TST02", "IN-TST03"},
				[]string{got[0].InitiativeID, got[1].InitiativeID, got[2].InitiativeID})
		})

		t.Run("保存したproduct_id/kind/name/insight_cost/effect_text/is_activeの値が、そのまま返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedProduct(t, productSeed{ProductID: "PD-TST01", Faction: "SHE", ProductName: "テストプロダクトA"})
			seedInitiative(t, initiativeSeed{InitiativeID: "IN-TST01", ProductID: "PD-TST01", Kind: "special", Name: "テスト施策A", InsightCost: 5, EffectText: "効果テキストA", Effect: `{}`})
			repo := repository.NewPgInitiativeRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 1)
			i := got[0]
			require.Equal(t, "PD-TST01", i.ProductID)
			require.Equal(t, "special", i.Kind)
			require.Equal(t, "テスト施策A", i.Name)
			require.Equal(t, int64(5), i.InsightCost)
			require.Equal(t, "効果テキストA", i.EffectText)
			require.True(t, i.IsActive)
		})

		t.Run("保存したeffect(JSON)の内容が、そのまま返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedProduct(t, productSeed{ProductID: "PD-TST01", Faction: "SHE", ProductName: "テストプロダクトA"})
			seedInitiative(t, initiativeSeed{InitiativeID: "IN-TST01", ProductID: "PD-TST01", Kind: "routine", Name: "テスト施策A", InsightCost: 1, EffectText: "効果A", Effect: `{"draw":2}`})
			repo := repository.NewPgInitiativeRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.JSONEq(t, `{"draw":2}`, string(got[0].Effect))
		})
	})
}
