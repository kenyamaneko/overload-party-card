package domain

import "time"

// CardPack は配布パック (どのカードを何枚配るか) を表す domain 値。
type CardPack struct {
	PackID        string
	Description   string
	Selection     Selection
	CopiesPerCard int
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Selection は配布対象カードの指定方式を表す sum type です。
// domain パッケージ内の private marker method で外部実装をブロックし、
// 呼び出し側の type switch で網羅性を担保します。
type Selection interface {
	isSelection()
}

// SelectionByFactions は指定 faction(s) に属する全 active card を配布対象とします。
type SelectionByFactions struct {
	Factions []string
}

func (SelectionByFactions) isSelection() {}

// SelectionByCardIDs は指定 card_id のみを配布対象とします (限定カード等)。
type SelectionByCardIDs struct {
	CardIDs []string
}

func (SelectionByCardIDs) isSelection() {}
