package routertest

import (
	"context"
	"maps"
	"slices"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// StoredDeck は偽のデータベースに保存したデッキとその構成カードです。
type StoredDeck struct {
	Deck  domain.Deck
	Cards []domain.DeckCard
}

// FakeDatabase はカード・所持カード・デッキ・カードパック・処理済みイベントを持つメモリ上のデータベースの偽物です。
type FakeDatabase struct {
	Cards           []*domain.Card
	PlayerCards     []domain.PlayerCard
	Decks           []StoredDeck
	NextDeckID      int64
	CardPacks       map[string]domain.CardPack
	ProcessedEvents map[string]string

	ErrFindAllCards   error
	ErrGetPlayerCards error
	ErrFindDeck       error
	ErrGetDeckCards   error
	ErrSaveDeck       error
	ErrDeleteDeck     error

	errAddCards      error
	failingAddCardID string
}

// NewFakeDatabase は空の偽のデータベースを生成します。
func NewFakeDatabase() *FakeDatabase {
	return &FakeDatabase{
		NextDeckID:      initialDeckID,
		CardPacks:       map[string]domain.CardPack{},
		ProcessedEvents: map[string]string{},
	}
}

// FindAll は保存したカード定義を返します。
func (db *FakeDatabase) FindAll(_ context.Context) ([]*domain.Card, error) {
	return db.Cards, db.ErrFindAllCards
}

// GetPlayerCards は指定プレイヤーの所持カードを返します。
func (db *FakeDatabase) GetPlayerCards(_ context.Context, playerID string) ([]*domain.PlayerCard, error) {
	if db.ErrGetPlayerCards != nil {
		return nil, db.ErrGetPlayerCards
	}
	owned := []*domain.PlayerCard{}
	for _, playerCard := range db.PlayerCards {
		if playerCard.PlayerID == playerID {
			copied := playerCard
			owned = append(owned, &copied)
		}
	}
	return owned, nil
}

// AddCards は所持カードの枚数を加算します。失敗を指定したカードを含むときはエラーを返します。
func (db *FakeDatabase) AddCards(_ context.Context, cards []domain.PlayerCard) error {
	if db.errAddCards != nil && slices.ContainsFunc(cards, func(card domain.PlayerCard) bool { return card.CardID == db.failingAddCardID }) {
		return db.errAddCards
	}
	for _, card := range cards {
		index := db.findPlayerCardIndex(card.PlayerID, card.CardID, card.ArtNo)
		if index < 0 {
			db.PlayerCards = append(db.PlayerCards, card)
			continue
		}
		db.PlayerCards[index].Count += card.Count
	}
	return nil
}

func (db *FakeDatabase) findPlayerCardIndex(playerID, cardID string, artNo int64) int {
	return slices.IndexFunc(db.PlayerCards, func(card domain.PlayerCard) bool {
		return card.PlayerID == playerID && card.CardID == cardID && card.ArtNo == artNo
	})
}

// SetOwned はプレイヤーの所持枚数を指定の値にします。
func (db *FakeDatabase) SetOwned(playerID, cardID string, artNo int64, count int) {
	index := db.findPlayerCardIndex(playerID, cardID, artNo)
	if index < 0 {
		db.PlayerCards = append(db.PlayerCards, domain.PlayerCard{PlayerID: playerID, CardID: cardID, ArtNo: artNo, Count: count})
		return
	}
	db.PlayerCards[index].Count = count
}

// SetOwnedComposition は構成のとおりにプレイヤーの所持枚数を指定します。
func (db *FakeDatabase) SetOwnedComposition(playerID string, composition []CardCount) {
	for _, entry := range composition {
		db.SetOwned(playerID, entry.CardID, entry.ArtNo, entry.Count)
	}
}

// FailAddingCard は指定カードを含む所持カードの加算を指定のエラーで失敗させます。
func (db *FakeDatabase) FailAddingCard(cardID string, err error) {
	db.errAddCards = err
	db.failingAddCardID = cardID
}

// RecoverAddingCards は所持カードの加算の失敗を解除します。
func (db *FakeDatabase) RecoverAddingCards() {
	db.errAddCards = nil
}

func (db *FakeDatabase) findDeckIndex(playerID string, deckID int64) int {
	return slices.IndexFunc(db.Decks, func(stored StoredDeck) bool {
		return stored.Deck.PlayerID == playerID && stored.Deck.DeckID == deckID
	})
}

// StoreDeck はデッキとその構成カードを保存します。
func (db *FakeDatabase) StoreDeck(deck domain.Deck, cards []domain.DeckCard) {
	db.Decks = append(db.Decks, StoredDeck{Deck: deck, Cards: cards})
}

func toDeckCards(entries []domain.DeckCardEntry, playerID string, deckID int64) []domain.DeckCard {
	cards := make([]domain.DeckCard, 0, len(entries))
	for _, entry := range entries {
		cards = append(cards, domain.DeckCard{PlayerID: playerID, DeckID: deckID, CardID: entry.CardID, ArtNo: entry.ArtNo, Count: entry.Count})
	}
	return cards
}

// Create はデッキを採番して保存します。
func (db *FakeDatabase) Create(_ context.Context, deck domain.Deck, entries []domain.DeckCardEntry) (int64, error) {
	if db.ErrSaveDeck != nil {
		return 0, db.ErrSaveDeck
	}
	deckID := db.NextDeckID
	db.NextDeckID++
	deck.DeckID = deckID
	db.StoreDeck(deck, toDeckCards(entries, deck.PlayerID, deckID))
	return deckID, nil
}

// FindByPlayerID は指定プレイヤーのデッキを返します。
func (db *FakeDatabase) FindByPlayerID(_ context.Context, playerID string) ([]*domain.Deck, error) {
	if db.ErrFindDeck != nil {
		return nil, db.ErrFindDeck
	}
	decks := []*domain.Deck{}
	for _, stored := range db.Decks {
		if stored.Deck.PlayerID == playerID {
			copied := stored.Deck
			decks = append(decks, &copied)
		}
	}
	return decks, nil
}

// FindByID は指定プレイヤーの指定デッキを返します。無ければ port.ErrNotFound を返します。
func (db *FakeDatabase) FindByID(_ context.Context, playerID string, deckID int64) (*domain.Deck, error) {
	if db.ErrFindDeck != nil {
		return nil, db.ErrFindDeck
	}
	index := db.findDeckIndex(playerID, deckID)
	if index < 0 {
		return nil, port.ErrNotFound
	}
	copied := db.Decks[index].Deck
	return &copied, nil
}

// GetDeckCards は指定デッキの構成カードを返します。
func (db *FakeDatabase) GetDeckCards(_ context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
	if db.ErrGetDeckCards != nil {
		return nil, db.ErrGetDeckCards
	}
	index := db.findDeckIndex(playerID, deckID)
	if index < 0 {
		return nil, nil
	}
	return slices.Clone(db.Decks[index].Cards), nil
}

// Update は作成日時を保ったままデッキと構成カードを差し替えます。無ければ port.ErrNotFound を返します。
func (db *FakeDatabase) Update(_ context.Context, deck domain.Deck, entries []domain.DeckCardEntry) error {
	if db.ErrSaveDeck != nil {
		return db.ErrSaveDeck
	}
	index := db.findDeckIndex(deck.PlayerID, deck.DeckID)
	if index < 0 {
		return port.ErrNotFound
	}
	deck.CreatedAt = db.Decks[index].Deck.CreatedAt
	db.Decks[index] = StoredDeck{Deck: deck, Cards: toDeckCards(entries, deck.PlayerID, deck.DeckID)}
	return nil
}

// Delete は指定デッキを削除します。無ければ port.ErrNotFound を返します。
func (db *FakeDatabase) Delete(_ context.Context, playerID string, deckID int64) error {
	if db.ErrDeleteDeck != nil {
		return db.ErrDeleteDeck
	}
	index := db.findDeckIndex(playerID, deckID)
	if index < 0 {
		return port.ErrNotFound
	}
	db.Decks = slices.Delete(db.Decks, index, index+1)
	return nil
}

// GetPack は配布パックを返します。無ければ port.ErrNotFound を返します。
func (db *FakeDatabase) GetPack(_ context.Context, packID string) (*domain.CardPack, error) {
	pack, isFound := db.CardPacks[packID]
	if !isFound {
		return nil, port.ErrNotFound
	}
	return &pack, nil
}

// Insert はイベントを処理済みとして記録し、初めての記録なら true を返します。
func (db *FakeDatabase) Insert(_ context.Context, eventID, eventType string) (bool, error) {
	_, isRecorded := db.ProcessedEvents[eventID]
	db.ProcessedEvents[eventID] = eventType
	return !isRecorded, nil
}

// RunInTransaction は fn を実行し、エラーのときは所持カードと処理済みイベントの変更をすべて取り消します。
func (db *FakeDatabase) RunInTransaction(_ context.Context, fn func(repositories port.GrantRepositories) error) error {
	savedPlayerCards := slices.Clone(db.PlayerCards)
	savedProcessedEvents := maps.Clone(db.ProcessedEvents)
	err := fn(port.GrantRepositories{ProcessedEvents: db, CardPacks: db, PlayerCards: db})
	if err != nil {
		db.PlayerCards = savedPlayerCards
		db.ProcessedEvents = savedProcessedEvents
	}
	return err
}
