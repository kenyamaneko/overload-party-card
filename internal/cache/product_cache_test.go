package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubProductRepo は port.ProductRepo のテスト用スタブ。
type stubProductRepo struct {
	products []*domain.Product
	err      error
}

func (r *stubProductRepo) FindAll(_ context.Context) ([]*domain.Product, error) {
	return r.products, r.err
}

// testProductsFixtureJSON は ID 検索の期待値を固定するための制御フィクスチャ。
// 生成データの並びに依存しないよう、既知の product_id を持つ複数件を用意する。
const testProductsFixtureJSON = `[
  {"product_id":"PD-TST-0001","faction":"SHE","product_name":"テストプロダクト1","is_active":true},
  {"product_id":"PD-TST-0002","faction":"ORD","product_name":"テストプロダクト2","is_active":true}
]`

func TestProductFindByID(t *testing.T) {
	t.Run("プロダクトの ID 検索", func(t *testing.T) {
		pc := NewProductCache()
		require.NoError(t, pc.LoadFromBytes([]byte(testProductsFixtureJSON)))

		t.Run("既知の ID のとき、該当するプロダクトが返る", func(t *testing.T) {
			got := pc.FindByID("PD-TST-0001")
			require.NotNil(t, got)
			assert.Equal(t, "PD-TST-0001", got.ProductID)
			assert.Equal(t, "SHE", got.Faction)
		})

		t.Run("未知の ID のとき、nil を返す", func(t *testing.T) {
			assert.Nil(t, pc.FindByID("PD-NOPE"))
		})
	})
}

func TestProductLoadFromBytes(t *testing.T) {
	t.Run("プロダクトキャッシュのロード", func(t *testing.T) {
		t.Run("0 件 JSON のとき、マスター欠落としてエラーになる", func(t *testing.T) {
			pc := NewProductCache()
			err := pc.LoadFromBytes([]byte(`[]`))
			assert.Error(t, err)
		})

		t.Run("JSON として不正なバイト列のとき、読み込みがエラーになる", func(t *testing.T) {
			pc := NewProductCache()
			err := pc.LoadFromBytes([]byte(`{`))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "parse product data")
		})
	})
}

func TestProductLoad(t *testing.T) {
	t.Run("DB からのプロダクトキャッシュ読み込み", func(t *testing.T) {
		t.Run("DB に定義があるとき、読み込んだ定義が検索で引けるようになる", func(t *testing.T) {
			repo := &stubProductRepo{products: []*domain.Product{
				{ProductID: "PD-TST-0001", Faction: "SHE", ProductName: "テストプロダクト"},
			}}
			pc := NewProductCache()

			err := pc.Load(context.Background(), repo)

			require.NoError(t, err)
			got := pc.FindByID("PD-TST-0001")
			require.NotNil(t, got)
			assert.Equal(t, "テストプロダクト", got.ProductName)
		})

		t.Run("DB が 0 件のとき、マスター欠落としてエラーになる", func(t *testing.T) {
			repo := &stubProductRepo{products: nil}
			pc := NewProductCache()

			err := pc.Load(context.Background(), repo)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "0 products loaded")
		})

		t.Run("DB 読み込みが失敗したとき、そのエラーが伝播する", func(t *testing.T) {
			dbErr := errors.New("db down")
			repo := &stubProductRepo{err: dbErr}
			pc := NewProductCache()

			err := pc.Load(context.Background(), repo)

			require.Error(t, err)
			assert.ErrorIs(t, err, dbErr)
		})
	})
}
