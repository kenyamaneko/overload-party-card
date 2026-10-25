package usecase

import (
	"context"
	"fmt"
	"slices"

	gamedesign "github.com/kenyamaneko/overload-party-common/packages/game-design-constants"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// DeckInteractor はデッキの作成・取得・更新・削除と、デッキ構成の検証を提供します。
type DeckInteractor struct {
	deckRepository       port.DeckRepository
	playerCardRepository port.PlayerCardRepository
	cardCache            port.CardCache
	productCache         port.ProductCache
	initiativeCache      port.InitiativeCache
	factionClient        port.FactionClient
	clock                port.Clock
}

// NewDeckInteractor は DeckInteractor を生成します。
func NewDeckInteractor(
	deckRepository port.DeckRepository,
	playerCardRepository port.PlayerCardRepository,
	cardCache port.CardCache,
	productCache port.ProductCache,
	initiativeCache port.InitiativeCache,
	factionClient port.FactionClient,
	clock port.Clock,
) *DeckInteractor {
	return &DeckInteractor{
		deckRepository:       deckRepository,
		playerCardRepository: playerCardRepository,
		cardCache:            cardCache,
		productCache:         productCache,
		initiativeCache:      initiativeCache,
		factionClient:        factionClient,
		clock:                clock,
	}
}

// CreateDeck は新しいデッキを作成します。所持カードと制限をバリデーションします。
func (di *DeckInteractor) CreateDeck(ctx context.Context, playerID string, draft domain.DeckDraft) (*domain.DeckDetail, error) {
	if err := di.validateDeckDraft(ctx, playerID, draft); err != nil {
		return nil, err
	}

	now := di.clock.Now()
	deck := newDeckFromDraft(playerID, draft)
	deck.CreatedAt = now
	deck.UpdatedAt = now

	deckID, err := di.deckRepository.Create(ctx, deck, draft.Cards)
	if err != nil {
		return nil, fmt.Errorf("create deck: %w", err)
	}
	deck.DeckID = deckID

	cards, err := di.deckRepository.GetDeckCards(ctx, deck.PlayerID, deck.DeckID)
	if err != nil {
		return nil, fmt.Errorf("get deck cards for deck %d: %w", deck.DeckID, err)
	}

	return &domain.DeckDetail{Deck: deck, Cards: cards, IsValid: countDeckCards(draft.Cards) == gamedesign.DeckSize}, nil
}

// GetDecks はプレイヤーの全デッキをバトルで使えるかの判定付きで返します。
func (di *DeckInteractor) GetDecks(ctx context.Context, playerID string) ([]domain.DeckDetail, error) {
	decks, err := di.deckRepository.FindByPlayerID(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("find decks: %w", err)
	}

	ownedCards, err := di.playerCardRepository.GetPlayerCards(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get owned cards: %w", err)
	}

	details := make([]domain.DeckDetail, len(decks))
	for i, deck := range decks {
		cards, err := di.deckRepository.GetDeckCards(ctx, deck.PlayerID, deck.DeckID)
		if err != nil {
			return nil, fmt.Errorf("get deck cards for deck %d: %w", deck.DeckID, err)
		}
		details[i] = domain.DeckDetail{Deck: *deck, Cards: cards, IsValid: di.isDeckValid(deck, cards, ownedCards)}
	}
	return details, nil
}

// GetDeck は指定デッキとそのカード構成を返します。
func (di *DeckInteractor) GetDeck(ctx context.Context, playerID string, deckID int64) (*domain.DeckDetail, error) {
	deck, err := di.deckRepository.FindByID(ctx, playerID, deckID)
	if err != nil {
		return nil, fmt.Errorf("find deck: %w", err)
	}

	cards, err := di.deckRepository.GetDeckCards(ctx, playerID, deckID)
	if err != nil {
		return nil, fmt.Errorf("get deck cards: %w", err)
	}

	ownedCards, err := di.playerCardRepository.GetPlayerCards(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("get owned cards: %w", err)
	}

	return &domain.DeckDetail{Deck: *deck, Cards: cards, IsValid: di.isDeckValid(deck, cards, ownedCards)}, nil
}

// UpdateDeck は既存デッキを更新します。所持カードと制限をバリデーションします。
func (di *DeckInteractor) UpdateDeck(ctx context.Context, playerID string, deckID int64, draft domain.DeckDraft) (*domain.DeckDetail, error) {
	if err := di.validateDeckDraft(ctx, playerID, draft); err != nil {
		return nil, err
	}

	deck := newDeckFromDraft(playerID, draft)
	deck.DeckID = deckID
	deck.UpdatedAt = di.clock.Now()

	if err := di.deckRepository.Update(ctx, deck, draft.Cards); err != nil {
		return nil, fmt.Errorf("update deck: %w", err)
	}

	updatedDeck, err := di.deckRepository.FindByID(ctx, playerID, deckID)
	if err != nil {
		return nil, fmt.Errorf("find updated deck %d: %w", deckID, err)
	}
	cards, err := di.deckRepository.GetDeckCards(ctx, playerID, deckID)
	if err != nil {
		return nil, fmt.Errorf("get deck cards for deck %d: %w", deckID, err)
	}
	return &domain.DeckDetail{Deck: *updatedDeck, Cards: cards, IsValid: countDeckCards(draft.Cards) == gamedesign.DeckSize}, nil
}

// DeleteDeck は指定デッキを削除します。
func (di *DeckInteractor) DeleteDeck(ctx context.Context, playerID string, deckID int64) error {
	return di.deckRepository.Delete(ctx, playerID, deckID)
}

// ValidateDeckForBattle はデッキがバトル可能かを検証します。
// 規定枚数ちょうど・全カード所持・制限枚数以内・宣言陣営との整合を確認します。
func (di *DeckInteractor) ValidateDeckForBattle(ctx context.Context, playerID string, deckID int64) error {
	deck, err := di.deckRepository.FindByID(ctx, playerID, deckID)
	if err != nil {
		return fmt.Errorf("find deck: %w", err)
	}

	deckCards, err := di.deckRepository.GetDeckCards(ctx, playerID, deckID)
	if err != nil {
		return fmt.Errorf("get deck cards: %w", err)
	}

	deckCardEntries := domain.ToDeckCardEntries(deckCards)
	totalCards := countDeckCards(deckCardEntries)
	if totalCards != gamedesign.DeckSize {
		return fmt.Errorf("%w: deck has %d cards, need exactly %d", port.ErrInvalidDeck, totalCards, gamedesign.DeckSize)
	}

	ownedCards, err := di.playerCardRepository.GetPlayerCards(ctx, playerID)
	if err != nil {
		return fmt.Errorf("get owned cards: %w", err)
	}

	if err := di.validateDeckCards(deck.Faction, deckCardEntries, ownedCards); err != nil {
		return err
	}
	return di.validateInitiatives(deck.Faction, deck.ProductID, deck.RoutineID, deck.SpecialID)
}

func newDeckFromDraft(playerID string, draft domain.DeckDraft) domain.Deck {
	return domain.Deck{
		PlayerID:  playerID,
		DeckName:  draft.DeckName,
		Faction:   draft.Faction,
		ProductID: draft.ProductID,
		RoutineID: draft.RoutineID,
		SpecialID: draft.SpecialID,
		PlaymatNo: draft.PlaymatNo,
		SleeveNo:  draft.SleeveNo,
	}
}

func countDeckCards(entries []domain.DeckCardEntry) int {
	total := 0
	for _, entry := range entries {
		total += entry.Count
	}
	return total
}

// validateDeckDraft はデッキの作成・更新で指定された内容を、所持カード・所持陣営・施策との整合まで検証します。
func (di *DeckInteractor) validateDeckDraft(ctx context.Context, playerID string, draft domain.DeckDraft) error {
	ownedCards, err := di.playerCardRepository.GetPlayerCards(ctx, playerID)
	if err != nil {
		return fmt.Errorf("get owned cards: %w", err)
	}

	if err := di.validateDeckCards(draft.Faction, draft.Cards, ownedCards); err != nil {
		return err
	}
	if err := di.validateFactionOwnership(ctx, playerID, draft.Faction); err != nil {
		return err
	}
	return di.validateInitiatives(draft.Faction, draft.ProductID, draft.RoutineID, draft.SpecialID)
}

// validateFactionOwnership は宣言陣営をプレイヤーが所持しているか account に照会して検証します。
func (di *DeckInteractor) validateFactionOwnership(ctx context.Context, playerID, faction string) error {
	ownedFactions, err := di.factionClient.ListPlayerFactions(ctx, playerID)
	if err != nil {
		return fmt.Errorf("check faction ownership: %w", err)
	}
	if !slices.Contains(ownedFactions, faction) {
		return fmt.Errorf("%w: faction %q is not owned by player", port.ErrInvalidDeck, faction)
	}
	return nil
}

// validateInitiatives はセットした施策が選択プロダクト・宣言陣営と整合するか検証します。
func (di *DeckInteractor) validateInitiatives(faction, productID, routineID, specialID string) error {
	product := di.productCache.FindByID(productID)
	if product == nil {
		return fmt.Errorf("%w: product %q not found", port.ErrInvalidDeck, productID)
	}
	if product.Faction != faction {
		return fmt.Errorf("%w: product %q belongs to faction %q, not deck faction %q",
			port.ErrInvalidDeck, productID, product.Faction, faction)
	}
	if err := di.validateInitiative(productID, routineID, domain.InitiativeKindRoutine); err != nil {
		return err
	}
	return di.validateInitiative(productID, specialID, domain.InitiativeKindSpecial)
}

// validateInitiative は施策が指定プロダクトに属し、かつ指定区分かを検証します。
func (di *DeckInteractor) validateInitiative(productID, initiativeID, kind string) error {
	initiative := di.initiativeCache.FindByID(initiativeID)
	if initiative == nil || initiative.ProductID != productID || initiative.Kind != kind {
		return fmt.Errorf("%w: %q is not a %s of product %q", port.ErrInvalidDeck, initiativeID, kind, productID)
	}
	return nil
}

func (di *DeckInteractor) validateDeckCards(faction string, deckCardEntries []domain.DeckCardEntry, ownedCards []*domain.PlayerCard) error {
	type ownedKey struct {
		cardID string
		artNo  int64
	}

	if !slices.Contains(gamedesign.SelectableFactions, faction) {
		return fmt.Errorf("%w: faction %q is not selectable", port.ErrInvalidDeck, faction)
	}

	for _, entry := range deckCardEntries {
		if entry.Count <= 0 {
			return fmt.Errorf("%w: card %s variant %d: count must be positive", port.ErrInvalidDeck, entry.CardID, entry.ArtNo)
		}
	}
	if countDeckCards(deckCardEntries) > gamedesign.DeckSize {
		return fmt.Errorf("%w: deck cannot exceed %d cards", port.ErrInvalidDeck, gamedesign.DeckSize)
	}

	owned := make(map[ownedKey]int, len(ownedCards))
	for _, ownedCard := range ownedCards {
		owned[ownedKey{ownedCard.CardID, ownedCard.ArtNo}] = ownedCard.Count
	}

	for _, entry := range deckCardEntries {
		key := ownedKey{entry.CardID, entry.ArtNo}
		if owned[key] < entry.Count {
			return fmt.Errorf("%w: card %s variant %d: not enough owned (need %d, have %d)",
				port.ErrUnowned, entry.CardID, entry.ArtNo, entry.Count, owned[key])
		}
	}

	cardIDTotals := make(map[string]int)
	for _, entry := range deckCardEntries {
		cardIDTotals[entry.CardID] += entry.Count
	}
	for cardID, total := range cardIDTotals {
		card := di.cardCache.FindByID(cardID)
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

func (di *DeckInteractor) isDeckValid(deck *domain.Deck, deckCards []domain.DeckCard, ownedCards []*domain.PlayerCard) bool {
	deckCardEntries := domain.ToDeckCardEntries(deckCards)
	if countDeckCards(deckCardEntries) != gamedesign.DeckSize {
		return false
	}
	if di.validateDeckCards(deck.Faction, deckCardEntries, ownedCards) != nil {
		return false
	}
	return di.validateInitiatives(deck.Faction, deck.ProductID, deck.RoutineID, deck.SpecialID) == nil
}
