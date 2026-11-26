package usecase

import (
	"context"
	"fmt"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// PlayerCardInteractor はプレイヤーの所持カード照会を提供します。
type PlayerCardInteractor struct {
	playerCardRepository port.PlayerCardRepository
	cardCache            port.CardCache
}

// NewPlayerCardInteractor は PlayerCardInteractor を生成します。
func NewPlayerCardInteractor(playerCardRepository port.PlayerCardRepository, cardCache port.CardCache) *PlayerCardInteractor {
	return &PlayerCardInteractor{playerCardRepository: playerCardRepository, cardCache: cardCache}
}

// GetPlayerCards はプレイヤーの所持カードにカード定義を付与して返します。
// カード定義が存在しないカードを所持している状態は整合性の異常なので、黙ってスキップせずエラーにします。
func (pi *PlayerCardInteractor) GetPlayerCards(ctx context.Context, playerID string) ([]domain.PlayerCardWithDefinition, error) {
	ownedCards, err := pi.playerCardRepository.GetPlayerCards(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get player cards: %w", err)
	}

	cardsWithDefinition := make([]domain.PlayerCardWithDefinition, len(ownedCards))
	for i, ownedCard := range ownedCards {
		card := pi.cardCache.FindByID(ownedCard.CardID)
		if card == nil {
			return nil, fmt.Errorf("player %s owns card %s but it is missing from the card cache", playerID, ownedCard.CardID)
		}
		cardsWithDefinition[i] = domain.PlayerCardWithDefinition{PlayerCard: *ownedCard, Card: card}
	}
	return cardsWithDefinition, nil
}
