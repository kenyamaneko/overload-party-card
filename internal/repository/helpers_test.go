//go:build integration

package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// cardSeed は card.card_definitions への最小シード入力。
// 省略可能なカラムはゼロ値で埋め、呼び出し側がテスト意図に関与するフィールドだけ
// 明示的に指定する。
type cardSeed struct {
	CardID      string
	CardName    string
	Faction     string // "SHE" / "Tenki" / "Sugar" / "Tuners" / "Neutral"
	CardType    string
	Restriction string // "unlimited" / "limited" / "semi_limited" / "forbidden"
	IsActive    bool
}

func seedCard(t *testing.T, s cardSeed) {
	t.Helper()
	_, err := sharedPg.Pool.Exec(context.Background(),
		`INSERT INTO card.card_definitions
		   (card_id, card_name, resource_label, faction, card_type, resizable, elastic, stats, restriction, is_active)
		 VALUES ($1, $2, '', $3, $4, false, false, $5, $6, $7)`,
		s.CardID, s.CardName, s.Faction, s.CardType, json.RawMessage(`{}`), s.Restriction, s.IsActive)
	require.NoError(t, err)
}

func seedCards(t *testing.T, cards []cardSeed) {
	t.Helper()
	for _, c := range cards {
		seedCard(t, c)
	}
}

