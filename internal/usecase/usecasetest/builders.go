package usecasetest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

const (
	// PlayerA はテストで使うプレイヤーAのダミーのプレイヤー ID です。
	PlayerA = "00000000-0000-0000-0000-00000000000a"

	// RestrictionUnlimited はカード定義の制限区分 unlimited です。
	RestrictionUnlimited = "unlimited"

	// FactionSHE は陣営 SHE の名前です。
	FactionSHE = "SHE"
	// FactionTenki は陣営 Tenki の名前です。
	FactionTenki = "Tenki"

	// DeckCopiesPerCard はテストのデッキで 1 種類のカードを入れる標準の枚数です。
	DeckCopiesPerCard = 3
	// CatalogCardCount は標準のカードマスターに含めるカードの種類数です。
	CatalogCardCount = 12
	// ValidDeckTotal はバトルに使えるデッキの規定のカード合計枚数です。
	ValidDeckTotal = 30
	// NonexistentDeckID はどのプレイヤーにも存在しないデッキ ID です。
	NonexistentDeckID = 999
)

// NewTestCard はテスト用のカード定義を生成します。
func NewTestCard(cardID, faction, restriction string) *domain.Card {
	return &domain.Card{
		CardID:        cardID,
		CardName:      "テストカード",
		ResourceLabel: "テストラベル",
		Faction:       faction,
		CardType:      "Strategy",
		Restriction:   restriction,
		IsActive:      true,
	}
}

// NewTestProduct はテスト用のプロダクトを生成します。
func NewTestProduct(productID, faction string) *domain.Product {
	return &domain.Product{ProductID: productID, Faction: faction, ProductName: "テストプロダクト", IsActive: true}
}

// NewTestInitiative はテスト用の施策を生成します。
func NewTestInitiative(initiativeID, productID, kind string) *domain.Initiative {
	return &domain.Initiative{
		InitiativeID: initiativeID,
		ProductID:    productID,
		Kind:         kind,
		Name:         "テスト施策",
		InsightCost:  1,
		EffectText:   "テスト効果",
		Effect:       map[string]any{},
		IsActive:     true,
	}
}

// FormatCardID は番号から TST-XXXX 形式のダミーのカード ID を作ります。
func FormatCardID(number int) string {
	return fmt.Sprintf("TST-%04d", number)
}

// NewUnlimitedSheCards は TST-0001 から指定の数までの、制限区分が unlimited の SHE 陣営のカード定義を生成します。
func NewUnlimitedSheCards(count int) []*domain.Card {
	var cards []*domain.Card
	for number := 1; number <= count; number++ {
		cards = append(cards, NewTestCard(FormatCardID(number), FactionSHE, RestrictionUnlimited))
	}
	return cards
}

// LoadCardCache は偽のカードマスターから読み込んだカード定義のキャッシュを返します。
func LoadCardCache(t *testing.T, cards []*domain.Card) *cache.CardCache {
	t.Helper()
	cardCache := cache.NewCardCache()
	require.NoError(t, cardCache.Load(t.Context(), &FakeCardRepository{Cards: cards}))
	return cardCache
}

// LoadProductCache は偽のプロダクトのデータベースから読み込んだプロダクトのキャッシュを返します。
func LoadProductCache(t *testing.T, products []*domain.Product) *cache.ProductCache {
	t.Helper()
	productCache := cache.NewProductCache()
	require.NoError(t, productCache.Load(t.Context(), &FakeProductRepository{Products: products}))
	return productCache
}

// LoadInitiativeCache は偽の施策のデータベースから読み込んだ施策のキャッシュを返します。
func LoadInitiativeCache(t *testing.T, initiatives []*domain.Initiative) *cache.InitiativeCache {
	t.Helper()
	initiativeCache := cache.NewInitiativeCache()
	require.NoError(t, initiativeCache.Load(t.Context(), &FakeInitiativeRepository{Initiatives: initiatives}))
	return initiativeCache
}

// AssertErrorContainsAll はエラー内容が指定の語句をすべて含むことを確かめます。
func AssertErrorContainsAll(t *testing.T, err error, phrases ...string) {
	t.Helper()
	for _, phrase := range phrases {
		assert.ErrorContains(t, err, phrase)
	}
}

// NewDeckCardEntry はデッキの作成・更新で指定するカード 1 種類を生成します。
func NewDeckCardEntry(cardID string, artNo int64, count int) domain.DeckCardEntry {
	return domain.DeckCardEntry{CardID: cardID, ArtNo: artNo, Count: count}
}

// BuildDeckCardEntries は合計が指定の枚数になる、アート 1 のカードの指定を TST-0001 から順に作ります。
func BuildDeckCardEntries(total int) []domain.DeckCardEntry {
	var entries []domain.DeckCardEntry
	remaining := total
	for number := 1; remaining > 0; number++ {
		entries = append(entries, NewDeckCardEntry(FormatCardID(number), 1, min(DeckCopiesPerCard, remaining)))
		remaining -= DeckCopiesPerCard
	}
	return entries
}

// NewOwnedCard はプレイヤーAの所持カードを生成します。
func NewOwnedCard(cardID string, artNo int64, count int) domain.PlayerCard {
	return domain.PlayerCard{PlayerID: PlayerA, CardID: cardID, ArtNo: artNo, Count: count}
}

// ToPlayerCards はカードの指定と同じ枚数を、プレイヤーAの所持カードに変換します。
func ToPlayerCards(entries []domain.DeckCardEntry) []domain.PlayerCard {
	var owned []domain.PlayerCard
	for _, e := range entries {
		owned = append(owned, NewOwnedCard(e.CardID, e.ArtNo, e.Count))
	}
	return owned
}

// CardArtKey はカード ID とアート番号の組です。
type CardArtKey struct {
	CardID string
	ArtNo  int64
}

// CountByCardArt はデッキのカード構成を、カード ID とアート番号ごとの枚数にまとめます。
func CountByCardArt(cards []domain.DeckCard) map[CardArtKey]int {
	counts := map[CardArtKey]int{}
	for _, c := range cards {
		counts[CardArtKey{c.CardID, c.ArtNo}] += c.Count
	}
	return counts
}

// SumDeckCardCounts はデッキのカード構成の合計枚数を返します。
func SumDeckCardCounts(cards []domain.DeckCard) int {
	total := 0
	for _, c := range cards {
		total += c.Count
	}
	return total
}
