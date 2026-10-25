package presenter

import (
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

// ToDeck はデッキを API の応答の型に変換します。カード構成が空のときは、応答にカード構成の項目を含めません。
func ToDeck(detail domain.DeckDetail) *apicard.Deck {
	deck := toDeckWithoutCards(detail)
	if len(detail.Cards) > 0 {
		deckCards := ToDeckCards(detail.Cards)
		deck.DeckCards = &deckCards
	}
	return deck
}

// ToDeckWithCards はデッキを API の応答の型に変換します。カード構成が空のときも、応答にカード構成の項目を含めます。
func ToDeckWithCards(detail domain.DeckDetail) *apicard.Deck {
	deck := toDeckWithoutCards(detail)
	deckCards := ToDeckCards(detail.Cards)
	deck.DeckCards = &deckCards
	return deck
}

// ToDecks はデッキの一覧を API の応答の型に変換します。
func ToDecks(details []domain.DeckDetail) []*apicard.Deck {
	decks := make([]*apicard.Deck, len(details))
	for i, detail := range details {
		decks[i] = ToDeck(detail)
	}
	return decks
}

// ToDeckCards はデッキのカード構成を API の応答の型に変換します。
func ToDeckCards(cards []domain.DeckCard) []apicard.DeckCard {
	result := make([]apicard.DeckCard, len(cards))
	for i, card := range cards {
		result[i] = apicard.DeckCard{
			PlayerID: card.PlayerID,
			DeckID:   card.DeckID,
			CardID:   card.CardID,
			ArtNo:    card.ArtNo,
			Count:    card.Count,
		}
	}
	return result
}

// ToDeckDraftFromCreateRequest はデッキ作成リクエストを、デッキの作成・更新で指定する内容に変換します。
func ToDeckDraftFromCreateRequest(request apicard.DeckCreateRequest) domain.DeckDraft {
	return domain.DeckDraft{
		DeckName:  request.DeckName,
		Faction:   request.Faction,
		ProductID: request.ProductID,
		RoutineID: request.RoutineID,
		SpecialID: request.SpecialID,
		PlaymatNo: request.PlaymatNo,
		SleeveNo:  request.SleeveNo,
		Cards:     toDeckCardEntries(request.Cards),
	}
}

// ToDeckDraftFromUpdateRequest はデッキ更新リクエストを、デッキの作成・更新で指定する内容に変換します。
func ToDeckDraftFromUpdateRequest(request apicard.DeckUpdateRequest) domain.DeckDraft {
	return ToDeckDraftFromCreateRequest(apicard.DeckCreateRequest(request))
}

func toDeckWithoutCards(detail domain.DeckDetail) *apicard.Deck {
	return &apicard.Deck{
		PlayerID:  detail.Deck.PlayerID,
		DeckID:    detail.Deck.DeckID,
		DeckName:  detail.Deck.DeckName,
		Faction:   detail.Deck.Faction,
		ProductID: detail.Deck.ProductID,
		RoutineID: detail.Deck.RoutineID,
		SpecialID: detail.Deck.SpecialID,
		IsValid:   detail.IsValid,
		PlaymatNo: detail.Deck.PlaymatNo,
		SleeveNo:  detail.Deck.SleeveNo,
		CreatedAt: detail.Deck.CreatedAt,
		UpdatedAt: detail.Deck.UpdatedAt,
	}
}

func toDeckCardEntries(entries []apicard.DeckCardEntry) []domain.DeckCardEntry {
	result := make([]domain.DeckCardEntry, len(entries))
	for i, entry := range entries {
		result[i] = domain.DeckCardEntry{
			CardID: entry.CardID,
			ArtNo:  entry.ArtNo,
			Count:  entry.Count,
		}
	}
	return result
}
