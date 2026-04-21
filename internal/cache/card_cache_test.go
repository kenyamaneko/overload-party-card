package cache

import (
	"testing"

	gencache "github.com/kenyamaneko/overload-party-card/data/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var resourceCardTypes = map[string]bool{"Compute": true, "Data": true}

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

func TestLoadFromBytes_CardCount(t *testing.T) {
	cc := loadTestCache(t)
	require.NotZero(t, cc.Count(), "no cards loaded")
}

// TestResourceLabel_PresenceMatchesType は、resource_label の有無が
// CardType の resource/support 区分と一致することを確認します。
// 1 枚ごとに「リソース種別なら label あり／それ以外なら label なし」を
// 等式で検証することで、if による分岐フィルタを不要にしています。
func TestResourceLabel_PresenceMatchesType(t *testing.T) {
	cc := loadTestCache(t)
	for cardID, card := range cc.All() {
		isResource := isResourceType(card.CardType)
		hasLabel := card.ResourceLabel != ""
		assert.Equalf(t, isResource, hasLabel,
			"card %s (type=%s, label=%q): resource types must have resource_label, support types must not",
			cardID, card.CardType, card.ResourceLabel)
	}
}
