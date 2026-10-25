package domain

// CardPack は配布パック (どのカードを何枚配るか) を表します。
type CardPack struct {
	PackID      string
	Description string
	IsActive    bool
	Cards       []CardPackCard
}

// CardPackCard は配布パックに含まれるカード 1 種類と配布枚数の組です。
type CardPackCard struct {
	CardID string
	Copies int
}
