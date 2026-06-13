package domain

import "time"

// Deck は decks テーブル 1 行に対応する domain 表現。
// IsValid (バトル可否) と DeckCards (構成カード) は API 応答時に組み立てる派生情報なので
// ここには持たない。
type Deck struct {
	PlayerID  string
	DeckID    int64
	DeckName  string
	Faction   string
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

// ToEntry は永続化済みの DeckCard を Entry 形式 (PlayerID/DeckID 抜き) に正規化します。
// バトル前バリデーション等、入力経路 (リクエスト / DB) を問わず同じ検証ロジックに
// 流すための共通正規化です。
func (d DeckCard) ToEntry() DeckCardEntry {
	return DeckCardEntry{
		CardID: d.CardID,
		ArtNo:  d.ArtNo,
		Count:  d.Count,
	}
}

// DeckCardEntriesFromCards は DeckCard slice を Entry slice に変換します。
func DeckCardEntriesFromCards(cards []DeckCard) []DeckCardEntry {
	entries := make([]DeckCardEntry, len(cards))
	for i, c := range cards {
		entries[i] = c.ToEntry()
	}
	return entries
}
