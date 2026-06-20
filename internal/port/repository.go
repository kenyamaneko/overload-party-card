package port

import (
	"context"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

// CardRepo はカード定義の永続化を抽象化するインターフェースです。
type CardRepo interface {
	FindAll(ctx context.Context) ([]*domain.Card, error)
}

// ProductRepo はプロダクト定義の永続化を抽象化するインターフェースです。
type ProductRepo interface {
	FindAll(ctx context.Context) ([]*domain.Product, error)
}

// InitiativeRepo は施策定義の永続化を抽象化するインターフェースです。
type InitiativeRepo interface {
	FindAll(ctx context.Context) ([]*domain.Initiative, error)
}

// FactionClient はプレイヤーの所持ファクションを所有サービス (account) から取得します。
type FactionClient interface {
	ListPlayerFactions(ctx context.Context, playerID string) ([]string, error)
}

// PlayerCardRepo はプレイヤー所持カードの永続化を抽象化するインターフェースです。
type PlayerCardRepo interface {
	GetPlayerCards(ctx context.Context, playerID string) ([]*domain.PlayerCard, error)
	// AddCards は UPSERT-with-add セマンティクスで所持カードを追加します。
	// 各 CardPackCard.Copies が個別に加算されます。戻り値は加算したコピー総数です。
	AddCards(ctx context.Context, playerID string, cards []domain.CardPackCard) (int, error)
}

// CardPackRepo は配布パックマスター (card.card_pack) の永続化を抽象化するインターフェースです。
// 実装は配布リクエストごとに DB を引く前提でキャッシュは行いません (理由は
// docs/ARCHITECTURE.md「pack マスターはキャッシュしない」を参照)。
type CardPackRepo interface {
	// GetPack は pack_id に対応するパック定義を返します。
	// 行が存在しない場合 ErrNotFound を返します (運用停止 = is_active=false は別エラー: ErrPackInactive)。
	GetPack(ctx context.Context, packID string) (*domain.CardPack, error)
}

// ProcessedEventRepo は処理済み Pub/Sub イベントを追跡するインターフェースです。
type ProcessedEventRepo interface {
	// Insert は新規行が挿入された場合 true を返します（冪等性ガード）。
	Insert(ctx context.Context, eventID, eventType string) (bool, error)
}

// DeckRepo はデッキの永続化を抽象化するインターフェースです。
type DeckRepo interface {
	// Create は decks 行と deck_cards 行を 1 トランザクションで挿入し、
	// 採番された deck_id を返します。入力は mutation しません。
	Create(ctx context.Context, deck domain.Deck, deckCardEntries []domain.DeckCardEntry) (int64, error)
	FindByPlayerID(ctx context.Context, playerID string) ([]*domain.Deck, error)
	FindByID(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error)
	GetDeckCards(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error)
	// Update はデッキを更新し deck_cards を差し替えます。入力は mutation しません。
	Update(ctx context.Context, deck domain.Deck, deckCardEntries []domain.DeckCardEntry) error
	Delete(ctx context.Context, playerID string, deckID int64) error
}
