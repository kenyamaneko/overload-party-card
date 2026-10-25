package usecase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"
	"github.com/kenyamaneko/overload-party-card/internal/usecase/usecasetest"
)

func collectInitiativeIDsByProductID(products []domain.ProductWithInitiatives) map[string][]string {
	grouped := map[string][]string{}
	for _, p := range products {
		ids := []string{}
		for _, i := range p.Initiatives {
			ids = append(ids, i.InitiativeID)
		}
		grouped[p.Product.ProductID] = ids
	}
	return grouped
}

func TestProductList(t *testing.T) {
	t.Run("[プロダクト]プロダクト一覧", func(t *testing.T) {
		t.Run("プロダクトの一覧を、プロダクトごとの施策つきで取得する", func(t *testing.T) {
			t.Run("正常系", func(t *testing.T) {
				t.Run("プロダクトTST-0001に施策TST-1001とTST-1002が属し、プロダクトTST-0002に施策TST-1003が属するとき", func(t *testing.T) {
					products := []*domain.Product{
						usecasetest.NewTestProduct("TST-0001", usecasetest.FactionSHE),
						usecasetest.NewTestProduct("TST-0002", usecasetest.FactionTenki),
					}
					initiatives := []*domain.Initiative{
						usecasetest.NewTestInitiative("TST-1001", "TST-0001", "routine"),
						usecasetest.NewTestInitiative("TST-1002", "TST-0001", "special"),
						usecasetest.NewTestInitiative("TST-1003", "TST-0002", "routine"),
					}

					t.Run("一覧のプロダクトTST-0001の施策が、TST-1001とTST-1002だけになる", func(t *testing.T) {
						interactor := usecase.NewProductInteractor(usecasetest.LoadProductCache(t, products), usecasetest.LoadInitiativeCache(t, initiatives))

						listed, err := interactor.ListProducts()

						assert.NoError(t, err)
						assert.ElementsMatch(t, []string{"TST-1001", "TST-1002"}, collectInitiativeIDsByProductID(listed)["TST-0001"])
					})

					t.Run("一覧のプロダクトTST-0002の施策が、TST-1003だけになる", func(t *testing.T) {
						interactor := usecase.NewProductInteractor(usecasetest.LoadProductCache(t, products), usecasetest.LoadInitiativeCache(t, initiatives))

						listed, err := interactor.ListProducts()

						assert.NoError(t, err)
						assert.ElementsMatch(t, []string{"TST-1003"}, collectInitiativeIDsByProductID(listed)["TST-0002"])
					})
				})
			})

			t.Run("異常系", func(t *testing.T) {
				t.Run("プロダクトTST-0001が施策を1件も持たず、プロダクトTST-0002に施策TST-1003が属するとき、エラー内容にTST-0001を含み、プロダクトに施策が無いことを示すエラーになる", func(t *testing.T) {
					products := []*domain.Product{
						usecasetest.NewTestProduct("TST-0001", usecasetest.FactionSHE),
						usecasetest.NewTestProduct("TST-0002", usecasetest.FactionTenki),
					}
					initiatives := []*domain.Initiative{usecasetest.NewTestInitiative("TST-1003", "TST-0002", "routine")}
					interactor := usecase.NewProductInteractor(usecasetest.LoadProductCache(t, products), usecasetest.LoadInitiativeCache(t, initiatives))

					_, err := interactor.ListProducts()

					assert.Error(t, err)
					assert.ErrorContains(t, err, "TST-0001")
				})
			})
		})
	})
}
