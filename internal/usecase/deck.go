package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/constants"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/presenter"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

// ErrNotFound は port.ErrNotFound の re-export です。
var ErrNotFound = port.ErrNotFound

// DeckInteractor はデッキの CRUD とバリデーションを提供します。
type DeckInteractor struct {
	deckRepo       port.DeckRepo
	playerCardRepo port.PlayerCardRepo
	cardCache      *cache.CardCache
}

// NewDeckInteractor は DeckInteractor を生成します。
func NewDeckInteractor(deckRepo port.DeckRepo, playerCardRepo port.PlayerCardRepo, cardCache *cache.CardCache) *DeckInteractor {
	return &DeckInteractor{deckRepo: deckRepo, playerCardRepo: playerCardRepo, cardCache: cardCache}
}

// CreateDeck は新しいデッキを作成します。所持カードと制限をバリデーションします。
func (s *DeckInteractor) CreateDeck(ctx context.Context, playerID string, req apicard.DeckCreateRequest) (*apicard.Deck, error) {
	ownedCards, err := s.playerCardRepo.GetPlayerCards(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get owned cards: %w", err)
	}

	deckCardEntries := presenter.DeckCardEntriesFromRequest(req.Cards)
	if err := s.validateDeckCards(deckCardEntries, ownedCards); err != nil {
		return nil, err
	}

	totalCards := 0
	for _, e := range deckCardEntries {
		totalCards += e.Count
	}

	now := time.Now()
	deck := domain.Deck{
		PlayerID:  playerID,
		DeckName:  req.DeckName,
		PlaymatNo: req.PlaymatNo,
		SleeveNo:  req.SleeveNo,
		CreatedAt: now,
		UpdatedAt: now,
	}

	deckID, err := s.deckRepo.Create(ctx, deck, deckCardEntries)
	if err != nil {
		return nil, fmt.Errorf("create deck: %w", err)
	}
	deck.DeckID = deckID

	cards, err := s.deckRepo.GetDeckCards(ctx, deck.PlayerID, deck.DeckID)
	if err != nil {
		return nil, fmt.Errorf("get deck cards for deck %d: %w", deck.DeckID, err)
	}

	return presenter.ToDeck(&deck, cards, totalCards == constants.DeckSize), nil
}

// GetDecks はプレイヤーの全デッキを is_valid 付きで返します。
func (s *DeckInteractor) GetDecks(ctx context.Context, playerID string) ([]*apicard.Deck, error) {
	decks, err := s.deckRepo.FindByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}

	ownedCards, err := s.playerCardRepo.GetPlayerCards(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get owned cards: %w", err)
	}

	result := make([]*apicard.Deck, len(decks))
	for i, d := range decks {
		cards, err := s.deckRepo.GetDeckCards(ctx, d.PlayerID, d.DeckID)
		if err != nil {
			return nil, fmt.Errorf("get deck cards for deck %d: %w", d.DeckID, err)
		}
		result[i] = presenter.ToDeck(d, cards, s.computeIsValid(cards, ownedCards))
	}
	return result, nil
}

// GetDeck は指定デッキとそのカード構成を返します。
func (s *DeckInteractor) GetDeck(ctx context.Context, playerID string, deckID int64) (*apicard.Deck, []apicard.DeckCard, error) {
	deck, err := s.deckRepo.FindByID(ctx, playerID, deckID)
	if err != nil {
		return nil, nil, fmt.Errorf("find deck: %w", err)
	}

	cards, err := s.deckRepo.GetDeckCards(ctx, playerID, deckID)
	if err != nil {
		return nil, nil, fmt.Errorf("get deck cards: %w", err)
	}

	apiCards := presenter.ToDeckCards(cards)
	apiDeck := presenter.ToDeck(deck, cards, false)
	apiDeck.DeckCards = &apiCards
	return apiDeck, apiCards, nil
}

