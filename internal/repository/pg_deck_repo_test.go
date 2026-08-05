//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

func TestCreate(t *testing.T) {
	t.Run("デッキと構成カードの作成", func(t *testing.T) {
		// deck_id は GENERATED ALWAYS AS IDENTITY で自動採番され、cards の deck_id に伝播する。
		tests := []struct {
			name      string
			entries   []domain.DeckCardEntry
			wantCards int
		}{
			{
				name: "cardsありのとき、deckとdeck_cardsが同時に作られる",
				entries: []domain.DeckCardEntry{
					{CardID: "SH-0001", ArtNo: 0, Count: 3},
					{CardID: "SH-0002", ArtNo: 0, Count: 2},
				},
				wantCards: 2,
			},
			{
				name:      "cardsが空のとき、deck行だけが作られる",
				entries:   nil,
				wantCards: 0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				sharedPg.Truncate(t)
				repo := repository.NewPgDeckRepository(sharedPg.Pool)
				ctx := context.Background()

				now := time.Now().UTC().Truncate(time.Microsecond)
				deck := domain.Deck{
					PlayerID:  playerA,
					DeckName:  "Starter",
					Faction:   "SHE",
					ProductID: "PD-TST",
					RoutineID: "IN-TST-R",
					SpecialID: "IN-TST-S",
					CreatedAt: now,
					UpdatedAt: now,
				}

				deckID, err := repo.Create(ctx, deck, tt.entries)
				require.NoError(t, err)
				assert.NotZero(t, deckID, "deck_id should be auto-assigned")

				persisted, err := repo.FindByID(ctx, playerA, deckID)
				require.NoError(t, err)
				assert.Equal(t, "SHE", persisted.Faction)
				assert.Equal(t, "PD-TST", persisted.ProductID)
				assert.Equal(t, "IN-TST-R", persisted.RoutineID)
				assert.Equal(t, "IN-TST-S", persisted.SpecialID)

				got, err := repo.GetDeckCards(ctx, playerA, deckID)
				require.NoError(t, err)
				assert.Len(t, got, tt.wantCards)
				for _, dc := range got {
					assert.Equal(t, deckID, dc.DeckID, "deck_id should propagate to deck_cards")
				}
			})
		}

		t.Run("同一カード・同一アートが重複した構成で作成すると、エラーになりデッキ本体も保存されない", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			ctx := context.Background()

			now := time.Now().UTC().Truncate(time.Microsecond)
			deck := domain.Deck{
				PlayerID:  playerA,
				DeckName:  "Broken",
				Faction:   "SHE",
				ProductID: "PD-TST",
				RoutineID: "IN-TST-R",
				SpecialID: "IN-TST-S",
				CreatedAt: now,
				UpdatedAt: now,
			}

			_, err := repo.Create(ctx, deck, []domain.DeckCardEntry{
				{CardID: "TST-0001", ArtNo: 0, Count: 1},
				{CardID: "TST-0001", ArtNo: 0, Count: 1},
			})

			require.Error(t, err)
			assert.Equal(t, 0, countRows(t, "card.decks"))
			assert.Equal(t, 0, countRows(t, "card.deck_cards"))
		})
	})
}

func TestFindByPlayerID(t *testing.T) {
	t.Run("プレイヤーのデッキ一覧取得", func(t *testing.T) {
		tests := []struct {
			name      string
			setup     func(t *testing.T)
			target    string
			wantNames []string
		}{
			{
				name: "複数デッキのとき、updated_at降順で返る",
				setup: func(t *testing.T) {
					// updated_at は now() デフォルト。手動で 3 件作って順序を制御する
					insertDeckAt(t, playerA, "Old", time.Now().Add(-2*time.Hour))
					insertDeckAt(t, playerA, "Middle", time.Now().Add(-1*time.Hour))
					insertDeckAt(t, playerA, "New", time.Now())
				},
				target:    playerA,
				wantNames: []string{"New", "Middle", "Old"},
			},
			{
				name: "他プレイヤーのデッキがあるとき、除外される",
				setup: func(t *testing.T) {
					insertDeck(t, playerA, "Mine")
					insertDeck(t, playerB, "Theirs")
				},
				target:    playerA,
				wantNames: []string{"Mine"},
			},
			{
				name:      "デッキ0件のとき、空スライスになる",
				setup:     func(t *testing.T) {},
				target:    playerA,
				wantNames: nil,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				sharedPg.Truncate(t)
				tt.setup(t)

				repo := repository.NewPgDeckRepository(sharedPg.Pool)
				got, err := repo.FindByPlayerID(context.Background(), tt.target)
				require.NoError(t, err)

				var gotNames []string
				for _, d := range got {
					gotNames = append(gotNames, d.DeckName)
				}
				assert.Equal(t, tt.wantNames, gotNames)
			})
		}
	})
}

