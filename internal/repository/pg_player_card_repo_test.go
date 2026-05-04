//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

type ownedRow struct {
	cardID string
	artNo  int64
	count  int
}

// TestGetPlayerCards は所持カードが (card_id, art_no) 昇順で
// プレイヤー単位に返ることを検証する。他プレイヤーのカードは含まれない PK スコープも確認。
func TestGetPlayerCards(t *testing.T) {
	tests := []struct {
		name   string
		seeds  []playerCardSeed
		target string
		want   []ownedRow
	}{
		{
			name: "card_id / art_no 昇順で返る",
			seeds: []playerCardSeed{
				{playerA, "SH-0002", 0, 3},
				{playerA, "SH-0001", 1, 1},
				{playerA, "SH-0001", 0, 2},
			},
			target: playerA,
			want: []ownedRow{
				{"SH-0001", 0, 2},
				{"SH-0001", 1, 1},
				{"SH-0002", 0, 3},
			},
		},
		{
			name: "他プレイヤーの行は除外される (PK スコープ)",
			seeds: []playerCardSeed{
				{playerA, "SH-0001", 0, 1},
				{playerB, "SH-0002", 0, 5},
			},
			target: playerA,
			want: []ownedRow{
				{"SH-0001", 0, 1},
			},
		},
		{
			name:   "未所持プレイヤーは空スライス",
			seeds:  []playerCardSeed{{playerB, "SH-0001", 0, 1}},
			target: playerA,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sharedPg.Truncate(t)
			for _, s := range tt.seeds {
				seedPlayerCard(t, s)
			}

			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)
			got, err := repo.GetPlayerCards(context.Background(), tt.target)
			require.NoError(t, err)

			var gotRows []ownedRow
			for _, pc := range got {
				gotRows = append(gotRows, ownedRow{pc.CardID, pc.ArtNo, pc.Count})
			}
			assert.Equal(t, tt.want, gotRows)
		})
	}
}

// TestAddCards は UPSERT-with-add 仕様を検証する。
// 未所持カードは INSERT（count=copiesPerCard）、既所持カードは count 加算。
// art_no は 0 固定 (配布 API の仕様)。
func TestAddCards(t *testing.T) {
	bulkIDs, bulkExpected := bulkScale(30, 3)

	tests := []struct {
		name           string
		seeds          []playerCardSeed
		cardIDs        []string
		copiesPerCard  int
		wantGranted    int
		wantFinalCount map[string]int // card_id (art_no=0) → 期待 count
	}{
		{
			name:           "全て新規カードは INSERT される",
			seeds:          nil,
			cardIDs:        []string{"SH-0001", "SH-0002"},
			copiesPerCard:  3,
			wantGranted:    6,
			wantFinalCount: map[string]int{"SH-0001": 3, "SH-0002": 3},
		},
		{
			name: "既所持カードは count に加算される (UPSERT)",
			seeds: []playerCardSeed{
				{playerA, "SH-0001", 0, 2},
			},
			cardIDs:        []string{"SH-0001"},
			copiesPerCard:  3,
			wantGranted:    3,
			wantFinalCount: map[string]int{"SH-0001": 5},
		},
		{
			name: "新規と既所持が混在するケース",
			seeds: []playerCardSeed{
				{playerA, "SH-0001", 0, 1},
			},
			cardIDs:        []string{"SH-0001", "SH-0002"},
			copiesPerCard:  3,
			wantGranted:    6,
			wantFinalCount: map[string]int{"SH-0001": 4, "SH-0002": 3},
		},
		{
			name:           "実 grant スケール (30 種類) を 1 文の bulk UPSERT で投入する",
			seeds:          nil,
			cardIDs:        bulkIDs,
			copiesPerCard:  3,
			wantGranted:    90,
			wantFinalCount: bulkExpected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sharedPg.Truncate(t)
			for _, s := range tt.seeds {
				seedPlayerCard(t, s)
			}

			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)
			got, err := repo.AddCards(context.Background(), playerA, tt.cardIDs, tt.copiesPerCard)
			require.NoError(t, err)
			assert.Equal(t, tt.wantGranted, got)

			for cid, wantCount := range tt.wantFinalCount {
				gotCount, ok := fetchPlayerCardCount(t, playerA, cid, 0)
				assert.Truef(t, ok, "expected row for card_id=%s", cid)
				assert.Equalf(t, wantCount, gotCount, "final count for card_id=%s", cid)
			}
		})
	}
}

