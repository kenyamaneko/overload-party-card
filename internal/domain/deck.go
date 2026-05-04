package domain

import "time"

// Deck は decks テーブル 1 行に対応する domain 表現。
// IsValid (バトル可否) と DeckCards (構成カード) は API 応答時に組み立てる派生情報なので
// ここには持たない。
type Deck struct {
	PlayerID  string
	DeckID    int64
	DeckName  string
	PlaymatNo *int64
	SleeveNo  *int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DeckCard は deck_cards テーブル 1 行に対応する domain 表現。
type DeckCard struct {
	PlayerID string
	DeckID   int64
	CardID   string
	ArtNo    int64
	Count    int
}

// DeckCardEntry はデッキ作成・更新リクエストのカード指定 1 件分。
// PlayerID / DeckID は呼び出し側コンテキストで決まるため含めない。
type DeckCardEntry struct {
	CardID string
	ArtNo  int64
	Count  int
}
