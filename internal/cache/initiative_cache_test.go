package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testInitiativesFixtureJSON は ID 検索の期待値を固定するための制御フィクスチャ。
// 生成データの並びに依存しないよう、既知の initiative_id を持つ複数件を用意する。
const testInitiativesFixtureJSON = `[
  {"initiative_id":"IN-TST-0001","product_id":"PD-TST-0001","kind":"routine","name":"テスト施策1","insight_cost":100,"effect_text":"","effect":{"ops":[]},"is_active":true},
  {"initiative_id":"IN-TST-0002","product_id":"PD-TST-0002","kind":"special","name":"テスト施策2","insight_cost":200,"effect_text":"","effect":{"ops":[]},"is_active":true}
]`

func TestInitiativeFindByID(t *testing.T) {
	t.Run("施策の ID 検索", func(t *testing.T) {
		ic := NewInitiativeCache()
		require.NoError(t, ic.LoadFromBytes([]byte(testInitiativesFixtureJSON)))

		t.Run("既知の ID のとき、該当する施策が返る", func(t *testing.T) {
			got := ic.FindByID("IN-TST-0001")
			require.NotNil(t, got)
			assert.Equal(t, "IN-TST-0001", got.InitiativeID)
			assert.Equal(t, "routine", got.Kind)
			assert.Equal(t, "テスト施策1", got.Name)
		})

		t.Run("未知の ID のとき、nil を返す", func(t *testing.T) {
			assert.Nil(t, ic.FindByID("IN-NOPE"))
		})
	})
}

func TestInitiativeLoadFromBytes(t *testing.T) {
	t.Run("施策キャッシュのロード", func(t *testing.T) {
		t.Run("0 件 JSON のとき、マスター欠落としてエラーになる", func(t *testing.T) {
			ic := NewInitiativeCache()
			err := ic.LoadFromBytes([]byte(`[]`))
			assert.Error(t, err)
		})
	})
}
