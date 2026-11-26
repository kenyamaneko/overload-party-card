package router_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/router/routertest"
)

func TestProductList(t *testing.T) {
	t.Run("[プロダクト]プロダクト一覧の取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("プロダクトTST-0001に施策TST-0011とTST-0012が属し、プロダクトTST-0002に施策TST-0013が属するとき", func(t *testing.T) {
				requestProducts := func(t *testing.T) []any {
					masterData := routertest.NewMasterData()
					masterData.Products = []*domain.Product{routertest.NewTestProduct("TST-0001", "SHE"), routertest.NewTestProduct("TST-0002", "Tenki")}
					masterData.Initiatives = []*domain.Initiative{
						routertest.NewTestInitiative("TST-0011", "TST-0001", "routine"),
						routertest.NewTestInitiative("TST-0012", "TST-0001", "special"),
						routertest.NewTestInitiative("TST-0013", "TST-0002", "routine"),
					}
					return routertest.DecodeArray(t, routertest.NewHarnessWithMasterData(t, masterData).SendGetRequest(routertest.ProductsPath))
				}

				t.Run("応答のプロダクトTST-0001のinitiativesの施策IDはTST-0011とTST-0012だけになる", func(t *testing.T) {
					products := requestProducts(t)

					product := routertest.FindObject(t, products, "product_id", "TST-0001")

					assert.ElementsMatch(t, []any{"TST-0011", "TST-0012"}, routertest.CollectValues(t, routertest.ToArray(t, product["initiatives"]), "initiative_id"))
				})
				t.Run("応答のプロダクトTST-0002のinitiativesの施策IDはTST-0013だけになる", func(t *testing.T) {
					products := requestProducts(t)

					product := routertest.FindObject(t, products, "product_id", "TST-0002")

					assert.ElementsMatch(t, []any{"TST-0013"}, routertest.CollectValues(t, routertest.ToArray(t, product["initiatives"]), "initiative_id"))
				})
			})
			t.Run("プロダクトTST-0001の施策がTST-0012、TST-0011の順に読み込まれているとき、応答のプロダクトTST-0001のinitiativesは施策IDの昇順に並ぶ", func(t *testing.T) {
				masterData := routertest.NewMasterData()
				masterData.Initiatives = []*domain.Initiative{
					routertest.NewTestInitiative("TST-0012", "TST-0001", "special"),
					routertest.NewTestInitiative("TST-0011", "TST-0001", "routine"),
				}
				h := routertest.NewHarnessWithMasterData(t, masterData)

				products := routertest.DecodeArray(t, h.SendGetRequest(routertest.ProductsPath))

				product := routertest.FindObject(t, products, "product_id", "TST-0001")
				assert.Equal(t, []any{"TST-0011", "TST-0012"}, routertest.CollectValues(t, routertest.ToArray(t, product["initiatives"]), "initiative_id"))
			})
		})
		t.Run("異常系", func(t *testing.T) {
			t.Run("プロダクトTST-0001が施策を1件も持たず、プロダクトTST-0002に施策TST-0013が属するとき", func(t *testing.T) {
				newScenario := func(t *testing.T) *routertest.Harness {
					masterData := routertest.NewMasterData()
					masterData.Products = []*domain.Product{routertest.NewTestProduct("TST-0001", "SHE"), routertest.NewTestProduct("TST-0002", "Tenki")}
					masterData.Initiatives = []*domain.Initiative{routertest.NewTestInitiative("TST-0013", "TST-0002", "routine")}
					return routertest.NewHarnessWithMasterData(t, masterData)
				}

				t.Run("内部エラーを示す500の応答になる", func(t *testing.T) {
					h := newScenario(t)

					recorder := h.SendGetRequest(routertest.ProductsPath)

					assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				})
				t.Run("Errorレベルのログのerror項目はTST-0001を含む", func(t *testing.T) {
					h := newScenario(t)

					h.SendGetRequest(routertest.ProductsPath)

					h.RequireErrorLogContaining(t, "error", "TST-0001")
				})
			})
		})
	})
}
