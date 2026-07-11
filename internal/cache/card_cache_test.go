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

func TestLoadFromBytes(t *testing.T) {
	t.Run("生成カードキャッシュのロード", func(t *testing.T) {
		t.Run("全カードで resource 種別なら resource_label があり、support 種別なら無い", func(t *testing.T) {
			// リソース種別か否かと label 有無を 1 枚ごとに等式で突き合わせ、if 分岐なしで網羅する。
			cc := loadTestCache(t)
			for cardID, card := range cc.All() {
				isResource := isResourceType(card.CardType)
				hasLabel := card.ResourceLabel != ""
				assert.Equalf(t, isResource, hasLabel,
					"card %s (type=%s, label=%q): resource types must have resource_label, support types must not",
					cardID, card.CardType, card.ResourceLabel)
			}
		})
	})

	t.Run("既知カード 2 件の制御フィクスチャをロードしたとき", func(t *testing.T) {
		cc := NewCardCache()
		require.NoError(t, cc.LoadFromBytes([]byte(controlCardsJSON)))

		t.Run("件数と card_id ごとの主要フィールドが保たれる", func(t *testing.T) {
			assert.Equal(t, 2, cc.Count())

			alpha := cc.Get("TST-0001")
			require.NotNil(t, alpha)
			assert.Equal(t, "Alpha", alpha.CardName)
			assert.Equal(t, gamedesign.CardTypeCompute, alpha.CardType)

			beta := cc.Get("TST-0002")
			require.NotNil(t, beta)
			assert.Equal(t, "Beta", beta.CardName)
		})

		t.Run("未知の card_id では nil が返る", func(t *testing.T) {
			assert.Nil(t, cc.Get("TST-9999"))
		})
	})
}
