package cache

import (
	"testing"

	gencache "github.com/kenyamaneko/overload-party-card/data/cache"
	gamedesign "github.com/kenyamaneko/overload-party-common/packages/game-design-constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var resourceCardTypes = map[string]bool{
	gamedesign.CardTypeCompute:      true,
	gamedesign.CardTypeDataResource: true,
}

func isResourceType(cardType string) bool {
	return resourceCardTypes[cardType]
}

func loadTestCache(t *testing.T) *CardCache {
	t.Helper()
	cc := NewCardCache()
	err := cc.LoadFromBytes(gencache.CardsJSON)
	require.NoError(t, err, "LoadFromBytes failed")
	return cc
}

// controlCardsJSON は既知カード 2 件の制御フィクスチャ。生成データの件数・並びに
// 依存せず「LoadFromBytes した各カードを card_id で引ける」ことを固定する。
const controlCardsJSON = `[
	{"card_id":"TST-0001","card_name":"Alpha","card_type":"Compute","resource_label":"S"},
	{"card_id":"TST-0002","card_name":"Beta","card_type":"Support"}
]`

// TestLoadFromBytes_LoadsCardsByID は、LoadFromBytes した各カードを card_id で
// 引けること・主要フィールドが保たれること・未知 ID では取得できないことを検証します。
func TestLoadFromBytes_LoadsCardsByID(t *testing.T) {
	cc := NewCardCache()
	require.NoError(t, cc.LoadFromBytes([]byte(controlCardsJSON)))

	assert.Equal(t, 2, cc.Count())

	alpha := cc.Get("TST-0001")
	require.NotNil(t, alpha)
	assert.Equal(t, "Alpha", alpha.CardName)
	assert.Equal(t, gamedesign.CardTypeCompute, alpha.CardType)

	beta := cc.Get("TST-0002")
	require.NotNil(t, beta)
	assert.Equal(t, "Beta", beta.CardName)

	assert.Nil(t, cc.Get("TST-9999"))
}

// TestLoadFromBytes_ResourceLabelInvariant は、resource_label の有無が
// CardType の resource/support 区分と一致することを確認します。
// 1 枚ごとに「リソース種別なら label あり／それ以外なら label なし」を
// 等式で検証することで、if による分岐フィルタを不要にしています。
func TestLoadFromBytes_ResourceLabelInvariant(t *testing.T) {
	cc := loadTestCache(t)
	for cardID, card := range cc.All() {
		isResource := isResourceType(card.CardType)
		hasLabel := card.ResourceLabel != ""
		assert.Equalf(t, isResource, hasLabel,
			"card %s (type=%s, label=%q): resource types must have resource_label, support types must not",
			cardID, card.CardType, card.ResourceLabel)
	}
}
