package service

import (
	"context"
	"fmt"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// CardService はカードマスターの読み取り操作を提供します。
type CardService struct {
	cardRepo       port.CardRepo
	playerCardRepo port.PlayerCardRepo
}

// NewCardService は CardService を生成します。
func NewCardService(cardRepo port.CardRepo, playerCardRepo port.PlayerCardRepo) *CardService {
	return &CardService{cardRepo: cardRepo, playerCardRepo: playerCardRepo}
}

// CardWithOwnership はカード定義にプレイヤーの所持状態を付与した型です。
type CardWithOwnership struct {
	*apicard.CardDefinition
	IsOwned bool `json:"is_owned"`
}

// GetAllCards は全カードにプレイヤーの所持状態を付与して返します。
func (s *CardService) GetAllCards(ctx context.Context, playerID string) ([]*CardWithOwnership, error) {
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

	result := make([]*CardWithOwnership, len(cards))
	for i, card := range cards {
		result[i] = &CardWithOwnership{
			CardDefinition: card,
			IsOwned:        owned[card.CardID],
		}
	}
	return result, nil
}

// FindAllRaw はカードマスター全件を返します。
// battle / gateway が起動時にインメモリキャッシュを構築するために使用します。
func (s *CardService) FindAllRaw(ctx context.Context) ([]*apicard.CardDefinition, error) {
	cards, err := s.cardRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("find all raw cards: %w", err)
	}
	return cards, nil
}
