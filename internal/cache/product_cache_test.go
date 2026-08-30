package cache_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

type fakeProductRepo struct {
	products []*domain.Product
	err      error
}

func (f *fakeProductRepo) FindAll(ctx context.Context) ([]*domain.Product, error) {
	return f.products, f.err
}

func TestProductCacheFindByID(t *testing.T) {
	t.Run("[プロダクトキャッシュ] プロダクト定義の取得", func(t *testing.T) {
		t.Run("複数件のプロダクト定義を読み込んだあと、読み込み済みのproduct_id TST-0001を指定して取得すると、そのproduct_idに対応するプロダクト定義が返る", func(t *testing.T) {
			c := cache.NewProductCache()
			repo := &fakeProductRepo{products: []*domain.Product{
				{ProductID: "TST-0001"},
				{ProductID: "TST-0002"},
			}}
			require.NoError(t, c.Load(context.Background(), repo))

			got := c.FindByID("TST-0001")

			require.NotNil(t, got)
			assert.Equal(t, "TST-0001", got.ProductID)
		})

		t.Run("複数件のプロダクト定義を読み込んだあと、読み込んでいないproduct_id TST-9999を指定して取得すると、見つからない", func(t *testing.T) {
			c := cache.NewProductCache()
			repo := &fakeProductRepo{products: []*domain.Product{
				{ProductID: "TST-0001"},
				{ProductID: "TST-0002"},
			}}
			require.NoError(t, c.Load(context.Background(), repo))

			got := c.FindByID("TST-9999")

			assert.Nil(t, got)
		})
	})
}

func TestProductCacheLoad(t *testing.T) {
	t.Run("[プロダクトキャッシュ] プロダクト定義の読み込み", func(t *testing.T) {
		cases := []struct {
			name           string
			repo           *fakeProductRepo
			wantErrContain string
		}{
			{
				name:           "プロダクト定義の取得元が0件を返したとき、読み込みはエラーになる",
				repo:           &fakeProductRepo{products: nil},
				wantErrContain: "0 products loaded",
			},
			{
				name:           "プロダクト定義の取得元がエラーを返したとき、読み込みはエラーになる",
				repo:           &fakeProductRepo{err: errors.New("fetch failed")},
				wantErrContain: "load product cache",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				c := cache.NewProductCache()

				err := c.Load(context.Background(), tc.repo)

				assert.ErrorContains(t, err, tc.wantErrContain)
			})
		}
	})
}

func TestProductCacheCount(t *testing.T) {
	t.Run("[プロダクトキャッシュ] プロダクト定義の件数", func(t *testing.T) {
		t.Run("読み込み前の件数は0になる", func(t *testing.T) {
			c := cache.NewProductCache()

			assert.Equal(t, 0, c.Count())
		})

		t.Run("プロダクト定義を読み込んだあと、件数は読み込んだ件数になる", func(t *testing.T) {
			c := cache.NewProductCache()
			repo := &fakeProductRepo{products: []*domain.Product{
				{ProductID: "TST-0001"},
				{ProductID: "TST-0002"},
				{ProductID: "TST-0003"},
			}}
			require.NoError(t, c.Load(context.Background(), repo))

			assert.Equal(t, 3, c.Count())
		})
	})
}

func TestProductCacheAll(t *testing.T) {
	t.Run("[プロダクトキャッシュ] プロダクト定義の全件取得", func(t *testing.T) {
		t.Run("プロダクト定義を読み込んだあと、全件取得すると読み込んだ全件を含む", func(t *testing.T) {
			c := cache.NewProductCache()
			repo := &fakeProductRepo{products: []*domain.Product{
				{ProductID: "TST-0001"},
				{ProductID: "TST-0002"},
			}}
			require.NoError(t, c.Load(context.Background(), repo))

			got := c.All()

			ids := make([]string, len(got))
			for i, p := range got {
				ids[i] = p.ProductID
			}
			assert.ElementsMatch(t, []string{"TST-0001", "TST-0002"}, ids)
		})
	})
}

func TestProductCacheLoadFromBytes(t *testing.T) {
	t.Run("[プロダクトキャッシュ] JSONからのプロダクト定義読み込み", func(t *testing.T) {
		t.Run("プロダクト定義1件以上を含むJSONデータから読み込むと、そのプロダクト定義が取得できるようになる", func(t *testing.T) {
			c := cache.NewProductCache()

			err := c.LoadFromBytes([]byte(`[{"product_id":"TST-0001"}]`))
			require.NoError(t, err)

			got := c.FindByID("TST-0001")
			require.NotNil(t, got)
			assert.Equal(t, "TST-0001", got.ProductID)
		})

		cases := []struct {
			name           string
			data           []byte
			wantErrContain string
		}{
			{"要素0件のJSON配列から読み込むと、エラーになる", []byte(`[]`), "0 products loaded"},
			{"JSONとして解釈できないデータから読み込むと、エラーになる", []byte(`not json`), "parse product data"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				c := cache.NewProductCache()

				err := c.LoadFromBytes(tc.data)

				assert.ErrorContains(t, err, tc.wantErrContain)
			})
		}
	})
}
