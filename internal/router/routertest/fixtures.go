package routertest

import (
	"errors"
	"slices"
	"time"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

const (
	// PlayerA はテストのプレイヤー A の ID です。
	PlayerA = "00000000-0000-0000-0000-000000000001"
	// PlayerB はテストのプレイヤー B の ID です。
	PlayerB = "00000000-0000-0000-0000-000000000002"

	// TokenA はプレイヤー A として検証されるトークンです。
	TokenA = "test-token-player-a"
	// InvalidToken は検証に失敗するトークンです。
	InvalidToken = "invalid-token"

	// PlayerCardsPath は所持カード一覧の取得の経路です。
	PlayerCardsPath = "/api/v1/cards/cards"
	// CardOwnershipPath は所持状態付きカード定義一覧の取得の経路です。
	CardOwnershipPath = "/api/v1/cards/cards/with-ownership"
	// ProductsPath はプロダクト一覧の取得の経路です。
	ProductsPath = "/api/v1/cards/products"
	// DecksPath はデッキ一覧の取得とデッキの作成の経路です。
	DecksPath = "/api/v1/cards/decks"
	// PlayerOnboardedPath はプレイヤー登録完了イベントの受信の経路です。
	PlayerOnboardedPath = "/internal/v1/pubsub/player-onboarded"
	// CardPackPurchasedPath はカードパック購入イベントの受信の経路です。
	CardPackPurchasedPath = "/internal/v1/pubsub/card-pack-purchased"

	// InvalidDeckIDBody はデッキ ID が不正なときの応答本文です。
	InvalidDeckIDBody = `{"error":"invalid deck_id"}`
	// MalformedRequestBodyBody はリクエストボディが不正なときの応答本文です。
	MalformedRequestBodyBody = `{"error":"malformed request body"}`
	// MalformedEnvelopeBody は受信内容が不正なときの応答本文です。
	MalformedEnvelopeBody = `{"error":"pubsub push: malformed envelope"}`
	// MalformedBase64Body は data の base64 が不正なときの応答本文です。
	MalformedBase64Body = `{"error":"pubsub push: malformed base64 data"}`

	initialDeckID = 42
)

var (
	// StoredCreatedAt は保存済みのデータに使う固定の作成日時です。
	StoredCreatedAt = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	// StoredUpdatedAt は保存済みのデータに使う固定の更新日時です。
	StoredUpdatedAt = time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)

	// ErrCardFindAllFailed はカード定義の取得の失敗として偽のデータベースに入れる原因です。
	ErrCardFindAllFailed = errors.New("card-find-all-failed")
	// ErrPlayerCardFindFailed は所持カードの取得の失敗として偽のデータベースに入れる原因です。
	ErrPlayerCardFindFailed = errors.New("player-card-find-failed")
	// ErrDeckFindFailed はデッキの取得の失敗として偽のデータベースに入れる原因です。
	ErrDeckFindFailed = errors.New("deck-find-failed")
	// ErrDeckCardFindFailed はデッキの構成カードの取得の失敗として偽のデータベースに入れる原因です。
	ErrDeckCardFindFailed = errors.New("deck-card-find-failed")
	// ErrDeckSaveFailed はデッキの保存の失敗として偽のデータベースに入れる原因です。
	ErrDeckSaveFailed = errors.New("deck-save-failed")
	// ErrDeckDeleteFailed はデッキの削除の失敗として偽のデータベースに入れる原因です。
	ErrDeckDeleteFailed = errors.New("deck-delete-failed")
	// ErrAccountFactionOwnedFailed は陣営の所持の照会の失敗として偽の照会に入れる原因です。
	ErrAccountFactionOwnedFailed = errors.New("account-faction-owned-failed")
	// ErrPlayerCardSaveFailed は所持カードの保存の失敗として偽のデータベースに入れる原因です。
	ErrPlayerCardSaveFailed = errors.New("player-card-save-failed")

	errInvalidToken = errors.New("test token is invalid")
)

// CardCount はデッキまたは所持カードの 1 種類のカードとアート番号と枚数です。
type CardCount struct {
	CardID string
	ArtNo  int64
	Count  int
}

// ValidDeckComposition はバトルに使える 30 枚のデッキの構成です。
var ValidDeckComposition = []CardCount{
	{"TST-0101", 0, 3}, {"TST-0102", 0, 3}, {"TST-0103", 0, 3}, {"TST-0104", 0, 3}, {"TST-0105", 0, 3},
	{"TST-0111", 0, 3}, {"TST-0112", 0, 3}, {"TST-0113", 0, 3}, {"TST-0114", 0, 3}, {"TST-0115", 0, 3},
}

