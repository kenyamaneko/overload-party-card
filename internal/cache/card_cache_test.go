package cache

import (
	"testing"

	gencache "github.com/kenyamaneko/overload-party-card/data/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func isResourceType(cardType string) bool {
	switch cardType {
	case "Compute", "Container", "Orchestrator", "Serverless", "AI/ML", "Database", "CacheDB", "ObjectStorage":
		return true
	}
	return false
}

func loadTestCache(t *testing.T) *CardCache {
	t.Helper()
	cc := NewCardCache()
	if err := cc.LoadFromBytes(gencache.CardsJSON); err != nil {
		t.Fatalf("LoadFromBytes: %v", err)
	}
	return cc
}

func TestLoadFromBytes_CardCount(t *testing.T) {
	cc := loadTestCache(t)
	require.NotZero(t, cc.Count(), "no cards loaded")
}

func TestResourceLabel_ResourceCardsHaveLabel(t *testing.T) {
	cc := loadTestCache(t)
	for cardID, card := range cc.All() {
		if isResourceType(card.CardType) {
			assert.NotEmptyf(t, card.ResourceLabel,
				"resource card %s (%s, type=%s) has empty resource_label",
				cardID, card.CardName, card.CardType)
		}
	}
}

func TestResourceLabel_SupportCardsHaveNoLabel(t *testing.T) {
	cc := loadTestCache(t)
	for cardID, card := range cc.All() {
		if !isResourceType(card.CardType) {
			assert.Emptyf(t, card.ResourceLabel,
				"support card %s (%s, type=%s) should have empty resource_label, got %q",
				cardID, card.CardName, card.CardType, card.ResourceLabel)
		}
	}
}
