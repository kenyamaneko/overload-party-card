//go:build integration

package router_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
	"github.com/kenyamaneko/overload-party-card/internal/repository/repositorytest"
	"github.com/kenyamaneko/overload-party-card/internal/router/routertest"
)

var sharedPg *repositorytest.Postgres

func TestMain(m *testing.M) {
	os.Exit(repositorytest.RunMain(m, &sharedPg,
		repositorytest.WithSchemaFile("db/schema.sql"),
		repositorytest.WithSchema("card"),
	))
}

const (
	insertCardDefinitionsSQL = `INSERT INTO card.card_definitions
		(card_id, card_name, resource_label, faction, card_type, resizable, elastic, stats, restriction, is_active)
		VALUES
		('TST-0001', 'テストカードA', 'テストラベル', 'SHE', 'Strategy', false, false, '{}', 'unlimited', true),
		('TST-0002', 'テストカードB', 'テストラベル', 'SHE', 'Strategy', false, false, '{}', 'unlimited', true)`
	insertProductSQL = `INSERT INTO card.products (product_id, faction, product_name)
		VALUES ('TST-0001', 'SHE', 'テストプロダクトA')`
	insertInitiativesSQL = `INSERT INTO card.initiatives
		(initiative_id, product_id, kind, name, insight_cost, effect_text, effect)
		VALUES
		('TST-0011', 'TST-0001', 'routine', 'テスト施策A', 1, 'テスト効果文', '{}'),
		('TST-0012', 'TST-0001', 'special', 'テスト施策B', 1, 'テスト効果文', '{}')`
	insertDeckSQL = `INSERT INTO card.decks
		(player_id, deck_id, deck_name, faction, product_id, routine_id, special_id)
		OVERRIDING SYSTEM VALUE
		VALUES ($1, $2, 'テストデッキA', 'SHE', 'TST-0001', 'TST-0011', 'TST-0012')`
)

func newIntegrationHarness(t *testing.T) *routertest.Harness {
	ctx := context.Background()
	sharedPg.Truncate(t)
	for _, statement := range []string{insertCardDefinitionsSQL, insertProductSQL, insertInitiativesSQL} {
		_, err := sharedPg.Pool.Exec(ctx, statement)
		require.NoError(t, err)
	}
	return &routertest.Harness{
		Logs: &bytes.Buffer{},
		Handler: routertest.NewRouter(t, routertest.Dependencies{
			Cards:             repository.NewPostgresCardRepository(sharedPg.Pool),
			PlayerCards:       repository.NewPostgresPlayerCardRepository(sharedPg.Pool),
			Decks:             repository.NewPostgresDeckRepository(sharedPg.Pool),
			TransactionRunner: repository.NewPostgresGrantTransactionRunner(sharedPg.Pool),
			Factions:          &routertest.FakeFactionClient{Factions: []string{"SHE"}},
			Products:          repository.NewPostgresProductRepository(sharedPg.Pool),
			Initiatives:       repository.NewPostgresInitiativeRepository(sharedPg.Pool),
		}),
	}
}

func ownCard(t *testing.T, playerID, cardID string) {
	err := repository.NewPostgresPlayerCardRepository(sharedPg.Pool).AddCards(context.Background(), []domain.PlayerCard{
		{PlayerID: playerID, CardID: cardID, ArtNo: 0, Count: 1},
	})
	require.NoError(t, err)
}

func insertDeck(t *testing.T, playerID string, deckID int64) {
	_, err := sharedPg.Pool.Exec(context.Background(), insertDeckSQL, playerID, deckID)
	require.NoError(t, err)
}

func TestPlayerCardListIntegration(t *testing.T) {
	t.Run("[所持カード]プレイヤーの所持カード一覧の取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("プレイヤーAがTST-0001、プレイヤーBがTST-0002を所持しているとき、プレイヤーAのトークンで要求すると、応答のcard_idはTST-0001だけになる", func(t *testing.T) {
				h := newIntegrationHarness(t)
				ownCard(t, routertest.PlayerA, "TST-0001")
				ownCard(t, routertest.PlayerB, "TST-0002")

				cards := routertest.DecodeArray(t, h.GetPlayerCards())

				assert.Equal(t, []any{"TST-0001"}, routertest.CollectValues(t, cards, "card_id"))
			})
		})
	})
}

func TestCardOwnershipListIntegration(t *testing.T) {
	t.Run("[カード定義]所持状態付きカード定義一覧の取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("カード定義にTST-0001とTST-0002があり、プレイヤーAのトークンで要求したとき", func(t *testing.T) {
				t.Run("プレイヤーBだけがTST-0002を所持していると、応答のカードTST-0002のis_ownedはfalseになる", func(t *testing.T) {
					h := newIntegrationHarness(t)
					ownCard(t, routertest.PlayerB, "TST-0002")

					cards := routertest.DecodeArray(t, h.SendGetRequest(routertest.CardOwnershipPath))

					assert.Equal(t, false, routertest.FindObject(t, cards, "card_id", "TST-0002")["is_owned"])
				})
			})
		})
	})
}

func TestDeckListIntegration(t *testing.T) {
	t.Run("[デッキ]プレイヤーのデッキ一覧の取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("プレイヤーAがデッキID1とデッキID2、プレイヤーBがデッキID3を持つとき、プレイヤーAのトークンで要求すると、応答のdeck_idは1と2だけになる", func(t *testing.T) {
				h := newIntegrationHarness(t)
				insertDeck(t, routertest.PlayerA, 1)
				insertDeck(t, routertest.PlayerA, 2)
				insertDeck(t, routertest.PlayerB, 3)

				decks := routertest.DecodeArray(t, h.GetDecks())

				assert.ElementsMatch(t, []any{float64(1), float64(2)}, routertest.CollectValues(t, decks, "deck_id"))
			})
		})
	})
}

func TestDeckDetailIntegration(t *testing.T) {
	t.Run("[デッキ]デッキ詳細の取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("プレイヤーAとプレイヤーBがそれぞれデッキID7のデッキを持つとき、プレイヤーAのトークンでデッキID7を要求すると、応答のplayer_idは00000000-0000-0000-0000-000000000001になる", func(t *testing.T) {
				h := newIntegrationHarness(t)
				insertDeck(t, routertest.PlayerA, 7)
				insertDeck(t, routertest.PlayerB, 7)

				deck := routertest.DecodeObject(t, h.GetDeck(7))

				assert.Equal(t, "00000000-0000-0000-0000-000000000001", deck["player_id"])
			})
		})
	})
}