// NewPointer は値へのポインタを返します。
func NewPointer[T any](value T) *T {
	return &value
}

// ChangeCompositionCount は構成の複製のうち指定カードの枚数を変えて返します。
func ChangeCompositionCount(composition []CardCount, cardID string, count int) []CardCount {
	changed := slices.Clone(composition)
	index := slices.IndexFunc(changed, func(entry CardCount) bool { return entry.CardID == cardID })
	changed[index].Count = count
	return changed
}

// RemoveFromComposition は構成の複製から指定カードを除いて返します。
func RemoveFromComposition(composition []CardCount, cardID string) []CardCount {
	return slices.DeleteFunc(slices.Clone(composition), func(entry CardCount) bool { return entry.CardID == cardID })
}

// MasterData はテストの起動時に読み込むカード定義・プロダクト・施策・配布パックです。
type MasterData struct {
	Cards       []*domain.Card
	Products    []*domain.Product
	Initiatives []*domain.Initiative
	CardPacks   map[string]domain.CardPack
}

// NewTestCard はテスト用のカード定義を生成します。
func NewTestCard(cardID, faction, restriction string) *domain.Card {
	return &domain.Card{
		CardID:        cardID,
		CardName:      "テストカード" + cardID,
		ResourceLabel: "テストラベル",
		Faction:       faction,
		CardType:      "Strategy",
		Restriction:   restriction,
		IsActive:      true,
		CreatedAt:     StoredCreatedAt,
		UpdatedAt:     StoredUpdatedAt,
	}
}

// NewTestInitiative はテスト用の施策を生成します。
func NewTestInitiative(initiativeID, productID, kind string) *domain.Initiative {
	return &domain.Initiative{
		InitiativeID: initiativeID,
		ProductID:    productID,
		Kind:         kind,
		Name:         "テスト施策" + initiativeID,
		EffectText:   "テスト効果文",
		Effect:       map[string]any{},
		IsActive:     true,
	}
}

// NewTestProduct はテスト用のプロダクトを生成します。
func NewTestProduct(productID, faction string) *domain.Product {
	return &domain.Product{ProductID: productID, Faction: faction, ProductName: "テストプロダクト" + productID, IsActive: true}
}

// NewTestCardPack はテスト用の配布パックを生成します。
func NewTestCardPack(packID string, cards ...domain.CardPackCard) domain.CardPack {
	return domain.CardPack{PackID: packID, Description: "テストパック", IsActive: true, Cards: cards}
}

// NewMasterData は既定のテスト用マスターデータを生成します。
func NewMasterData() MasterData {
	cards := []*domain.Card{
		NewTestCard("TST-0001", "SHE", "unlimited"),
		NewTestCard("TST-0002", "SHE", "unlimited"),
		NewTestCard("TST-0003", "Neutral", "limited"),
		NewTestCard("TST-0101", "SHE", "unlimited"),
		NewTestCard("TST-0102", "SHE", "unlimited"),
		NewTestCard("TST-0103", "SHE", "unlimited"),
		NewTestCard("TST-0104", "SHE", "unlimited"),
		NewTestCard("TST-0105", "SHE", "unlimited"),
		NewTestCard("TST-0111", "Neutral", "unlimited"),
		NewTestCard("TST-0112", "Neutral", "unlimited"),
		NewTestCard("TST-0113", "Neutral", "unlimited"),
		NewTestCard("TST-0114", "Neutral", "unlimited"),
		NewTestCard("TST-0115", "Neutral", "unlimited"),
		NewTestCard("TST-0116", "Neutral", "unlimited"),
	}
	cardPacks := map[string]domain.CardPack{
		"basic":             NewTestCardPack("basic", domain.CardPackCard{CardID: "TST-0001", Copies: 3}),
		"faction_set_she":   NewTestCardPack("faction_set_she", domain.CardPackCard{CardID: "TST-0002", Copies: 2}),
		"faction_set_tenki": NewTestCardPack("faction_set_tenki", domain.CardPackCard{CardID: "TST-0003", Copies: 1}),
		"TST-0021":          NewTestCardPack("TST-0021", domain.CardPackCard{CardID: "TST-0001", Copies: 3}),
		"TST-0022":          NewTestCardPack("TST-0022", domain.CardPackCard{CardID: "TST-0002", Copies: 2}),
	}
	return MasterData{
		Cards:    cards,
		Products: []*domain.Product{NewTestProduct("TST-0001", "SHE")},
		Initiatives: []*domain.Initiative{
			NewTestInitiative("TST-0011", "TST-0001", "routine"),
			NewTestInitiative("TST-0012", "TST-0001", "special"),
		},
		CardPacks: cardPacks,
	}
}