// fullCardSeed は card.card_definitions への全列シード入力。scanCard の全フィールド
// 写像を検証するテスト用に、省略可能なカラムも含め明示的に指定する。
type fullCardSeed struct {
	CardID        string
	CardName      string
	ResourceLabel string
	Faction       string
	CardType      string
	Subtype       string
	Resizable     bool
	Elastic       bool
	Stats         string
	EffectText    string
	Effects       string
	Restriction   string
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func seedFullCard(t *testing.T, s fullCardSeed) {
	t.Helper()
	_, err := sharedPg.Pool.Exec(context.Background(),
		`INSERT INTO card.card_definitions
		   (card_id, card_name, resource_label, faction, card_type, subtype, resizable, elastic, stats, effect_text, effects, restriction, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		s.CardID, s.CardName, s.ResourceLabel, s.Faction, s.CardType, s.Subtype, s.Resizable, s.Elastic,
		json.RawMessage(s.Stats), s.EffectText, json.RawMessage(s.Effects), s.Restriction, s.IsActive, s.CreatedAt, s.UpdatedAt)
	require.NoError(t, err)
}

// productSeed は card.products への最小シード入力。
type productSeed struct {
	ProductID   string
	Faction     string
	ProductName string
}

func seedProduct(t *testing.T, s productSeed) {
	t.Helper()
	_, err := sharedPg.Pool.Exec(context.Background(),
		`INSERT INTO card.products (product_id, faction, product_name)
		 VALUES ($1, $2, $3)`,
		s.ProductID, s.Faction, s.ProductName)
	require.NoError(t, err)
}

// initiativeSeed は card.initiatives への最小シード入力。Effect は JSON 文字列。
type initiativeSeed struct {
	InitiativeID string
	ProductID    string
	Kind         string
	Name         string
	InsightCost  int64
	EffectText   string
	Effect       string
}

func seedInitiative(t *testing.T, s initiativeSeed) {
	t.Helper()
	_, err := sharedPg.Pool.Exec(context.Background(),
		`INSERT INTO card.initiatives
		   (initiative_id, product_id, kind, name, insight_cost, effect_text, effect)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.InitiativeID, s.ProductID, s.Kind, s.Name, s.InsightCost, s.EffectText, json.RawMessage(s.Effect))
	require.NoError(t, err)
}

// seedInactiveProduct は is_active=false のプロダクトを挿入する (論理削除の除外検証用)。
func seedInactiveProduct(t *testing.T, s productSeed) {
	t.Helper()
	_, err := sharedPg.Pool.Exec(context.Background(),
		`INSERT INTO card.products (product_id, faction, product_name, is_active)
		 VALUES ($1, $2, $3, false)`,
		s.ProductID, s.Faction, s.ProductName)
	require.NoError(t, err)
}

// seedInactiveInitiative は is_active=false の施策を挿入する (論理削除の除外検証用)。
func seedInactiveInitiative(t *testing.T, s initiativeSeed) {
	t.Helper()
	_, err := sharedPg.Pool.Exec(context.Background(),
		`INSERT INTO card.initiatives
		   (initiative_id, product_id, kind, name, insight_cost, effect_text, effect, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, false)`,
		s.InitiativeID, s.ProductID, s.Kind, s.Name, s.InsightCost, s.EffectText, json.RawMessage(s.Effect))
	require.NoError(t, err)
}

type playerCardSeed struct {
	PlayerID string
	CardID   string
	ArtNo    int64
	Count    int
}

func seedPlayerCard(t *testing.T, s playerCardSeed) {
	t.Helper()
	_, err := sharedPg.Pool.Exec(context.Background(),
		`INSERT INTO card.player_cards (player_id, card_id, art_no, count)
		 VALUES ($1, $2, $3, $4)`,
		s.PlayerID, s.CardID, s.ArtNo, s.Count)
	require.NoError(t, err)
}

// insertDeck は player_id を決めてデッキを 1 つ挿入し、採番された deck_id を返す。
// テストで deck_id を前提にする他テーブル (deck_cards) への seed 用。
func insertDeck(t *testing.T, playerID, deckName string) int64 {
	t.Helper()
	var deckID int64
	err := sharedPg.Pool.QueryRow(context.Background(),
		`INSERT INTO card.decks (player_id, deck_name, faction, product_id, routine_id, special_id) VALUES ($1, $2, 'SHE', 'PD-TST', 'IN-TST-R', 'IN-TST-S') RETURNING deck_id`,
		playerID, deckName).Scan(&deckID)
	require.NoError(t, err)
	return deckID
}

type deckCardSeed struct {
	PlayerID string
	DeckID   int64
	CardID   string
	ArtNo    int64
	Count    int
}

func seedDeckCard(t *testing.T, s deckCardSeed) {
	t.Helper()
	_, err := sharedPg.Pool.Exec(context.Background(),
		`INSERT INTO card.deck_cards (player_id, deck_id, card_id, art_no, count)
		 VALUES ($1, $2, $3, $4, $5)`,
		s.PlayerID, s.DeckID, s.CardID, s.ArtNo, s.Count)
	require.NoError(t, err)
}

// countRows は指定テーブルの行数を返す。CASCADE 動作や tx 境界の検証で使う。
func countRows(t *testing.T, table string) int {
	t.Helper()
	var n int
	err := sharedPg.Pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM "+table).Scan(&n)
	require.NoError(t, err)
	return n
}

// fetchPlayerCardCount は player_cards の指定行の count を返す。UPSERT 検証で使う。
// 行不在は (0, false)、存在は (n, true)。
func fetchPlayerCardCount(t *testing.T, playerID, cardID string, artNo int64) (int, bool) {
	t.Helper()
	var n int
	err := sharedPg.Pool.QueryRow(context.Background(),
		`SELECT count FROM card.player_cards
		   WHERE player_id = $1 AND card_id = $2 AND art_no = $3`,
		playerID, cardID, artNo).Scan(&n)
	switch {
	case err == nil:
		return n, true
	case errors.Is(err, pgx.ErrNoRows):
		return 0, false
	default:
		t.Fatalf("fetchPlayerCardCount: unexpected error: %v", err)
		return 0, false
	}
}

// テスト用プレイヤー UUID。account スキーマとの cross-schema reference は
// app 層整合性なので FK がなく、テストでは任意の UUID を使える。
const (
	playerA = "11111111-1111-1111-1111-111111111111"
	playerB = "22222222-2222-2222-2222-222222222222"
)

// cardPackSeed は card.card_pack への最小シード入力。
type cardPackSeed struct {
	PackID   string
	IsActive bool
}

func seedCardPack(t *testing.T, s cardPackSeed) {
	t.Helper()
	_, err := sharedPg.Pool.Exec(context.Background(),
		`INSERT INTO card.card_pack (pack_id, is_active) VALUES ($1, $2)`,
		s.PackID, s.IsActive)
	require.NoError(t, err)
}

// fullCardPackSeed は card.card_pack への全列シード入力。GetPack の全フィールド
// 写像を検証するテスト用に、省略可能なカラムも含め明示的に指定する。
type fullCardPackSeed struct {
	PackID      string
	Description string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func seedFullCardPack(t *testing.T, s fullCardPackSeed) {
	t.Helper()
	_, err := sharedPg.Pool.Exec(context.Background(),
		`INSERT INTO card.card_pack (pack_id, description, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		s.PackID, s.Description, s.IsActive, s.CreatedAt, s.UpdatedAt)
	require.NoError(t, err)
}

func seedCardPackCard(t *testing.T, packID, cardID string, copies int) {
	t.Helper()
	_, err := sharedPg.Pool.Exec(context.Background(),
		`INSERT INTO card.card_pack_cards (pack_id, card_id, copies) VALUES ($1, $2, $3)`,
		packID, cardID, copies)
	require.NoError(t, err)
}

// fetchProcessedEventType は card.processed_events の指定行の event_type を返す。
// Insert の冪等性 (先勝ち) を DB 直読みで確認するために使う。
func fetchProcessedEventType(t *testing.T, eventID string) string {
	t.Helper()
	var eventType string
	err := sharedPg.Pool.QueryRow(context.Background(),
		`SELECT event_type FROM card.processed_events WHERE event_id = $1`, eventID).Scan(&eventType)
	require.NoError(t, err)
	return eventType
}

// int64Ptr はテストで PlaymatNo / SleeveNo のような *int64 フィールドを
// リテラルから組み立てるためのヘルパー。
func int64Ptr(v int64) *int64 {
	return &v
}
