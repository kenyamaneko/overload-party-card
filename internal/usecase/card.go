package usecase

import (
	"context"
	"fmt"

	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/presenter"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

// CardInteractor はカードマスターの読み取り操作を提供します。
type CardInteractor struct {
	cardRepo       port.CardRepo
	playerCardRepo port.PlayerCardRepo
}

// NewCardInteractor は CardInteractor を生成します。
func NewCardInteractor(cardRepo port.CardRepo, playerCardRepo port.PlayerCardRepo) *CardInteractor {
	return &CardInteractor{cardRepo: cardRepo, playerCardRepo: playerCardRepo}
}

// ListCardsWithOwnership は全カードにプレイヤーの所持状態を付与して返します。
func (s *CardInteractor) ListCardsWithOwnership(ctx context.Context, playerID string) ([]*apicard.CardWithOwnership, error) {
	cards, err := s.cardRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all cards: %w", err)
	}

	playerCards, err := s.playerCardRepo.GetPlayerCards(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get player cards: %w", err)
	}

	owned := make(map[string]bool, len(playerCards))
	for _, pc := range playerCards {
		owned[pc.CardID] = true
	}

	result := make([]*apicard.CardWithOwnership, len(cards))
	for i, card := range cards {
		result[i] = presenter.ToCardWithOwnership(card, owned[card.CardID])
	}
	return result, nil
}

// ListCards はカードマスター全件を返します。
// battle / gateway が起動時にインメモリキャッシュを構築するために使用します。
func (s *CardInteractor) ListCards(ctx context.Context) ([]*apicard.CardDefinition, error) {
	cards, err := s.cardRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("find all raw cards: %w", err)
	}
	return presenter.ToCardDefinitions(cards), nil
}
