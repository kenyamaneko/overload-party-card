package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gencache "github.com/kenyamaneko/overload-party-card/data/cache"
)

func loadTestProductCache(t *testing.T) *ProductCache {
	t.Helper()
	pc := NewProductCache()
	err := pc.LoadFromBytes(gencache.ProductsJSON)
	require.NoError(t, err, "LoadFromBytes failed")
	return pc
}

func TestProductFindByID(t *testing.T) {
	t.Run("プロダクトの ID 検索", func(t *testing.T) {
		pc := loadTestProductCache(t)
		existingID := pc.All()[0].ProductID

		tests := []struct {
			name      string
			productID string
			wantNil   bool
		}{
			{"既知の ID のとき、プロダクトを引ける", existingID, false},
			{"未知の ID のとき、nil を返す", "PD-NOPE", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.wantNil, pc.FindByID(tt.productID) == nil)
			})
		}
	})
}

func TestProductLoadFromBytes(t *testing.T) {
	t.Run("プロダクトキャッシュのロード", func(t *testing.T) {
		t.Run("0 件 JSON のとき、マスター欠落としてエラーになる", func(t *testing.T) {
			pc := NewProductCache()
			err := pc.LoadFromBytes([]byte(`[]`))
			assert.Error(t, err)
		})
	})
}
