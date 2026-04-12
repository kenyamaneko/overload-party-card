package service

import (
	"context"
	"fmt"
	"time"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/constants"
	"github.com/kenyamaneko/overload-party-card/internal/model"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// ErrNotFound は port.ErrNotFound の re-export です。
var ErrNotFound = port.ErrNotFound

// DeckService はデッキの CRUD とバリデーションを提供します。
type DeckService struct {
	deckRepo       port.DeckRepo
	playerCardRepo port.PlayerCardRepo
	cardCache      *cache.CardCache
}

// NewDeckService は DeckService を生成します。
func NewDeckService(deckRepo port.DeckRepo, playerCardRepo port.PlayerCardRepo, cardCache *cache.CardCache) *DeckService {
	return &DeckService{deckRepo: deckRepo, playerCardRepo: playerCardRepo, cardCache: cardCache}
}

// GetDecks はプレイヤーの全デッキを is_valid 付きで返します。
func (s *DeckService) GetDecks(ctx context.Context, playerID string) ([]*apicard.Deck, error) {
	decks, err := s.deckRepo.FindByPlayerID(ctx, playerID)
	if err != nil {
		return nil, err
	}

	ownedCards, err := s.playerCardRepo.GetPlayerCards(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get owned cards: %w", err)
	}

	for _, d := range decks {
		if err := s.populateDeckCards(ctx, d); err != nil {
			return nil, err
		}
		d.IsValid = s.computeIsValid(d.DeckCards, ownedCards)
	}
	return decks, nil
}

// GetDeck は指定デッキとそのカード構成を返します。
func (s *DeckService) GetDeck(ctx context.Context, playerID string, deckID int64) (*apicard.Deck, []apicard.DeckCard, error) {
	deck, err := s.deckRepo.FindByID(ctx, playerID, deckID)
	if err != nil {
		return nil, nil, fmt.Errorf("find deck: %w", err)
	}

	cards, err := s.deckRepo.GetDeckCards(ctx, playerID, deckID)
	if err != nil {
		return nil, nil, fmt.Errorf("get deck cards: %w", err)
	}

	return deck, cards, nil
}

// CreateDeckRequest は DeckCreateRequest の型エイリアスです。
type CreateDeckRequest = apicard.DeckCreateRequest

// CreateDeck は新しいデッキを作成します。所持カードと制限をバリデーションします。
func (s *DeckService) CreateDeck(ctx context.Context, playerID string, req CreateDeckRequest) (*apicard.Deck, error) {
	ownedCards, err := s.playerCardRepo.GetPlayerCards(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get owned cards: %w", err)
	}

	if err := s.validateDeckCards(req.Cards, ownedCards); err != nil {
		return nil, err
	}

	totalCards := 0
	for _, c := range req.Cards {
		totalCards += c.Count
	}

	now := time.Now()
	deck := &apicard.Deck{
		PlayerID:  playerID,
		DeckName:  req.DeckName,
		IsValid:   totalCards == constants.DeckSize,
		PlaymatNo: req.PlaymatNo,
		SleeveNo:  req.SleeveNo,
		CreatedAt: now,
		UpdatedAt: now,
	}

	var deckCards []apicard.DeckCard
	for _, entry := range req.Cards {
		deckCards = append(deckCards, apicard.DeckCard{
			PlayerID: playerID,
			CardID:   entry.CardID,
			ArtNo:    entry.ArtNo,
			Count:    entry.Count,
		})
	}

	if err := s.deckRepo.Create(ctx, deck, deckCards); err != nil {
		return nil, fmt.Errorf("create deck: %w", err)
	}

	if err := s.populateDeckCards(ctx, deck); err != nil {
		return nil, err
	}
	return deck, nil
}

// UpdateDeckRequest は DeckUpdateRequest の型エイリアスです。
type UpdateDeckRequest = apicard.DeckUpdateRequest

// UpdateDeck は既存デッキを更新します。所持カードと制限をバリデーションします。
func (s *DeckService) UpdateDeck(ctx context.Context, playerID string, deckID int64, req UpdateDeckRequest) (*apicard.Deck, error) {
	ownedCards, err := s.playerCardRepo.GetPlayerCards(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get owned cards: %w", err)
	}

	if err := s.validateDeckCards(req.Cards, ownedCards); err != nil {
		return nil, err
	}

	totalCards := 0
	for _, c := range req.Cards {
		totalCards += c.Count
	}

	deck := &apicard.Deck{
		PlayerID:  playerID,
		DeckID:    deckID,
		DeckName:  req.DeckName,
		IsValid:   totalCards == constants.DeckSize,
		PlaymatNo: req.PlaymatNo,
		SleeveNo:  req.SleeveNo,
		UpdatedAt: time.Now(),
	}

	var deckCards []apicard.DeckCard
	for _, entry := range req.Cards {
		deckCards = append(deckCards, apicard.DeckCard{
			PlayerID: playerID,
			DeckID:   deckID,
			CardID:   entry.CardID,
			ArtNo:    entry.ArtNo,
			Count:    entry.Count,
		})
	}

	if err := s.deckRepo.Update(ctx, deck, deckCards); err != nil {
		return nil, fmt.Errorf("update deck: %w", err)
	}

	if err := s.populateDeckCards(ctx, deck); err != nil {
		return nil, err
	}
	return deck, nil
}

// ValidateDeckForBattle はデッキがバトル可能かを検証します。
// DeckSize 枚ちょうど・全カード所持・制限枚数以内を確認します。
func (s *DeckService) ValidateDeckForBattle(ctx context.Context, playerID string, deckID int64) error {
	deckCards, err := s.deckRepo.GetDeckCards(ctx, playerID, deckID)
	if err != nil {
		return fmt.Errorf("get deck cards: %w", err)
	}

	entries := make([]apicard.DeckCardEntry, len(deckCards))
	for i, dc := range deckCards {
		entries[i] = apicard.DeckCardEntry{CardID: dc.CardID, ArtNo: dc.ArtNo, Count: dc.Count}
	}

	totalCards := 0
	for _, e := range entries {
		totalCards += e.Count
	}
	if totalCards != constants.DeckSize {
		return fmt.Errorf("%w: deck has %d cards, need exactly %d", port.ErrInvalidDeck, totalCards, constants.DeckSize)
	}

	ownedCards, err := s.playerCardRepo.GetPlayerCards(ctx, playerID)
	if err != nil {
		return fmt.Errorf("get owned cards: %w", err)
	}

	return s.validateDeckCards(entries, ownedCards)
}

// DeleteDeck は指定デッキを削除します。
func (s *DeckService) DeleteDeck(ctx context.Context, playerID string, deckID int64) error {
	return s.deckRepo.Delete(ctx, playerID, deckID)
}

func (s *DeckService) populateDeckCards(ctx context.Context, deck *apicard.Deck) error {
	cards, err := s.deckRepo.GetDeckCards(ctx, deck.PlayerID, deck.DeckID)
	if err != nil {
		return fmt.Errorf("get deck cards for deck %d: %w", deck.DeckID, err)
	}
	deck.DeckCards = cards
	return nil
}

// GetPlayerCards はプレイヤーの所持カードにカード定義を付与して返します。
func (s *DeckService) GetPlayerCards(ctx context.Context, playerID string) ([]*apicard.PlayerCardWithDef, error) {
	pcs, err := s.playerCardRepo.GetPlayerCards(ctx, playerID)
	if err != nil {
		return nil, err
	}

	result := make([]*apicard.PlayerCardWithDef, 0, len(pcs))
	for _, pc := range pcs {
		cd := s.cardCache.Get(pc.CardID)
		if cd == nil {
			return nil, fmt.Errorf("player %s owns card %s but it is missing from the card cache; refresh cache or investigate inconsistent state", playerID, pc.CardID)
		}
		result = append(result, &apicard.PlayerCardWithDef{
			CardID:      pc.CardID,
			ArtNo:       pc.ArtNo,
			Count:       pc.Count,
			CardName:    cd.CardName,
			Faction:     cd.Faction,
			CardType:    cd.CardType,
			Resizable:   cd.Resizable,
			Elastic:     cd.Elastic,
			Stats:       cd.Stats,
			EffectText:  cd.EffectText,
			Restriction: cd.Restriction,
		})
	}
	return result, nil
}

type ownedKey struct {
	cardID  string
	variant int64
}

func (s *DeckService) validateDeckCards(entries []apicard.DeckCardEntry, ownedCards []*model.PlayerCard) error {
	totalCards := 0
	for _, e := range entries {
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

	for _, e := range entries {
		key := ownedKey{e.CardID, e.ArtNo}
		if owned[key] < e.Count {
			return fmt.Errorf("%w: card %s variant %d: not enough owned (need %d, have %d)",
				port.ErrUnowned, e.CardID, e.ArtNo, e.Count, owned[key])
		}
	}

	cardIDTotals := make(map[string]int)
	for _, e := range entries {
		cardIDTotals[e.CardID] += e.Count
	}
	for cardID, total := range cardIDTotals {
		card := s.cardCache.Get(cardID)
		if card == nil {
			return fmt.Errorf("%w: card %s not found in card definitions", port.ErrInvalidDeck, cardID)
		}
		limit := constants.RestrictionCopyCount(card.Restriction)
		if total > limit {
			return fmt.Errorf("%w: card %s (%s): exceeds restriction limit (%d/%d)",
				port.ErrRestrictionExceeded, cardID, card.Restriction, total, limit)
		}
	}

	return nil
}

func (s *DeckService) computeIsValid(deckCards []apicard.DeckCard, ownedCards []*model.PlayerCard) bool {
	totalCards := 0
	for _, dc := range deckCards {
		totalCards += dc.Count
	}
	if totalCards != constants.DeckSize {
		return false
	}

	entries := make([]apicard.DeckCardEntry, len(deckCards))
	for i, dc := range deckCards {
		entries[i] = apicard.DeckCardEntry{CardID: dc.CardID, ArtNo: dc.ArtNo, Count: dc.Count}
	}
	return s.validateDeckCards(entries, ownedCards) == nil
}