// NewStoredDeck はプレイヤー A の保存済みデッキを生成します。
func NewStoredDeck(deckID int64) domain.Deck {
	return domain.Deck{
		PlayerID:  PlayerA,
		DeckID:    deckID,
		DeckName:  "テストデッキA",
		Faction:   "SHE",
		ProductID: "TST-0001",
		RoutineID: "TST-0011",
		SpecialID: "TST-0012",
		CreatedAt: StoredCreatedAt,
		UpdatedAt: StoredUpdatedAt,
	}
}

// BuildDeckCards は構成から保存済みデッキの構成カードを組み立てます。
func BuildDeckCards(playerID string, deckID int64, composition []CardCount) []domain.DeckCard {
	cards := make([]domain.DeckCard, 0, len(composition))
	for _, entry := range composition {
		cards = append(cards, domain.DeckCard{PlayerID: playerID, DeckID: deckID, CardID: entry.CardID, ArtNo: entry.ArtNo, Count: entry.Count})
	}
	return cards
}

// DeckRequestCard はデッキの作成・更新の要求に含めるカードです。
type DeckRequestCard struct {
	CardID string `json:"card_id"`
	ArtNo  int64  `json:"art_no"`
	Count  int    `json:"count"`
}

// DeckRequestBody はデッキの作成・更新の要求本文です。
type DeckRequestBody struct {
	DeckName  string            `json:"deck_name"`
	Faction   string            `json:"faction"`
	ProductID string            `json:"product_id"`
	RoutineID string            `json:"routine_id"`
	SpecialID string            `json:"special_id"`
	PlaymatNo *int64            `json:"playmat_no,omitempty"`
	SleeveNo  *int64            `json:"sleeve_no,omitempty"`
	Cards     []DeckRequestCard `json:"cards"`
}

// ToDeckRequestCards は構成を要求に含めるカードの一覧に変換します。
func ToDeckRequestCards(composition []CardCount) []DeckRequestCard {
	cards := make([]DeckRequestCard, 0, len(composition))
	for _, entry := range composition {
		cards = append(cards, DeckRequestCard{CardID: entry.CardID, ArtNo: entry.ArtNo, Count: entry.Count})
	}
	return cards
}

// NewDeckRequestBody は構成を指定したデッキの要求本文を生成します。
func NewDeckRequestBody(composition []CardCount) DeckRequestBody {
	return DeckRequestBody{
		DeckName:  "テストデッキA",
		Faction:   "SHE",
		ProductID: "TST-0001",
		RoutineID: "TST-0011",
		SpecialID: "TST-0012",
		Cards:     ToDeckRequestCards(composition),
	}
}

// NewValidDeckRequestBody は 30 枚のデッキの要求本文を生成します。
func NewValidDeckRequestBody() DeckRequestBody {
	return NewDeckRequestBody(ValidDeckComposition)
}

// AddThirtyFirstCard は要求に 31 枚目のカードを足し、プレイヤー A に所持させます。
func AddThirtyFirstCard(h *Harness, body *DeckRequestBody) {
	h.DB.SetOwned(PlayerA, "TST-0116", 0, 1)
	body.Cards = append(body.Cards, DeckRequestCard{CardID: "TST-0116", ArtNo: 0, Count: 1})
}

// ReduceOwnedFirstCard はプレイヤー A の TST-0101 の所持を 2 枚にします。
func ReduceOwnedFirstCard(h *Harness) {
	h.DB.SetOwned(PlayerA, "TST-0101", 0, 2)
}

// DeclareNeutral は要求の宣言陣営を Neutral にします。
func DeclareNeutral(body *DeckRequestBody) {
	body.Faction = "Neutral"
}

// SwapInLimitedCards は要求の TST-0115 を制限区分が limited の TST-0003 の 2 枚に置き換え、プレイヤー A に 2 枚所持させます。
func SwapInLimitedCards(h *Harness, body *DeckRequestBody) {
	h.DB.SetOwned(PlayerA, "TST-0003", 0, 2)
	body.Cards = slices.DeleteFunc(body.Cards, func(card DeckRequestCard) bool { return card.CardID == "TST-0115" })
	body.Cards = append(body.Cards, DeckRequestCard{CardID: "TST-0003", ArtNo: 0, Count: 2})
}

// ReduceToTwentyNineCards は要求の TST-0115 を 2 枚にして合計 29 枚にします。
func ReduceToTwentyNineCards(body *DeckRequestBody) {
	index := slices.IndexFunc(body.Cards, func(card DeckRequestCard) bool { return card.CardID == "TST-0115" })
	body.Cards[index].Count = 2
}
