// Package presenter は domain ↔ wire DTO の境界変換を集約する層です。
// 設計意図と移行方針は docs/ARCHITECTURE.md「Presenter 層の位置づけ」を参照。
package presenter

import (
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

// ToCardDefinition は domain.Card を REST wire の CardDefinition に詰め替えます。
func ToCardDefinition(c *domain.Card) *apicard.CardDefinition {
	return &apicard.CardDefinition{
		CardID:        c.CardID,
		CardName:      c.CardName,
		ResourceLabel: c.ResourceLabel,
		Faction:       c.Faction,
		CardType:      c.CardType,
		Subtype:       c.Subtype,
		Resizable:     c.Resizable,
		Elastic:       c.Elastic,
		Stats:         c.Stats,
		EffectText:    c.EffectText,
		Effects:       c.Effects,
		Restriction:   c.Restriction,
		IsActive:      c.IsActive,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

// ToCardDefinitions は domain.Card slice を CardDefinition slice に詰め替えます。
func ToCardDefinitions(cards []*domain.Card) []*apicard.CardDefinition {
	result := make([]*apicard.CardDefinition, len(cards))
	for i, c := range cards {
		result[i] = ToCardDefinition(c)
	}
	return result
}