func TestFindByID(t *testing.T) {
	t.Run("deck_id指定の取得", func(t *testing.T) {
		t.Run("自分のdeck_idのとき、デッキ全項目を返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			deckID := insertDeck(t, playerA, "Target Deck")

			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			got, err := repo.FindByID(context.Background(), playerA, deckID)
			require.NoError(t, err)
			assert.Equal(t, "Target Deck", got.DeckName)
			assert.Equal(t, playerA, got.PlayerID)
			assert.Equal(t, "SHE", got.Faction)
			assert.Equal(t, "PD-TST", got.ProductID)
			assert.Equal(t, "IN-TST-R", got.RoutineID)
			assert.Equal(t, "IN-TST-S", got.SpecialID)
		})

		// PK スコープを超えた認可漏れがないことを示す。
		notFoundCases := []struct {
			name     string
			setup    func(t *testing.T) int64
			queryPID string
		}{
			{
				name: "未登録のdeck_idのとき、ErrNotFoundになる",
				setup: func(t *testing.T) int64 {
					return 9999 // 未発行の ID
				},
				queryPID: playerA,
			},
			{
				name: "他プレイヤーのdeck_idのとき、ErrNotFoundになる (PKスコープ)",
				setup: func(t *testing.T) int64 {
					return insertDeck(t, playerB, "Other's Deck")
				},
				queryPID: playerA,
			},
		}

		for _, tt := range notFoundCases {
			t.Run(tt.name, func(t *testing.T) {
				sharedPg.Truncate(t)
				deckID := tt.setup(t)

				repo := repository.NewPgDeckRepository(sharedPg.Pool)
				_, err := repo.FindByID(context.Background(), tt.queryPID, deckID)
				assert.ErrorIs(t, err, port.ErrNotFound)
			})
		}
	})
}

func TestGetDeckCards(t *testing.T) {
	t.Run("デッキ構成カードの取得", func(t *testing.T) {
		tests := []struct {
			name      string
			setup     func(t *testing.T) (string, int64) // target player_id, deck_id
			wantCount int
		}{
			{
				name: "対象デッキにカードがあるとき、全件返る",
				setup: func(t *testing.T) (string, int64) {
					d := insertDeck(t, playerA, "Full")
					seedDeckCard(t, deckCardSeed{playerA, d, "SH-0001", 0, 3})
					seedDeckCard(t, deckCardSeed{playerA, d, "SH-0002", 0, 2})
					return playerA, d
				},
				wantCount: 2,
			},
			{
				name: "deckにcardsが無いとき、空になる",
				setup: func(t *testing.T) (string, int64) {
					d := insertDeck(t, playerA, "Empty")
					return playerA, d
				},
				wantCount: 0,
			},
			{
				name: "他プレイヤーのdeck_idをqueryするとき、空になる (PKスコープ)",
				setup: func(t *testing.T) (string, int64) {
					d := insertDeck(t, playerB, "Theirs")
					seedDeckCard(t, deckCardSeed{playerB, d, "SH-0001", 0, 3})
					return playerA, d
				},
				wantCount: 0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				sharedPg.Truncate(t)
				pid, did := tt.setup(t)

				repo := repository.NewPgDeckRepository(sharedPg.Pool)
				got, err := repo.GetDeckCards(context.Background(), pid, did)
				require.NoError(t, err)
				assert.Len(t, got, tt.wantCount)
			})
		}
	})
}

