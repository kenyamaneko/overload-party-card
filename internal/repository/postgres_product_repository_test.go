//go:build integration

package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
	"github.com/kenyamaneko/overload-party-card/internal/repository/repositorytest"
)

func toProductIDs(products []*domain.Product) []string {
	productIDs := make([]string, 0, len(products))
	for _, product := range products {
		productIDs = append(productIDs, product.ProductID)
	}
	return productIDs
}

func TestPostgresProductRepositoryFindAll(t *testing.T) {
	t.Run("[プロダクト]有効なプロダクトの一覧取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("TST-2001は有効フラグがtrue、TST-2002は有効フラグがfalseのプロダクトが保存されているとき、取得すると、TST-2001だけが返る", func(t *testing.T) {
				pool := preparePool(t)
				active := repositorytest.NewProductRow("TST-2001")
				active.IsActive = true
				inactive := repositorytest.NewProductRow("TST-2002")
				inactive.IsActive = false
				repositorytest.InsertProduct(t, pool, active)
				repositorytest.InsertProduct(t, pool, inactive)

				got, err := repository.NewPostgresProductRepository(pool).FindAll(t.Context())

				require.NoError(t, err)
				assert.Equal(t, []string{"TST-2001"}, toProductIDs(got))
			})

			t.Run("プロダクトがTST-2003、TST-2001、TST-2002の順に保存されているとき、取得すると、TST-2001、TST-2002、TST-2003の順に返る", func(t *testing.T) {
				pool := preparePool(t)
				repositorytest.InsertProduct(t, pool, repositorytest.NewProductRow("TST-2003"))
				repositorytest.InsertProduct(t, pool, repositorytest.NewProductRow("TST-2001"))
				repositorytest.InsertProduct(t, pool, repositorytest.NewProductRow("TST-2002"))

				got, err := repository.NewPostgresProductRepository(pool).FindAll(t.Context())

				require.NoError(t, err)
				assert.Equal(t, []string{"TST-2001", "TST-2002", "TST-2003"}, toProductIDs(got))
			})

			t.Run("プロダクトが1件も保存されていないとき、取得すると、空の一覧が返る", func(t *testing.T) {
				pool := preparePool(t)

				got, err := repository.NewPostgresProductRepository(pool).FindAll(t.Context())

				require.NoError(t, err)
				assert.Len(t, got, 0)
			})

			t.Run("プロダクトIDがTST-2001、陣営がTenki、プロダクト名がテストプロダクト、有効フラグがtrueの有効なプロダクトが保存されているとき、取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got *domain.Product)
				}{
					{"プロダクトIDがTST-2001になる", func(t *testing.T, got *domain.Product) { assert.Equal(t, "TST-2001", got.ProductID) }},
					{"陣営がTenkiになる", func(t *testing.T, got *domain.Product) { assert.Equal(t, "Tenki", got.Faction) }},
					{"プロダクト名がテストプロダクトになる", func(t *testing.T, got *domain.Product) { assert.Equal(t, "テストプロダクト", got.ProductName) }},
					{"有効フラグがtrueになる", func(t *testing.T, got *domain.Product) { assert.True(t, got.IsActive) }},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						pool := preparePool(t)
						row := repositorytest.NewProductRow("TST-2001")
						row.Faction = "Tenki"
						row.ProductName = "テストプロダクト"
						row.IsActive = true
						repositorytest.InsertProduct(t, pool, row)

						got, err := repository.NewPostgresProductRepository(pool).FindAll(t.Context())

						require.NoError(t, err)
						require.Len(t, got, 1)
						tc.check(t, got[0])
					})
				}
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("DBに接続できないとき、取得すると、DBに接続できないことを示すエラーが返る", func(t *testing.T) {
				pool := repositorytest.NewUnreachablePool(t)

				_, err := repository.NewPostgresProductRepository(pool).FindAll(t.Context())

				repositorytest.RequireConnectionError(t, err)
			})
		})
	})
}
