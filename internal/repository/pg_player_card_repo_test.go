//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

type ownedRow struct {
	cardID string
	artNo  int64
	count  int
}

func TestGetPlayerCards(t *testing.T) {
	t.Run("所持カード一覧の取得", func(t *testing.T) {
		tests := []struct {
			name   string
			seeds  []playerCardSeed
			target string
			want   []ownedRow
		}{
			{
				name: "複数所持のとき、card_id / art_no昇順で返る",
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
				name: "他プレイヤーの行があるとき、除外される (PKスコープ)",
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
				name:   "未所持プレイヤーのとき、空スライスになる",
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
	})
}

func TestAddCards(t *testing.T) {
	t.Run("所持カードのUPSERT加算", func(t *testing.T) {
		bulkCards, bulkExpected := bulkCardPackCards(30, 3)

		tests := []struct {
			name           string
			seeds          []playerCardSeed
			cards          []domain.CardPackCard
			wantGranted    int
			wantFinalCount map[string]int // card_id (art_no=0) → 期待 count
		}{
			{
				name:  "全て新規カードのとき、INSERTされる",
				seeds: nil,
				cards: []domain.CardPackCard{
					{CardID: "SH-0001", Copies: 3},
					{CardID: "SH-0002", Copies: 3},
				},
				wantGranted:    6,
				wantFinalCount: map[string]int{"SH-0001": 3, "SH-0002": 3},
			},
			{
				name: "既所持カードのとき、countに加算される (UPSERT)",
				seeds: []playerCardSeed{
					{playerA, "SH-0001", 0, 2},
				},
				cards:          []domain.CardPackCard{{CardID: "SH-0001", Copies: 3}},
				wantGranted:    3,
				wantFinalCount: map[string]int{"SH-0001": 5},
			},
			{
				name: "カードごとに枚数が異なるとき、それぞれ一括加算される",
				seeds: []playerCardSeed{
					{playerA, "SH-0001", 0, 1},
				},
				cards: []domain.CardPackCard{
					{CardID: "SH-0001", Copies: 3},
					{CardID: "SH-0002", Copies: 1},
				},
				wantGranted:    4,
				wantFinalCount: map[string]int{"SH-0001": 4, "SH-0002": 1},
			},
			{
				name:           "30種類 (実grantスケール)のとき、1文のbulk UPSERTで投入される",
				seeds:          nil,
				cards:          bulkCards,
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
				got, err := repo.AddCards(context.Background(), playerA, tt.cards)
				require.NoError(t, err)
				assert.Equal(t, tt.wantGranted, got)

				for cid, wantCount := range tt.wantFinalCount {
					gotCount, ok := fetchPlayerCardCount(t, playerA, cid, 0)
					assert.Truef(t, ok, "expected row for card_id=%s", cid)
					assert.Equalf(t, wantCount, gotCount, "final count for card_id=%s", cid)
				}
			})
		}

		t.Run("別アート(art_no=1)のみ所持するカードを配布すると、加算ではなく通常アート(art_no=0)の行が新規作成される", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedPlayerCard(t, playerCardSeed{playerA, "TST-0001", 1, 2})

			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)
			got, err := repo.AddCards(context.Background(), playerA, []domain.CardPackCard{{CardID: "TST-0001", Copies: 3}})
			require.NoError(t, err)
			assert.Equal(t, 3, got)

			count0, ok0 := fetchPlayerCardCount(t, playerA, "TST-0001", 0)
			assert.True(t, ok0)
			assert.Equal(t, 3, count0)

			count1, ok1 := fetchPlayerCardCount(t, playerA, "TST-0001", 1)
			assert.True(t, ok1)
			assert.Equal(t, 2, count1)
		})
	})
}
