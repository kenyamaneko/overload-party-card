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

// TestProductMaster_EachProductHasRoutineAndSpecial は、各プロダクトが
// ルーチンとスペシャルをそれぞれ 1 つ以上持つ仕様を検証します (数は区分ごとに異なってよい)。
func TestProductMaster_EachProductHasRoutineAndSpecial(t *testing.T) {
	products := loadProductMaster(t)

	for _, product := range products {
		kinds := make(map[string]int)
		for _, initiative := range product.Initiatives {
			kinds[initiative.Kind]++
		}
		assert.GreaterOrEqualf(t, kinds[domain.InitiativeKindRoutine], 1, "product %s must have at least one routine", product.ProductID)
		assert.GreaterOrEqualf(t, kinds[domain.InitiativeKindSpecial], 1, "product %s must have at least one special", product.ProductID)
	}
}

// TestProductMaster_InitiativeFieldsPopulated は、施策の必須フィールドが
// 全件埋まっていることを検証します。
func TestProductMaster_InitiativeFieldsPopulated(t *testing.T) {
	products := loadProductMaster(t)

	for _, product := range products {
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
