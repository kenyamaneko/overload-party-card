package rest_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

// newTestProductCache builds a cache.ProductCache preloaded with the given products via LoadFromBytes.
func newTestProductCache(t *testing.T, products ...domain.Product) *cache.ProductCache {
	t.Helper()
	c := cache.NewProductCache()
	encoded, err := json.Marshal(products)
	require.NoError(t, err)
	require.NoError(t, c.LoadFromBytes(encoded))
	return c
}

// newTestInitiativeCache builds a cache.InitiativeCache preloaded with the given initiatives via LoadFromBytes.
func newTestInitiativeCache(t *testing.T, initiatives ...domain.Initiative) *cache.InitiativeCache {
	t.Helper()
	c := cache.NewInitiativeCache()
	encoded, err := json.Marshal(initiatives)
	require.NoError(t, err)
	require.NoError(t, c.LoadFromBytes(encoded))
	return c
}

func TestProductHandlerListAll(t *testing.T) {
	t.Run("[プロダクトAPI] プロダクトマスター配信", func(t *testing.T) {
		t.Run("プロダクトが1件以上登録されているとき、200になり各要素の内容がプロダクトキャッシュの登録値と一致する", func(t *testing.T) {
			product := domain.Product{ProductID: "PD-TST01", Faction: "SHE", ProductName: "テストプロダクト", IsActive: true}
			initiative := domain.Initiative{InitiativeID: "IN-TST01", ProductID: "PD-TST01", Kind: domain.InitiativeKindRoutine, Name: "テスト施策", InsightCost: 1, EffectText: "効果", Effect: json.RawMessage(`{}`), IsActive: true}
			engine := newTestRouter(t,
				withProductCache(newTestProductCache(t, product)),
				withInitiativeCache(newTestInitiativeCache(t, initiative)),
			)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/products", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			var body []productWithInitiativesBody
			decodeJSON(t, rr, &body)
			require.Len(t, body, 1)
			assert.Equal(t, product.ProductID, body[0].ProductID)
			assert.Equal(t, product.Faction, body[0].Faction)
			assert.Equal(t, product.ProductName, body[0].ProductName)
			assert.Equal(t, product.IsActive, body[0].IsActive)
		})

		t.Run("プロダクトに紐づく施策が施策キャッシュに1件以上存在するとき、そのプロダクトのinitiatives配列にそれらの施策が含まれる", func(t *testing.T) {
			product := domain.Product{ProductID: "PD-TST01", Faction: "SHE", ProductName: "テストプロダクト", IsActive: true}
			initiative := domain.Initiative{InitiativeID: "IN-TST01", ProductID: "PD-TST01", Kind: domain.InitiativeKindRoutine, Name: "テスト施策", InsightCost: 1, EffectText: "効果", Effect: json.RawMessage(`{}`), IsActive: true}
			engine := newTestRouter(t,
				withProductCache(newTestProductCache(t, product)),
				withInitiativeCache(newTestInitiativeCache(t, initiative)),
			)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/products", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			var body []productWithInitiativesBody
			decodeJSON(t, rr, &body)
			require.Len(t, body, 1)
			require.Len(t, body[0].Initiatives, 1)
			assert.Equal(t, initiative.InitiativeID, body[0].Initiatives[0].InitiativeID)
		})

		t.Run("プロダクトに紐づく施策が施策キャッシュに1件も存在しないとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			product := domain.Product{ProductID: "PD-TST01", Faction: "SHE", ProductName: "テストプロダクト", IsActive: true}
			otherInitiative := domain.Initiative{InitiativeID: "IN-TST01", ProductID: "PD-OTHER", Kind: domain.InitiativeKindRoutine, Name: "テスト施策", InsightCost: 1, EffectText: "効果", Effect: json.RawMessage(`{}`), IsActive: true}
			engine := newTestRouter(t,
				withProductCache(newTestProductCache(t, product)),
				withInitiativeCache(newTestInitiativeCache(t, otherInitiative)),
			)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/products", nil)

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("同一プロダクトに紐づく施策が複数存在するとき、initiatives配列は施策IDの昇順で並ぶ", func(t *testing.T) {
			product := domain.Product{ProductID: "PD-TST01", Faction: "SHE", ProductName: "テストプロダクト", IsActive: true}
			second := domain.Initiative{InitiativeID: "IN-TST02", ProductID: "PD-TST01", Kind: domain.InitiativeKindSpecial, Name: "施策2", InsightCost: 1, EffectText: "効果2", Effect: json.RawMessage(`{}`), IsActive: true}
			first := domain.Initiative{InitiativeID: "IN-TST01", ProductID: "PD-TST01", Kind: domain.InitiativeKindRoutine, Name: "施策1", InsightCost: 1, EffectText: "効果1", Effect: json.RawMessage(`{}`), IsActive: true}
			engine := newTestRouter(t,
				withProductCache(newTestProductCache(t, product)),
				withInitiativeCache(newTestInitiativeCache(t, second, first)),
			)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/products", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			var body []productWithInitiativesBody
			decodeJSON(t, rr, &body)
			require.Len(t, body, 1)
			require.Len(t, body[0].Initiatives, 2)
			assert.Equal(t, "IN-TST01", body[0].Initiatives[0].InitiativeID)
			assert.Equal(t, "IN-TST02", body[0].Initiatives[1].InitiativeID)
		})

		t.Run("プロダクトキャッシュが1件もプロダクトを保持していないとき、200になりレスポンスボディは空配列になる", func(t *testing.T) {
			engine := newTestRouter(t,
				withProductCache(cache.NewProductCache()),
				withInitiativeCache(cache.NewInitiativeCache()),
			)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/products", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			assert.JSONEq(t, "[]", rr.Body.String())
		})
	})
}

// productWithInitiativesBody mirrors the ProductHandler.ListAll response element shape
// (domain.Product fields flattened alongside an initiatives array).
type productWithInitiativesBody struct {
	ProductID   string              `json:"product_id"`
	Faction     string              `json:"faction"`
	ProductName string              `json:"product_name"`
	IsActive    bool                `json:"is_active"`
	Initiatives []domain.Initiative `json:"initiatives"`
}
