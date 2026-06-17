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
	gamedesign "github.com/kenyamaneko/overload-party-common/packages/game-design-constants"
)

// ErrNotFound は port.ErrNotFound の re-export です。
var ErrNotFound = port.ErrNotFound

// DeckInteractor はデッキの CRUD とバリデーションを提供します。
type DeckInteractor struct {
	deckRepo        port.DeckRepo
	playerCardRepo  port.PlayerCardRepo
	cardCache       *cache.CardCache
	productCache    *cache.ProductCache
	initiativeCache *cache.InitiativeCache
	factionClient   port.FactionClient
}

// NewDeckInteractor は DeckInteractor を生成します。
func NewDeckInteractor(deckRepo port.DeckRepo, playerCardRepo port.PlayerCardRepo, cardCache *cache.CardCache, productCache *cache.ProductCache, initiativeCache *cache.InitiativeCache, factionClient port.FactionClient) *DeckInteractor {
	return &DeckInteractor{deckRepo: deckRepo, playerCardRepo: playerCardRepo, cardCache: cardCache, productCache: productCache, initiativeCache: initiativeCache, factionClient: factionClient}
}

// validateFactionOwnership は宣言陣営をプレイヤーが所持しているか account に照会して検証します。
func (s *DeckInteractor) validateFactionOwnership(ctx context.Context, playerID, faction string) error {
	owned, err := s.factionClient.ListPlayerFactions(ctx, playerID)
	if err != nil {
		return fmt.Errorf("check faction ownership: %w", err)
	}
	for _, f := range owned {
		if f == faction {
			return nil
		}
	}
	return fmt.Errorf("%w: faction %q is not owned by player", port.ErrInvalidDeck, faction)
}

// CreateDeck は新しいデッキを作成します。所持カードと制限をバリデーションします。
func (s *DeckInteractor) CreateDeck(ctx context.Context, playerID string, req apicard.DeckCreateRequest) (*apicard.Deck, error) {
	ownedCards, err := s.playerCardRepo.GetPlayerCards(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get owned cards: %w", err)
	}

	deckCardEntries := presenter.DeckCardEntriesFromRequest(req.Cards)
	if err := s.validateDeckCards(req.Faction, deckCardEntries, ownedCards); err != nil {
		return nil, err
	}
	if err := s.validateFactionOwnership(ctx, playerID, req.Faction); err != nil {
		return nil, err
	}
	if err := s.validateInitiatives(req.Faction, req.ProductID, req.RoutineID, req.SpecialID); err != nil {
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
		Faction:   req.Faction,
		ProductID: req.ProductID,
		RoutineID: req.RoutineID,
		SpecialID: req.SpecialID,
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
		result[i] = presenter.ToDeck(d, cards, s.computeIsValid(d, cards, ownedCards))
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
	if err := s.validateDeckCards(req.Faction, deckCardEntries, ownedCards); err != nil {
		return nil, err
	}
	if err := s.validateFactionOwnership(ctx, playerID, req.Faction); err != nil {
		return nil, err
	}
	if err := s.validateInitiatives(req.Faction, req.ProductID, req.RoutineID, req.SpecialID); err != nil {
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
		Faction:   req.Faction,
		ProductID: req.ProductID,
		RoutineID: req.RoutineID,
		SpecialID: req.SpecialID,
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
// DeckSize 枚ちょうど・全カード所持・制限枚数以内・宣言陣営との整合を確認します。
func (s *DeckInteractor) ValidateDeckForBattle(ctx context.Context, playerID string, deckID int64) error {
	deck, err := s.deckRepo.FindByID(ctx, playerID, deckID)
	if err != nil {
		return fmt.Errorf("find deck: %w", err)
	}

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

	if err := s.validateDeckCards(deck.Faction, deckCardEntries, ownedCards); err != nil {
		return err
	}
	return s.validateInitiatives(deck.Faction, deck.ProductID, deck.RoutineID, deck.SpecialID)
}

// validateInitiatives はセットした施策が選択プロダクト・宣言陣営と整合するか検証します。
func (s *DeckInteractor) validateInitiatives(faction, productID, routineID, specialID string) error {
	product := s.productCache.FindByID(productID)
	if product == nil {
		return fmt.Errorf("%w: product %q not found", port.ErrInvalidDeck, productID)
	}
	if product.Faction != faction {
		return fmt.Errorf("%w: product %q belongs to faction %q, not deck faction %q",
			port.ErrInvalidDeck, productID, product.Faction, faction)
	}
	if err := s.validateInitiative(productID, routineID, domain.InitiativeKindRoutine); err != nil {
		return err
	}
	return s.validateInitiative(productID, specialID, domain.InitiativeKindSpecial)
}

// validateInitiative は施策が指定プロダクトに属し、かつ指定区分かを検証します。
func (s *DeckInteractor) validateInitiative(productID, initiativeID, kind string) error {
	initiative := s.initiativeCache.FindByID(initiativeID)
	if initiative == nil || initiative.ProductID != productID || initiative.Kind != kind {
		return fmt.Errorf("%w: %q is not a %s of product %q", port.ErrInvalidDeck, initiativeID, kind, productID)
	}
	return nil
}

func (s *DeckInteractor) validateDeckCards(faction string, deckCardEntries []domain.DeckCardEntry, ownedCards []*domain.PlayerCard) error {
	type ownedKey struct {
		cardID string
		artNo  int64
	}

	if !isSelectableFaction(faction) {
		return fmt.Errorf("%w: faction %q is not selectable", port.ErrInvalidDeck, faction)
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
		if card.Faction != faction && card.Faction != gamedesign.FactionNeutral {
			return fmt.Errorf("%w: card %s (%s): deck faction is %s, only %s and Neutral cards are allowed",
				port.ErrInvalidDeck, cardID, card.Faction, faction, faction)
		}
		limit, ok := gamedesign.RestrictionCopyCount[card.Restriction]
		if !ok {
			return fmt.Errorf("card %s: unknown restriction %q", cardID, card.Restriction)
		}
		if total > limit {
			return fmt.Errorf("%w: card %s (%s): exceeds restriction limit (%d/%d)",
				port.ErrRestrictionExceeded, cardID, card.Restriction, total, limit)
		}
	}

	return nil
}

func (s *DeckInteractor) computeIsValid(deck *domain.Deck, deckCards []domain.DeckCard, ownedCards []*domain.PlayerCard) bool {
	totalCards := 0
	for _, dc := range deckCards {
		totalCards += dc.Count
	}
	if totalCards != constants.DeckSize {
		return false
	}

	deckCardEntries := domain.DeckCardEntriesFromCards(deckCards)
	if s.validateDeckCards(deck.Faction, deckCardEntries, ownedCards) != nil {
		return false
	}
	return s.validateInitiatives(deck.Faction, deck.ProductID, deck.RoutineID, deck.SpecialID) == nil
}

// isSelectableFaction は faction がプレイヤーの選択可能な陣営かを判定します。
func isSelectableFaction(faction string) bool {
	for _, f := range gamedesign.SelectableFactions {
		if f == faction {
			return true
		}
	}
	return false
}
