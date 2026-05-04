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

// TestCreate はデッキヘッダと deck_cards を同一 tx で書き込む仕様を検証する。
// - deck_id は GENERATED ALWAYS AS IDENTITY で自動採番される
// - 渡された cards の deck_id は自動で埋められる
// - cards が空でも deck 行だけが作られる
func TestCreate(t *testing.T) {
	tests := []struct {
		name      string
		entries   []domain.DeckCardEntry
		wantCards int
	}{
		{
			name: "deck + 複数 cards を同時に作成",
			entries: []domain.DeckCardEntry{
				{CardID: "SH-0001", ArtNo: 0, Count: 3},
				{CardID: "SH-0002", ArtNo: 0, Count: 2},
			},
			wantCards: 2,
		},
		{
			name:      "cards 空でも deck は作られる",
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
				CreatedAt: now,
				UpdatedAt: now,
			}

			deckID, err := repo.Create(ctx, deck, tt.entries)
			require.NoError(t, err)
			assert.NotZero(t, deckID, "deck_id should be auto-assigned")

			got, err := repo.GetDeckCards(ctx, playerA, deckID)
			require.NoError(t, err)
			assert.Len(t, got, tt.wantCards)
			for _, dc := range got {
				assert.Equal(t, deckID, dc.DeckID, "deck_id should propagate to deck_cards")
			}
		})
	}
}

// TestFindByPlayerID はプレイヤーのデッキが updated_at 降順で返ること、
// 他プレイヤーのデッキは含まれない PK スコープを検証する。
func TestFindByPlayerID(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) []string // 戻り値: 期待される deck_name 順序 (player_A 視点、updated_at DESC)
		target    string
		wantNames []string
	}{
		{
			name: "複数デッキは updated_at 降順で返る",
			setup: func(t *testing.T) []string {
				// updated_at は now() デフォルト。手動で 3 件作って順序を制御する
				insertDeckAt(t, playerA, "Old", time.Now().Add(-2*time.Hour))
				insertDeckAt(t, playerA, "Middle", time.Now().Add(-1*time.Hour))
				insertDeckAt(t, playerA, "New", time.Now())
				return nil
			},
			target:    playerA,
			wantNames: []string{"New", "Middle", "Old"},
		},
		{
			name: "他プレイヤーのデッキは除外される",
			setup: func(t *testing.T) []string {
				insertDeck(t, playerA, "Mine")
				insertDeck(t, playerB, "Theirs")
				return nil
			},
			target:    playerA,
			wantNames: []string{"Mine"},
		},
		{
			name:      "デッキ 0 件なら空スライス",
			setup:     func(t *testing.T) []string { return nil },
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
}

// TestFindByID_Success は (player_id, deck_id) で絞った取得が成功する仕様を検証する。
func TestFindByID_Success(t *testing.T) {
	sharedPg.Truncate(t)
	deckID := insertDeck(t, playerA, "Target Deck")

	repo := repository.NewPgDeckRepository(sharedPg.Pool)
	got, err := repo.FindByID(context.Background(), playerA, deckID)
	require.NoError(t, err)
	assert.Equal(t, "Target Deck", got.DeckName)
	assert.Equal(t, playerA, got.PlayerID)
}

// TestFindByID_NotFound は存在しない / 他プレイヤー配下の deck_id が
// 全て ErrNotFound で返る仕様を検証する。PK スコープを超えた認可漏れがないことを示す。
func TestFindByID_NotFound(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) int64
		queryPID string
	}{
		{
			name: "未登録の deck_id は ErrNotFound",
			setup: func(t *testing.T) int64 {
				return 9999 // 未発行の ID
			},
			queryPID: playerA,
		},
		{
			name: "他プレイヤーの deck_id は ErrNotFound (PK スコープ)",
			setup: func(t *testing.T) int64 {
				return insertDeck(t, playerB, "Other's Deck")
			},
			queryPID: playerA,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sharedPg.Truncate(t)
			deckID := tt.setup(t)

			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			_, err := repo.FindByID(context.Background(), tt.queryPID, deckID)
			assert.ErrorIs(t, err, port.ErrNotFound)
		})
	}
}

// TestGetDeckCards は (player_id, deck_id) スコープで deck_cards を
// 返す仕様を検証する。
func TestGetDeckCards(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) (string, int64) // target player_id, deck_id
		wantCount int
	}{
		{
			name: "対象デッキのカードが全件返る",
			setup: func(t *testing.T) (string, int64) {
				d := insertDeck(t, playerA, "Full")
				seedDeckCard(t, deckCardSeed{playerA, d, "SH-0001", 0, 3})
				seedDeckCard(t, deckCardSeed{playerA, d, "SH-0002", 0, 2})
				return playerA, d
			},
			wantCount: 2,
		},
		{
			name: "deck に cards がなければ空",
			setup: func(t *testing.T) (string, int64) {
				d := insertDeck(t, playerA, "Empty")
				return playerA, d
			},
			wantCount: 0,
		},
		{
			name: "他プレイヤーの deck_id を query しても空 (PK スコープ)",
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
}

// TestUpdate はカード構成が「差し替え」セマンティクスで書き換わること、
// deck_name / playmat_no が更新されることを検証する。
// 既存 deck_cards を全削除してから新規 bulk insert する実装の仕様。
func TestUpdate(t *testing.T) {
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
		PlaymatNo: &newPlaymat,
		UpdatedAt: time.Now(),
	}, []domain.DeckCardEntry{
		{CardID: "SH-0003", ArtNo: 0, Count: 3},
	})
	require.NoError(t, err)

	got, err := repo.FindByID(ctx, playerA, deckID)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", got.DeckName)
	require.NotNil(t, got.PlaymatNo)
	assert.Equal(t, newPlaymat, *got.PlaymatNo)

	cards, err := repo.GetDeckCards(ctx, playerA, deckID)
	require.NoError(t, err)
	require.Len(t, cards, 1, "old deck_cards should be replaced")
	assert.Equal(t, "SH-0003", cards[0].CardID)
}

// TestDelete はデッキ削除で deck_cards も CASCADE 削除される仕様と、
// 他プレイヤーのデッキには影響しない PK スコープを検証する。
func TestDelete(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(t *testing.T) (int64, int64) // target deck_id, other deck_id
		deletePID       string
		deleteDID       func(targetID int64) int64
		wantDecksAfter  int
		wantCardsAfter  int
	}{
		{
			name: "自分のデッキは cards も CASCADE で削除される",
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
			name: "他プレイヤーの deck_id は削除されない",
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
}

// insertDeckAt は updated_at を明示して deck を挿入する。FindByPlayerID の
// 順序確認テストで時刻順序を制御するために使う。
func insertDeckAt(t *testing.T, playerID, deckName string, updatedAt time.Time) int64 {
	t.Helper()
	var deckID int64
	err := sharedPg.Pool.QueryRow(context.Background(),
		`INSERT INTO card.decks (player_id, deck_name, created_at, updated_at)
		 VALUES ($1, $2, $3, $3) RETURNING deck_id`,
		playerID, deckName, updatedAt).Scan(&deckID)
	require.NoError(t, err)
	return deckID
}
