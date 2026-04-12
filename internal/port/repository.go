package port

import (
	"context"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	"github.com/kenyamaneko/overload-party-card/internal/model"
)

// CardRepo はカード定義の永続化を抽象化するインターフェースです。
type CardRepo interface {
	FindAll(ctx context.Context) ([]*apicard.CardDefinition, error)
	FindByCardID(ctx context.Context, cardID string) (*apicard.CardDefinition, error)
	// FindCardIDsByFactions は指定ファクション群に属する有効カードの card_id を返します。
	// 配布対象カードをハードコードではなく card_definitions に委ねるためのメソッドです。
	FindCardIDsByFactions(ctx context.Context, factions []string) ([]string, error)
}

// PlayerCardRepo はプレイヤー所持カードの永続化を抽象化するインターフェースです。
type PlayerCardRepo interface {
	GetPlayerCards(ctx context.Context, playerID string) ([]*model.PlayerCard, error)
	// AddCards は UPSERT-with-add セマンティクスで所持カードを追加します。
	// 戻り値は追加されたコピー総数（= len(cardIDs) * countPerCard）です。
	AddCards(ctx context.Context, playerID string, cardIDs []string, countPerCard int) (int, error)
}

// ProcessedEventRepo は処理済み Pub/Sub イベントを追跡するインターフェースです。
type ProcessedEventRepo interface {
	// Insert は新規行が挿入された場合 true を返します（冪等性ガード）。
	Insert(ctx context.Context, eventID, eventType string) (bool, error)
}

// DeckRepo はデッキの永続化を抽象化するインターフェースです。
type DeckRepo interface {
	Create(ctx context.Context, deck *apicard.Deck, cards []apicard.DeckCard) error
	FindByPlayerID(ctx context.Context, playerID string) ([]*apicard.Deck, error)
	FindByID(ctx context.Context, playerID string, deckID int64) (*apicard.Deck, error)
	GetDeckCards(ctx context.Context, playerID string, deckID int64) ([]apicard.DeckCard, error)
	Update(ctx context.Context, deck *apicard.Deck, cards []apicard.DeckCard) error
	Delete(ctx context.Context, playerID string, deckID int64) error
}
