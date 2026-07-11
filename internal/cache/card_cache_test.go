package cache

import (
	"testing"

	gencache "github.com/kenyamaneko/overload-party-card/data/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var resourceCardTypes = map[string]bool{"Compute": true, "DataResource": true}

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

func TestLoadFromBytes(t *testing.T) {
	t.Run("生成カードキャッシュのロード", func(t *testing.T) {
		t.Run("生成 JSON をロードするとき、カードが 1 枚以上読み込まれる", func(t *testing.T) {
			cc := loadTestCache(t)
			require.NotZero(t, cc.Count(), "no cards loaded")
		})

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
}
