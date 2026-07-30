//go:build integration

package router

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/repository/postgrestest"
)

// sharedPg はパッケージ全体で共有する Postgres コンテナ。
// TestMain で起動し、各テストは Truncate で状態リセットして使う。
var sharedPg *postgrestest.Postgres

func TestMain(m *testing.M) {
	os.Exit(postgrestest.RunMain(m, &sharedPg,
		postgrestest.WithSchemaFile("db/schema.sql"),
		postgrestest.WithSchema("card"),
	))
}

// fakeFactionClient は router 結合テスト用の port.FactionClient 最小 fake。
// デッキ関連 handler を結線するために router.New が要求するが、本パッケージの
// 結合テストはマスター配信・所持一覧のみを対象とするため常に空スライスを返す。
type fakeFactionClient struct{}

var _ port.FactionClient = fakeFactionClient{}

func (fakeFactionClient) ListPlayerFactions(context.Context, string) ([]string, error) {
	return nil, nil
}

// cardSeed は card.card_definitions への最小シード入力。
type cardSeed struct {
	CardID      string
	CardName    string
	Faction     string
	CardType    string
	Restriction string
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

// playerCardSeed は card.player_cards への最小シード入力。
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
