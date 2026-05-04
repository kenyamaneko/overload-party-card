package usecase

import (
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

// usecase は domain ↔ apicard の境界に立つ層なので、translation は usecase 内に閉じる。
// repository / cache 側は domain 型のみ扱い、handler 側は apicard 型のみ扱う。

func cardToAPI(c *domain.Card) *apicard.CardDefinition {
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

func cardsToAPI(cards []*domain.Card) []*apicard.CardDefinition {
	result := make([]*apicard.CardDefinition, len(cards))
	for i, c := range cards {
		result[i] = cardToAPI(c)
	}
	return result
}

func deckToAPI(d *domain.Deck, cards []domain.DeckCard, isValid bool) *apicard.Deck {
	return &apicard.Deck{
		PlayerID:  d.PlayerID,
		DeckID:    d.DeckID,
		DeckName:  d.DeckName,
		IsValid:   isValid,
		PlaymatNo: d.PlaymatNo,
		SleeveNo:  d.SleeveNo,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
		DeckCards: deckCardsToAPI(cards),
	}
}

func deckCardsToAPI(cards []domain.DeckCard) []apicard.DeckCard {
	result := make([]apicard.DeckCard, len(cards))
	for i, c := range cards {
		result[i] = apicard.DeckCard{
			PlayerID: c.PlayerID,
			DeckID:   c.DeckID,
			CardID:   c.CardID,
			ArtNo:    c.ArtNo,
			Count:    c.Count,
		}
	}
	return result
}

func entriesFromAPI(entries []apicard.DeckCardEntry) []domain.DeckCardEntry {
	result := make([]domain.DeckCardEntry, len(entries))
	for i, e := range entries {
		result[i] = domain.DeckCardEntry{
			CardID: e.CardID,
			ArtNo:  e.ArtNo,
			Count:  e.Count,
		}
	}
	return result
}

func entriesFromDeckCards(cards []domain.DeckCard) []domain.DeckCardEntry {
	result := make([]domain.DeckCardEntry, len(cards))
	for i, c := range cards {
		result[i] = domain.DeckCardEntry{
			CardID: c.CardID,
			ArtNo:  c.ArtNo,
			Count:  c.Count,
		}
	}
	return result
}
