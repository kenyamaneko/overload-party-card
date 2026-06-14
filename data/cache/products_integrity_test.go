package cache

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gamedesign "github.com/kenyamaneko/overload-party-common/packages/game-design-constants"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

func loadProductMaster(t *testing.T) []domain.Product {
	t.Helper()
	var products []domain.Product
	require.NoError(t, json.Unmarshal(ProductsJSON, &products), "unmarshal products_gen.json")
	require.NotEmpty(t, products, "products_gen.json must not be empty")
	return products
}

func loadInitiativeMaster(t *testing.T) []domain.Initiative {
	t.Helper()
	var initiatives []domain.Initiative
	require.NoError(t, json.Unmarshal(InitiativesJSON, &initiatives), "unmarshal initiatives_gen.json")
	require.NotEmpty(t, initiatives, "initiatives_gen.json must not be empty")
	return initiatives
}

// TestProductMaster_EveryFactionHasProduct は、選択可能な全陣営に
// 少なくとも 1 つのプロダクトが存在する仕様 (陣営:プロダクト = 1:N) を検証します。
func TestProductMaster_EveryFactionHasProduct(t *testing.T) {
	products := loadProductMaster(t)

	byFaction := make(map[string]int)
	for _, product := range products {
		byFaction[product.Faction]++
	}

	for _, faction := range gamedesign.SelectableFactions {
		assert.GreaterOrEqualf(t, byFaction[faction], 1, "faction %s must have at least one product", faction)
	}
}

// TestInitiativeMaster_EachProductHasRoutineAndSpecial は、各プロダクトが
// ルーチンとスペシャルをそれぞれ 1 つ以上持つ仕様を検証します (数は区分ごとに異なってよい)。
func TestInitiativeMaster_EachProductHasRoutineAndSpecial(t *testing.T) {
	kindsByProduct := make(map[string]map[string]int)
	for _, product := range loadProductMaster(t) {
		kindsByProduct[product.ProductID] = map[string]int{}
	}

	for _, initiative := range loadInitiativeMaster(t) {
		require.Containsf(t, kindsByProduct, initiative.ProductID,
			"initiative %s references unknown product %s", initiative.InitiativeID, initiative.ProductID)
		kindsByProduct[initiative.ProductID][initiative.Kind]++
	}

	for productID, kinds := range kindsByProduct {
		assert.GreaterOrEqualf(t, kinds[domain.InitiativeKindRoutine], 1, "product %s must have at least one routine", productID)
		assert.GreaterOrEqualf(t, kinds[domain.InitiativeKindSpecial], 1, "product %s must have at least one special", productID)
	}
}

// TestInitiativeMaster_FieldsPopulated は、施策の必須フィールドが
// 全件埋まっていることを検証します。
func TestInitiativeMaster_FieldsPopulated(t *testing.T) {
	for _, initiative := range loadInitiativeMaster(t) {
		label := initiative.ProductID + "/" + initiative.Kind
		assert.NotEmptyf(t, initiative.InitiativeID, "%s: initiative_id", label)
		assert.NotEmptyf(t, initiative.ProductID, "%s: product_id", label)
		assert.NotEmptyf(t, initiative.Name, "%s: name", label)
		assert.NotEmptyf(t, initiative.EffectText, "%s: effect_text", label)
		assert.NotEmptyf(t, initiative.Effect, "%s: effect", label)
		assert.GreaterOrEqualf(t, initiative.InsightCost, int64(0), "%s: insight_cost", label)
	}
}
