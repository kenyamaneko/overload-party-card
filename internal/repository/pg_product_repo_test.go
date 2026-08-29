//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

func TestProductRepoFindAll(t *testing.T) {
	t.Run("[プロダクトリポジトリ] プロダクト定義一覧取得", func(t *testing.T) {
		t.Run("有効なプロダクト定義が0件のとき、空の一覧を返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgProductRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Empty(t, got)
		})

		t.Run("is_active=trueのプロダクト定義があるとき、それらを返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedProduct(t, productSeed{ProductID: "PD-TST01", Faction: "SHE", ProductName: "テストプロダクトA"})
			repo := repository.NewPgProductRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.Equal(t, "PD-TST01", got[0].ProductID)
		})

		t.Run("is_active=falseのプロダクト定義は、返る一覧に含まれない", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedInactiveProduct(t, productSeed{ProductID: "PD-TST01", Faction: "SHE", ProductName: "テストプロダクトA"})
			repo := repository.NewPgProductRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Empty(t, got)
		})

		t.Run("is_active=trueのプロダクト定義が複数件あるとき、product_idの昇順で返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedProduct(t, productSeed{ProductID: "PD-TST03", Faction: "SHE", ProductName: "テストプロダクトC"})
			seedProduct(t, productSeed{ProductID: "PD-TST01", Faction: "SHE", ProductName: "テストプロダクトA"})
			seedProduct(t, productSeed{ProductID: "PD-TST02", Faction: "SHE", ProductName: "テストプロダクトB"})
			repo := repository.NewPgProductRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 3)
			require.Equal(t, []string{"PD-TST01", "PD-TST02", "PD-TST03"},
				[]string{got[0].ProductID, got[1].ProductID, got[2].ProductID})
		})

		t.Run("保存したfaction/product_name/is_activeの値が、そのまま返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedProduct(t, productSeed{ProductID: "PD-TST01", Faction: "Tenki", ProductName: "テストプロダクトA"})
			repo := repository.NewPgProductRepository(sharedPg.Pool)

			got, err := repo.FindAll(context.Background())

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.Equal(t, "Tenki", got[0].Faction)
			require.Equal(t, "テストプロダクトA", got[0].ProductName)
			require.True(t, got[0].IsActive)
		})
	})
}