func TestUpdate(t *testing.T) {
	t.Run("デッキの更新", func(t *testing.T) {
		t.Run("カード構成とdeck_name/playmat_noを更新するとき、差し替えで反映される", func(t *testing.T) {
			// カード構成は既存 deck_cards を全削除してから新規 bulk insert する差し替えセマンティクス。
			sharedPg.Truncate(t)
			deckID := insertDeck(t, playerA, "Original")
			seedDeckCard(t, deckCardSeed{playerA, deckID, "SH-0001", 0, 3})
			seedDeckCard(t, deckCardSeed{playerA, deckID, "SH-0002", 0, 3})

			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			ctx := context.Background()

			newPlaymat := int64(7)
			err := repo.Update(ctx, domain.Deck{
				PlayerID:  playerA,
				DeckID:    deckID,
				DeckName:  "Renamed",
				Faction:   "Tenki",
				ProductID: "PD-TST2",
				RoutineID: "IN-TST-R2",
				SpecialID: "IN-TST-S2",
				PlaymatNo: &newPlaymat,
				UpdatedAt: time.Now(),
			}, []domain.DeckCardEntry{
				{CardID: "SH-0003", ArtNo: 0, Count: 3},
			})
			require.NoError(t, err)

			got, err := repo.FindByID(ctx, playerA, deckID)
			require.NoError(t, err)
			assert.Equal(t, "Renamed", got.DeckName)
			assert.Equal(t, "Tenki", got.Faction)
			assert.Equal(t, "PD-TST2", got.ProductID, "product can be updated")
			assert.Equal(t, "IN-TST-R2", got.RoutineID, "routine can be updated")
			assert.Equal(t, "IN-TST-S2", got.SpecialID, "special can be updated")
			require.NotNil(t, got.PlaymatNo)
			assert.Equal(t, newPlaymat, *got.PlaymatNo)

			cards, err := repo.GetDeckCards(ctx, playerA, deckID)
			require.NoError(t, err)
			require.Len(t, cards, 1, "old deck_cards should be replaced")
			assert.Equal(t, "SH-0003", cards[0].CardID)
		})
	})
}

func TestDelete(t *testing.T) {
	t.Run("デッキの削除", func(t *testing.T) {
		tests := []struct {
			name           string
			setup          func(t *testing.T) (int64, int64) // target deck_id, other deck_id
			deletePID      string
			deleteDID      func(targetID int64) int64
			wantDecksAfter int
			wantCardsAfter int
		}{
			{
				name: "自分のデッキを削除するとき、cardsもCASCADEで削除される",
				setup: func(t *testing.T) (int64, int64) {
					d := insertDeck(t, playerA, "Mine")
					seedDeckCard(t, deckCardSeed{playerA, d, "SH-0001", 0, 3})
					return d, 0
				},
				deletePID:      playerA,
				deleteDID:      func(d int64) int64 { return d },
				wantDecksAfter: 0,
				wantCardsAfter: 0,
			},
			{
				name: "他プレイヤーのdeck_idを指定するとき、削除されない",
				setup: func(t *testing.T) (int64, int64) {
					mine := insertDeck(t, playerA, "Mine")
					theirs := insertDeck(t, playerB, "Theirs")
					seedDeckCard(t, deckCardSeed{playerB, theirs, "SH-0001", 0, 3})
					return mine, theirs
				},
				deletePID:      playerA,
				deleteDID:      func(_ int64) int64 { return 999 }, // 存在しない deck_id
				wantDecksAfter: 2,
				wantCardsAfter: 1,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				sharedPg.Truncate(t)
				targetID, _ := tt.setup(t)

				repo := repository.NewPgDeckRepository(sharedPg.Pool)
				err := repo.Delete(context.Background(), tt.deletePID, tt.deleteDID(targetID))
				require.NoError(t, err)

				assert.Equal(t, tt.wantDecksAfter, countRows(t, "card.decks"))
				assert.Equal(t, tt.wantCardsAfter, countRows(t, "card.deck_cards"))
			})
		}
	})
}

// insertDeckAt は updated_at を明示して deck を挿入する。FindByPlayerID の
// 順序確認テストで時刻順序を制御するために使う。
func insertDeckAt(t *testing.T, playerID, deckName string, updatedAt time.Time) int64 {
	t.Helper()
	var deckID int64
	err := sharedPg.Pool.QueryRow(context.Background(),
		`INSERT INTO card.decks (player_id, deck_name, faction, product_id, routine_id, special_id, created_at, updated_at)
		 VALUES ($1, $2, 'SHE', 'PD-TST', 'IN-TST-R', 'IN-TST-S', $3, $3) RETURNING deck_id`,
		playerID, deckName, updatedAt).Scan(&deckID)
	require.NoError(t, err)
	return deckID
}
