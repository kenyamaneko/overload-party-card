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

// TestProductLoadFromBytes_EveryFactionHasProduct は、選択可能な全陣営に
// 少なくとも 1 つのプロダクトが存在する仕様 (陣営:プロダクト = 1:N) を検証します。
func TestProductLoadFromBytes_EveryFactionHasProduct(t *testing.T) {
	pc := loadTestProductCache(t)

	byFaction := make(map[string]int)
	for _, product := range pc.All() {
		byFaction[product.Faction]++
	}

	for _, faction := range []string{"SHE", "Tenki", "Sugar", "Tuners"} {
		assert.GreaterOrEqualf(t, byFaction[faction], 1, "faction %s must have at least one product", faction)
	}
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

// TestProductLoadFromBytes_InitiativeKinds は、各プロダクトがルーチンと
// スペシャルをそれぞれ 1 つ以上持つ仕様を検証します (数は区分ごとに異なってよい)。
func TestProductLoadFromBytes_InitiativeKinds(t *testing.T) {
	pc := loadTestProductCache(t)

	for _, product := range pc.All() {
		kinds := make(map[string]int)
		for _, initiative := range product.Initiatives {
			kinds[initiative.Kind]++
		}
		assert.GreaterOrEqualf(t, kinds["routine"], 1, "product %s must have at least one routine", product.ProductID)
		assert.GreaterOrEqualf(t, kinds["special"], 1, "product %s must have at least one special", product.ProductID)
	}
}

// TestProductLoadFromBytes_InitiativeFields は、施策の必須フィールドが
// 全件埋まっていることを検証します。
func TestProductLoadFromBytes_InitiativeFields(t *testing.T) {
	pc := loadTestProductCache(t)

	for _, product := range pc.All() {
		for _, initiative := range product.Initiatives {
			label := product.ProductID + "/" + initiative.Kind
			assert.NotEmptyf(t, initiative.InitiativeID, "%s: initiative_id", label)
			assert.NotEmptyf(t, initiative.Name, "%s: name", label)
			assert.NotEmptyf(t, initiative.EffectText, "%s: effect_text", label)
			assert.NotEmptyf(t, initiative.Effect, "%s: effect", label)
			assert.GreaterOrEqualf(t, initiative.InsightCost, int64(0), "%s: insight_cost", label)
		}
	}
}

// TestProductLoadFromBytes_Empty は、0 件 JSON がマスター欠落としてエラーに
// なることを検証します。
func TestProductLoadFromBytes_Empty(t *testing.T) {
	pc := NewProductCache()
	err := pc.LoadFromBytes([]byte(`[]`))
	assert.Error(t, err)
}
