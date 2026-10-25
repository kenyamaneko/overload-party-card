package cache_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/cache/cachetest"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

func loadProductCache(t *testing.T, productIDs ...string) *cache.ProductCache {
	t.Helper()
	productCache := cache.NewProductCache()
	require.NoError(t, productCache.Load(context.Background(), cachetest.NewFakeProductRepository(productIDs...)))
	return productCache
}

func listProductIDs(products []domain.Product) []string {
	productIDs := make([]string, 0, len(products))
	for _, product := range products {
		productIDs = append(productIDs, product.ProductID)
	}
	return productIDs
}

func TestProductCache(t *testing.T) {
	t.Run("[プロダクト定義キャッシュ]プロダクト定義の読み込み", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("プロダクト定義がTST-0001の1件のとき、読み込むと", func(t *testing.T) {
				t.Run("TST-0001で参照したプロダクト定義のプロダクトIDがTST-0001になる", func(t *testing.T) {
					productCache := loadProductCache(t, "TST-0001")

					product := productCache.FindByID("TST-0001")

					require.NotNil(t, product)
					assert.Equal(t, "TST-0001", product.ProductID)
				})

				t.Run("保持件数が1件になる", func(t *testing.T) {
					productCache := loadProductCache(t, "TST-0001")

					assert.Equal(t, 1, productCache.Count())
				})
			})

			t.Run("プロダクト定義がTST-0001からTST-0003の3件のとき、読み込むと", func(t *testing.T) {
				t.Run("TST-0002で参照したプロダクト定義のプロダクトIDがTST-0002になる", func(t *testing.T) {
					productCache := loadProductCache(t, "TST-0001", "TST-0002", "TST-0003")

					product := productCache.FindByID("TST-0002")

					require.NotNil(t, product)
					assert.Equal(t, "TST-0002", product.ProductID)
				})

				t.Run("TST-0004ではプロダクト定義が見つからない", func(t *testing.T) {
					productCache := loadProductCache(t, "TST-0001", "TST-0002", "TST-0003")

					assert.Nil(t, productCache.FindByID("TST-0004"))
				})

				t.Run("保持件数が3件になる", func(t *testing.T) {
					productCache := loadProductCache(t, "TST-0001", "TST-0002", "TST-0003")

					assert.Equal(t, 3, productCache.Count())
				})
			})

			t.Run("TST-0001とTST-0002を読み込み済みで、保管先がTST-0004の1件だけを返すとき、再度読み込むと", func(t *testing.T) {
				t.Run("一覧のプロダクトIDがTST-0004だけになる", func(t *testing.T) {
					productCache := loadProductCache(t, "TST-0001", "TST-0002")
					require.NoError(t, productCache.Load(context.Background(), cachetest.NewFakeProductRepository("TST-0004")))

					assert.Equal(t, []string{"TST-0004"}, listProductIDs(productCache.FindAll()))
				})

				t.Run("TST-0001でもTST-0002でもプロダクト定義が見つからない", func(t *testing.T) {
					productCache := loadProductCache(t, "TST-0001", "TST-0002")
					require.NoError(t, productCache.Load(context.Background(), cachetest.NewFakeProductRepository("TST-0004")))

					assert.Nil(t, productCache.FindByID("TST-0001"))
					assert.Nil(t, productCache.FindByID("TST-0002"))
				})
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("プロダクト定義の保管先がstore-unreachableというエラーを返したとき、読み込みのエラーの原因にstore-unreachableが含まれる", func(t *testing.T) {
				repository := &cachetest.FakeProductRepository{Err: cachetest.ErrStoreUnreachable}

				err := cache.NewProductCache().Load(context.Background(), repository)

				assert.ErrorIs(t, err, cachetest.ErrStoreUnreachable)
			})

			t.Run("プロダクト定義の保管先が0件を返したとき、読み込みはプロダクト定義が0件であることを示すエラーを返す", func(t *testing.T) {
				err := cache.NewProductCache().Load(context.Background(), cachetest.NewFakeProductRepository())

				assert.ErrorIs(t, err, cache.ErrNoProductDefinitions)
			})

			t.Run("TST-0001とTST-0002をこの順で読み込み済みのキャッシュに再度読み込んで失敗したとき", func(t *testing.T) {
				tests := []struct {
					name       string
					repository *cachetest.FakeProductRepository
				}{
					{
						name:       "保管先が取得に失敗したとき、一覧のプロダクトIDの並びがTST-0001、TST-0002になる",
						repository: &cachetest.FakeProductRepository{Err: cachetest.ErrStoreUnreachable},
					},
					{
						name:       "保管先が0件を返したとき、一覧のプロダクトIDの並びがTST-0001、TST-0002になる",
						repository: cachetest.NewFakeProductRepository(),
					},
				}
				for _, tt := range tests {
					t.Run(tt.name, func(t *testing.T) {
						productCache := loadProductCache(t, "TST-0001", "TST-0002")
						require.Error(t, productCache.Load(context.Background(), tt.repository))

						assert.Equal(t, []string{"TST-0001", "TST-0002"}, listProductIDs(productCache.FindAll()))
					})
				}
			})
		})
	})
}
