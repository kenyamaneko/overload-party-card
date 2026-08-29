package rest_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

func TestInitiativeHandlerListAll(t *testing.T) {
	t.Run("[handler] 施策マスター配信", func(t *testing.T) {
		t.Run("施策が1件以上登録されているとき、200になり登録した施策の内容と一致する配列になる", func(t *testing.T) {
			initiative := domain.Initiative{
				InitiativeID: "IN-TST01",
				ProductID:    "PD-TST01",
				Kind:         domain.InitiativeKindRoutine,
				Name:         "テスト施策",
				InsightCost:  3,
				EffectText:   "効果テキスト",
				Effect:       json.RawMessage(`{"op":"noop"}`),
				IsActive:     true,
			}
			engine := newTestRouter(t, withInitiativeCache(newTestInitiativeCache(t, initiative)))

			rr := doRequest(t, engine, http.MethodGet, "/internal/v1/initiatives", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			var body []apicard.Initiative
			decodeJSON(t, rr, &body)
			require.Len(t, body, 1)
			got := body[0]
			assert.Equal(t, initiative.InitiativeID, got.InitiativeID)
			assert.Equal(t, initiative.ProductID, got.ProductID)
			assert.Equal(t, string(initiative.Kind), string(got.Kind))
			assert.Equal(t, initiative.Name, got.Name)
			assert.Equal(t, initiative.InsightCost, got.InsightCost)
			assert.Equal(t, initiative.EffectText, got.EffectText)
			assert.JSONEq(t, string(initiative.Effect), toJSON(t, got.Effect))
			assert.Equal(t, initiative.IsActive, got.IsActive)
		})

		t.Run("施策が複数登録されているとき、レスポンスの配列はinitiative_idの昇順で並ぶ", func(t *testing.T) {
			second := domain.Initiative{InitiativeID: "IN-TST02", ProductID: "PD-TST01", Kind: domain.InitiativeKindSpecial, Name: "施策2", InsightCost: 1, EffectText: "効果2", Effect: json.RawMessage(`{}`), IsActive: true}
			first := domain.Initiative{InitiativeID: "IN-TST01", ProductID: "PD-TST01", Kind: domain.InitiativeKindRoutine, Name: "施策1", InsightCost: 1, EffectText: "効果1", Effect: json.RawMessage(`{}`), IsActive: true}
			engine := newTestRouter(t, withInitiativeCache(newTestInitiativeCache(t, second, first)))

			rr := doRequest(t, engine, http.MethodGet, "/internal/v1/initiatives", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			var body []apicard.Initiative
			decodeJSON(t, rr, &body)
			require.Len(t, body, 2)
			assert.Equal(t, "IN-TST01", body[0].InitiativeID)
			assert.Equal(t, "IN-TST02", body[1].InitiativeID)
		})

		t.Run("施策が1件も登録されていないとき、200になりレスポンスボディは空配列になる", func(t *testing.T) {
			engine := newTestRouter(t, withInitiativeCache(cache.NewInitiativeCache()))

			rr := doRequest(t, engine, http.MethodGet, "/internal/v1/initiatives", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			assert.JSONEq(t, "[]", rr.Body.String())
		})
	})
}

// toJSON re-encodes v for comparison against a domain.Initiative.Effect json.RawMessage fixture.
func toJSON(t *testing.T, v map[string]interface{}) string {
	t.Helper()
	encoded, err := json.Marshal(v)
	require.NoError(t, err)
	return string(encoded)
}
