package port

import (
	"context"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

// CardRepository はカード定義の永続化を抽象化するインターフェースです。
type CardRepository interface {
	// FindAll は有効な全カード定義をカード ID の昇順で返します。
	FindAll(ctx context.Context) ([]*domain.Card, error)
}

// ProductRepository はプロダクトの永続化を抽象化するインターフェースです。
type ProductRepository interface {
	// FindAll は有効な全プロダクトをプロダクト ID の昇順で返します。
	FindAll(ctx context.Context) ([]*domain.Product, error)
}

// InitiativeRepository は施策の永続化を抽象化するインターフェースです。
type InitiativeRepository interface {
	// FindAll は有効な全施策を施策 ID の昇順で返します。
	FindAll(ctx context.Context) ([]*domain.Initiative, error)
}

// FactionClient はプレイヤーの所持陣営を所有サービス (account) から取得します。
type FactionClient interface {
	// ListPlayerFactions は指定プレイヤーが所持する陣営の一覧を返します。
	ListPlayerFactions(ctx context.Context, playerID string) ([]string, error)
}

// PlayerCardRepository はプレイヤーの所持カードの永続化を抽象化するインターフェースです。
type PlayerCardRepository interface {
	// GetPlayerCards は指定プレイヤーの所持カードをカード ID・アート番号の昇順で返します。
	GetPlayerCards(ctx context.Context, playerID string) ([]*domain.PlayerCard, error)
	// AddCards は指定の所持カードを、プレイヤー・カード・アート番号ごとに所持枚数へ加算します。
	// 所持していないカードは新たに所持カードに加えます。
	AddCards(ctx context.Context, cards []domain.PlayerCard) error
}

// CardPackRepository は配布パックの永続化を抽象化するインターフェースです。
type CardPackRepository interface {
	// GetPack はパック ID に対応する配布パックを返します。
	// パックが存在しない場合は ErrNotFound を返します。
	GetPack(ctx context.Context, packID string) (*domain.CardPack, error)
}

// ProcessedEventRepository は処理済みの Pub/Sub イベントの記録を抽象化するインターフェースです。
type ProcessedEventRepository interface {
	// Insert はイベントを処理済みとして記録します。
	// 初めて記録したときは isInserted が true、記録済みのイベントのときは false になります。
	Insert(ctx context.Context, eventID, eventType string) (isInserted bool, err error)
}

// DeckRepository はデッキの永続化を抽象化するインターフェースです。
type DeckRepository interface {
	// Create はデッキとそのカード構成を同時に作成し、採番したデッキ ID を返します。
	Create(ctx context.Context, deck domain.Deck, deckCardEntries []domain.DeckCardEntry) (int64, error)
	// FindByPlayerID は指定プレイヤーの全デッキを更新日時の降順で返します。
	FindByPlayerID(ctx context.Context, playerID string) ([]*domain.Deck, error)
	// FindByID は指定プレイヤーの指定デッキを返します。
	// 該当するデッキが無い場合は ErrNotFound を返します。
	FindByID(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error)
	// GetDeckCards は指定デッキのカード構成を返します。
	GetDeckCards(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error)
	// Update はデッキを更新し、カード構成を指定の内容に差し替えます。
	// 該当するデッキが無い場合は ErrNotFound を返します。
	Update(ctx context.Context, deck domain.Deck, deckCardEntries []domain.DeckCardEntry) error
	// Delete は指定デッキとそのカード構成を削除します。
	// 該当するデッキが無い場合は ErrNotFound を返します。
	Delete(ctx context.Context, playerID string, deckID int64) error
}
