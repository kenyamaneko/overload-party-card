package presenter

import (
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

// ToDeck は domain.Deck と構成カード・有効性フラグから wire の Deck を組み立てます。
// isValid は usecase 側で算出された派生値です。
func ToDeck(d *domain.Deck, cards []domain.DeckCard, isValid bool) *apicard.Deck {
	return &apicard.Deck{
		PlayerID:  d.PlayerID,
		DeckID:    d.DeckID,
		DeckName:  d.DeckName,
		IsValid:   isValid,
		PlaymatNo: d.PlaymatNo,
		SleeveNo:  d.SleeveNo,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
		DeckCards: ToDeckCards(cards),
	}
}

// ToDeckCards は domain.DeckCard slice を wire の DeckCard slice に詰め替えます。
func ToDeckCards(cards []domain.DeckCard) []apicard.DeckCard {
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

// DeckCardEntriesFromRequest はデッキ作成・更新リクエストの wire 入力を
// domain.DeckCardEntry slice に変換します。
func DeckCardEntriesFromRequest(reqEntries []apicard.DeckCardEntry) []domain.DeckCardEntry {
	result := make([]domain.DeckCardEntry, len(reqEntries))
	for i, e := range reqEntries {
		result[i] = domain.DeckCardEntry{
			CardID: e.CardID,
			ArtNo:  e.ArtNo,
			Count:  e.Count,
		}
	}
	return result
}