// UpdateDeck は既存デッキを更新します。所持カードと制限をバリデーションします。
func (s *DeckInteractor) UpdateDeck(ctx context.Context, playerID string, deckID int64, req apicard.DeckUpdateRequest) (*apicard.Deck, error) {
	ownedCards, err := s.playerCardRepo.GetPlayerCards(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get owned cards: %w", err)
	}

	deckCardEntries := presenter.DeckCardEntriesFromRequest(req.Cards)
	if err := s.validateDeckCards(deckCardEntries, ownedCards); err != nil {
		return nil, err
	}

	totalCards := 0
	for _, e := range deckCardEntries {
		totalCards += e.Count
	}

	deck := domain.Deck{
		PlayerID:  playerID,
		DeckID:    deckID,
		DeckName:  req.DeckName,
		PlaymatNo: req.PlaymatNo,
		SleeveNo:  req.SleeveNo,
		UpdatedAt: time.Now(),
	}

	if err := s.deckRepo.Update(ctx, deck, deckCardEntries); err != nil {
		return nil, fmt.Errorf("update deck: %w", err)
	}

	cards, err := s.deckRepo.GetDeckCards(ctx, deck.PlayerID, deck.DeckID)
	if err != nil {
		return nil, fmt.Errorf("get deck cards for deck %d: %w", deck.DeckID, err)
	}
	return presenter.ToDeck(&deck, cards, totalCards == constants.DeckSize), nil
}

// DeleteDeck は指定デッキを削除します。
func (s *DeckInteractor) DeleteDeck(ctx context.Context, playerID string, deckID int64) error {
	return s.deckRepo.Delete(ctx, playerID, deckID)
}

// ValidateDeckForBattle はデッキがバトル可能かを検証します。
// DeckSize 枚ちょうど・全カード所持・制限枚数以内を確認します。
func (s *DeckInteractor) ValidateDeckForBattle(ctx context.Context, playerID string, deckID int64) error {
	deckCards, err := s.deckRepo.GetDeckCards(ctx, playerID, deckID)
	if err != nil {
		return fmt.Errorf("get deck cards: %w", err)
	}

	deckCardEntries := domain.DeckCardEntriesFromCards(deckCards)

	totalCards := 0
	for _, e := range deckCardEntries {
		totalCards += e.Count
	}
	if totalCards != constants.DeckSize {
		return fmt.Errorf("%w: deck has %d cards, need exactly %d", port.ErrInvalidDeck, totalCards, constants.DeckSize)
	}

	ownedCards, err := s.playerCardRepo.GetPlayerCards(ctx, playerID)
	if err != nil {
		return fmt.Errorf("get owned cards: %w", err)
	}

	return s.validateDeckCards(deckCardEntries, ownedCards)
}

func (s *DeckInteractor) validateDeckCards(deckCardEntries []domain.DeckCardEntry, ownedCards []*domain.PlayerCard) error {
	type ownedKey struct {
		cardID string
		artNo  int64
	}

	totalCards := 0
	for _, e := range deckCardEntries {
		if e.Count <= 0 {
			return fmt.Errorf("%w: card %s variant %d: count must be positive", port.ErrInvalidDeck, e.CardID, e.ArtNo)
		}
		totalCards += e.Count
	}
	if totalCards > constants.DeckSize {
		return fmt.Errorf("%w: deck cannot exceed %d cards", port.ErrInvalidDeck, constants.DeckSize)
	}

	owned := make(map[ownedKey]int, len(ownedCards))
	for _, c := range ownedCards {
		owned[ownedKey{c.CardID, c.ArtNo}] = c.Count
	}

	for _, e := range deckCardEntries {
		key := ownedKey{e.CardID, e.ArtNo}
		if owned[key] < e.Count {
			return fmt.Errorf("%w: card %s variant %d: not enough owned (need %d, have %d)",
				port.ErrUnowned, e.CardID, e.ArtNo, e.Count, owned[key])
		}
	}

	cardIDTotals := make(map[string]int)
	for _, e := range deckCardEntries {
		cardIDTotals[e.CardID] += e.Count
	}
	for cardID, total := range cardIDTotals {
		card := s.cardCache.Get(cardID)
		if card == nil {
			return fmt.Errorf("%w: card %s not found in card definitions", port.ErrInvalidDeck, cardID)
		}
		limit, err := constants.RestrictionCopyCount(card.Restriction)
		if err != nil {
			return fmt.Errorf("card %s: %w", cardID, err)
		}
		if total > limit {
			return fmt.Errorf("%w: card %s (%s): exceeds restriction limit (%d/%d)",
				port.ErrRestrictionExceeded, cardID, card.Restriction, total, limit)
		}
	}

	return nil
}

func (s *DeckInteractor) computeIsValid(deckCards []domain.DeckCard, ownedCards []*domain.PlayerCard) bool {
	totalCards := 0
	for _, dc := range deckCards {
		totalCards += dc.Count
	}
	if totalCards != constants.DeckSize {
		return false
	}

	deckCardEntries := domain.DeckCardEntriesFromCards(deckCards)
	return s.validateDeckCards(deckCardEntries, ownedCards) == nil
}
