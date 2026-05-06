package domain

import "time"

// CardPack は配布パック (どのカードを何枚配るか) を表す domain 値です。
type CardPack struct {
	PackID      string
	Description string
	IsActive    bool
	Cards       []CardPackCard
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CardPackCard は CardPack に含まれる「カード 1 種と配布枚数」の組です。
type CardPackCard struct {
	CardID string
	Copies int
}
