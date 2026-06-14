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

// TestProductFindByID は、ID でプロダクトを引けること・未知 ID で nil が返ることを検証します。
func TestProductFindByID(t *testing.T) {
	pc := loadTestProductCache(t)
	existingID := pc.All()[0].ProductID

	tests := []struct {
		name      string
		productID string
		wantNil   bool
	}{
		{"existing id returns a product", existingID, false},
		{"unknown id returns nil", "PD-NOPE", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantNil, pc.FindByID(tt.productID) == nil)
		})
	}
}

// TestProductLoadFromBytes_Empty は、0 件 JSON がマスター欠落としてエラーに
// なることを検証します。
func TestProductLoadFromBytes_Empty(t *testing.T) {
	pc := NewProductCache()
	err := pc.LoadFromBytes([]byte(`[]`))
	assert.Error(t, err)
}
