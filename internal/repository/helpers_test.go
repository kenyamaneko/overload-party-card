//go:build integration

package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
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
		`INSERT INTO card.decks (player_id, deck_name) VALUES ($1, $2) RETURNING deck_id`,
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
	}
	t.Fatalf("fetchPlayerCardCount: unexpected error: %v", err)
	return 0, false
}

// テスト用プレイヤー UUID。account スキーマとの cross-schema reference は
// app 層整合性なので FK がなく、テストでは任意の UUID を使える。
const (
	playerA = "11111111-1111-1111-1111-111111111111"
	playerB = "22222222-2222-2222-2222-222222222222"
)

// bulkCardPackCards は AddCards の bulk UPSERT を実 grant スケール (数十枚) で
// 検証するためのフィクスチャを生成する。n 件の (card_id, copies) ペアと
// 「全件 copies コピーずつ」の期待値マップをペアで返す。
func bulkCardPackCards(n, copies int) ([]domain.CardPackCard, map[string]int) {
	cards := make([]domain.CardPackCard, n)
	expected := make(map[string]int, n)
	for i := range cards {
		id := fmt.Sprintf("BK-%04d", i+1)
		cards[i] = domain.CardPackCard{CardID: id, Copies: copies}
		expected[id] = copies
	}
	return cards, expected
}
