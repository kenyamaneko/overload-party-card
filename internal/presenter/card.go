package presenter

import (
	"encoding/json"

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
		Effects:       optionalRawMessage(c.Effects),
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

// ToCardWithOwnership は domain.Card と所持フラグから wire の CardWithOwnership を組み立てます。
func ToCardWithOwnership(c *domain.Card, isOwned bool) *apicard.CardWithOwnership {
	return &apicard.CardWithOwnership{
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
		Effects:       optionalRawMessage(c.Effects),
		Restriction:   c.Restriction,
		IsActive:      c.IsActive,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
		IsOwned:       isOwned,
	}
}

// optionalRawMessage は空 (nil または長さ 0) の RawMessage を nil ポインタに正規化します。
// oapi-codegen 生成型は optional な JSON object フィールドを *json.RawMessage として
// 出力するため、空値時に omitempty を効かせるにはポインタを nil にする必要があります。
func optionalRawMessage(b json.RawMessage) *json.RawMessage {
	if len(b) == 0 {
		return nil
	}
	return &b
}
