package domain

import "time"

// Deck はプレイヤーのデッキを表します。
// カード構成 (DeckCard) とバトルで使えるか (is_valid) は、デッキとは別に取得・算出するため持ちません。
type Deck struct {
	PlayerID  string
	DeckID    int64
	DeckName  string
	Faction   string
	ProductID string
	RoutineID string
	SpecialID string
	PlaymatNo *int64
	SleeveNo  *int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DeckDetail はデッキと、そのカード構成、バトルで使えるかの判定をまとめたものです。
type DeckDetail struct {
	Deck    Deck
	Cards   []DeckCard
	IsValid bool
}

// DeckDraft はデッキの作成・更新で利用者が指定する内容です。
type DeckDraft struct {
	DeckName  string
	Faction   string
	ProductID string
	RoutineID string
	SpecialID string
	PlaymatNo *int64
	SleeveNo  *int64
	Cards     []DeckCardEntry
}

// DeckCard はデッキに含まれるカード 1 種類 (アート番号ごと) とその枚数を表します。
type DeckCard struct {
	PlayerID string
	DeckID   int64
	CardID   string
	ArtNo    int64
	Count    int
}

// DeckCardEntry はデッキの作成・更新で指定するカード 1 種類 (アート番号ごと) とその枚数です。
// プレイヤー ID とデッキ ID は呼び出し側の文脈で決まるため含めません。
type DeckCardEntry struct {
	CardID string
	ArtNo  int64
	Count  int
}

// DeckCardEntry はデッキに含まれるカードを、デッキの作成・更新の指定形式に変換します。
func (d DeckCard) DeckCardEntry() DeckCardEntry {
	return DeckCardEntry{
		CardID: d.CardID,
		ArtNo:  d.ArtNo,
		Count:  d.Count,
	}
}

// ToDeckCardEntries はデッキに含まれるカードの一覧を、デッキの作成・更新の指定形式の一覧に変換します。
func ToDeckCardEntries(cards []DeckCard) []DeckCardEntry {
	entries := make([]DeckCardEntry, len(cards))
	for i, card := range cards {
		entries[i] = card.DeckCardEntry()
	}
	return entries
}
