package usecase

import (
	"context"
	"fmt"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// CardInteractor はカードマスターの読み取り操作を提供します。
type CardInteractor struct {
	cardRepository       port.CardRepository
	playerCardRepository port.PlayerCardRepository
}

// NewCardInteractor は CardInteractor を生成します。
func NewCardInteractor(cardRepository port.CardRepository, playerCardRepository port.PlayerCardRepository) *CardInteractor {
	return &CardInteractor{cardRepository: cardRepository, playerCardRepository: playerCardRepository}
}

// ListCardsWithOwnership は全カードにプレイヤーの所持状態を付与して返します。
func (ci *CardInteractor) ListCardsWithOwnership(ctx context.Context, playerID string) ([]domain.CardWithOwnership, error) {
	cards, err := ci.cardRepository.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("find all cards: %w", err)
	}

	ownedCards, err := ci.playerCardRepository.GetPlayerCards(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get player cards: %w", err)
	}

	ownedCardIDs := make(map[string]bool, len(ownedCards))
	for _, ownedCard := range ownedCards {
		ownedCardIDs[ownedCard.CardID] = true
	}

	cardsWithOwnership := make([]domain.CardWithOwnership, len(cards))
	for i, card := range cards {
		cardsWithOwnership[i] = domain.CardWithOwnership{Card: card, IsOwned: ownedCardIDs[card.CardID]}
	}
	return cardsWithOwnership, nil
}
