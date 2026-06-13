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

// TestProductLoadFromBytes_FactionOneToOne は、選択可能な全陣営にプロダクトが
// ちょうど 1 つずつ存在する仕様 (陣営 1:1) を検証します。
func TestProductLoadFromBytes_FactionOneToOne(t *testing.T) {
	pc := loadTestProductCache(t)

	byFaction := make(map[string]int)
	for _, product := range pc.All() {
		byFaction[product.Faction]++
	}

	selectableFactions := []string{"SHE", "Tenki", "Sugar", "Tuners"}
	assert.Equal(t, len(selectableFactions), pc.Count())
	for _, faction := range selectableFactions {
		assert.Equalf(t, 1, byFaction[faction], "faction %s must have exactly one product", faction)
	}
}

// TestProductLoadFromBytes_InitiativeKinds は、各プロダクトがルーチンと
// スペシャルをちょうど 1 つずつ持つ仕様を検証します。
func TestProductLoadFromBytes_InitiativeKinds(t *testing.T) {
	pc := loadTestProductCache(t)

	for _, product := range pc.All() {
		kinds := make(map[string]int)
		for _, initiative := range product.Initiatives {
			kinds[initiative.Kind]++
		}
		assert.Equalf(t, map[string]int{"routine": 1, "special": 1}, kinds,
			"product %s must have exactly one routine and one special", product.ProductID)
	}
}

// TestProductLoadFromBytes_InitiativeFields は、施策の必須フィールドが
// 全件埋まっていることを検証します。
func TestProductLoadFromBytes_InitiativeFields(t *testing.T) {
	pc := loadTestProductCache(t)

	for _, product := range pc.All() {
		for _, initiative := range product.Initiatives {
			label := product.ProductID + "/" + initiative.Kind
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
