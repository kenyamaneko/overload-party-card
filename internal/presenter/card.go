package presenter

import (
	"encoding/json"
	"fmt"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

// cardStatsBody はカード定義の応答に含めるステータスの JSON の形式です。
type cardStatsBody struct {
	Availability    *int64 `json:"availability,omitempty"`
	MaintenanceCost *int64 `json:"maintenance_cost,omitempty"`
	SLAPenalty      *int64 `json:"sla_penalty,omitempty"`
	Throughput      *int64 `json:"throughput,omitempty"`
	ThroughputMax   *int64 `json:"throughput_max,omitempty"`
	Yield           *int64 `json:"yield,omitempty"`
	YieldMax        *int64 `json:"yield_max,omitempty"`
}

// ToCardsWithOwnership は所持状態つきのカード定義の一覧を API の応答の型に変換します。
func ToCardsWithOwnership(cardsWithOwnership []domain.CardWithOwnership) ([]*apicard.CardWithOwnership, error) {
	result := make([]*apicard.CardWithOwnership, len(cardsWithOwnership))
	for i, cardWithOwnership := range cardsWithOwnership {
		card := cardWithOwnership.Card
		stats, err := toStatsBody(card.Stats)
		if err != nil {
			return nil, err
		}
		effects, err := toEffectsBody(card.Effects)
		if err != nil {
			return nil, err
		}
		result[i] = &apicard.CardWithOwnership{
			CardID:        card.CardID,
			CardName:      card.CardName,
			ResourceLabel: card.ResourceLabel,
			Faction:       card.Faction,
			CardType:      card.CardType,
			Subtype:       card.Subtype,
			Resizable:     card.IsResizable,
			Elastic:       card.IsElastic,
			Stats:         stats,
			EffectText:    card.EffectText,
			Effects:       effects,
			Restriction:   card.Restriction,
			IsActive:      card.IsActive,
			CreatedAt:     card.CreatedAt,
			UpdatedAt:     card.UpdatedAt,
			IsOwned:       cardWithOwnership.IsOwned,
		}
	}
	return result, nil
}

// ToPlayerCardsWithDefinition はカード定義つきの所持カードの一覧を API の応答の型に変換します。
func ToPlayerCardsWithDefinition(playerCards []domain.PlayerCardWithDefinition) ([]*apicard.PlayerCardWithDef, error) {
	result := make([]*apicard.PlayerCardWithDef, len(playerCards))
	for i, playerCard := range playerCards {
		stats, err := toStatsBody(playerCard.Card.Stats)
		if err != nil {
			return nil, err
		}
		result[i] = &apicard.PlayerCardWithDef{
			CardID:        playerCard.PlayerCard.CardID,
			ArtNo:         playerCard.PlayerCard.ArtNo,
			Count:         playerCard.PlayerCard.Count,
			CardName:      playerCard.Card.CardName,
			ResourceLabel: playerCard.Card.ResourceLabel,
			Faction:       playerCard.Card.Faction,
			CardType:      playerCard.Card.CardType,
			Subtype:       playerCard.Card.Subtype,
			Resizable:     playerCard.Card.IsResizable,
			Elastic:       playerCard.Card.IsElastic,
			Stats:         stats,
			EffectText:    playerCard.Card.EffectText,
			Restriction:   playerCard.Card.Restriction,
		}
	}
	return result, nil
}

func toStatsBody(stats domain.CardStats) (json.RawMessage, error) {
	body, err := json.Marshal(cardStatsBody(stats))
	if err != nil {
		return nil, fmt.Errorf("marshal card stats: %w", err)
	}
	return body, nil
}

// toEffectsBody は効果定義を応答の形式に変換します。効果定義が未設定のときは nil を返し、応答に含めません。
func toEffectsBody(effects []map[string]any) (*json.RawMessage, error) {
	if effects == nil {
		return nil, nil
	}
	body, err := json.Marshal(effects)
	if err != nil {
		return nil, fmt.Errorf("marshal card effects: %w", err)
	}
	raw := json.RawMessage(body)
	return &raw, nil
}
