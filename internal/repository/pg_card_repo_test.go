package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

// TestPgCardRepository_FindAll は is_active=true のカードだけが card_id 昇順で
// 返ることを、データの母集団を変えた複数ケースで検証する。
func TestPgCardRepository_FindAll(t *testing.T) {
	tests := []struct {
		name    string
		seeds   []cardSeed
		wantIDs []string
	}{
		{
			name: "active のみ昇順で返る / inactive は除外",
			seeds: []cardSeed{
				{"SH-0002", "SHE B", "SHE", "Compute", "unlimited", true},
				{"SH-0001", "SHE A", "SHE", "Compute", "unlimited", true},
				{"SH-0099", "SHE Inactive", "SHE", "Compute", "unlimited", false},
			},
			wantIDs: []string{"SH-0001", "SH-0002"},
		},
		{
			name:    "空テーブルは空スライス",
			seeds:   nil,
			wantIDs: nil,
		},
		{
			name: "全て inactive なら空スライス",
			seeds: []cardSeed{
				{"SH-0001", "Dormant A", "SHE", "Compute", "unlimited", false},
				{"SH-0002", "Dormant B", "SHE", "Compute", "unlimited", false},
			},
			wantIDs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCards(t, tt.seeds)

			repo := repository.NewPgCardRepository(sharedPg.Pool)
			got, err := repo.FindAll(context.Background())
			require.NoError(t, err)

			var gotIDs []string
			for _, c := range got {
				gotIDs = append(gotIDs, c.CardID)
			}
			assert.Equal(t, tt.wantIDs, gotIDs)
		})
	}
}

// TestPgCardRepository_FindCardIDsByFactions は指定 faction 群の active カードの
// card_id だけが昇順で返ることを検証する。配布対象をハードコードせず
// card_definitions 由来にする契約を仕様として確認する。
func TestPgCardRepository_FindCardIDsByFactions(t *testing.T) {
	allSeeds := []cardSeed{
		{"SH-0001", "SHE A", "SHE", "Compute", "unlimited", true},
		{"SH-0002", "SHE B", "SHE", "Compute", "unlimited", true},
		{"SH-0003", "SHE Inactive", "SHE", "Compute", "unlimited", false},
		{"TK-0001", "Tenki A", "Tenki", "Compute", "unlimited", true},
		{"NU-0001", "Neutral A", "Neutral", "Compute", "unlimited", true},
	}

	tests := []struct {
		name     string
		factions []string
		wantIDs  []string
	}{
		{
			name:     "単一 faction: SHE の active カードのみ",
			factions: []string{"SHE"},
			wantIDs:  []string{"SH-0001", "SH-0002"},
		},
		{
			name:     "複数 faction: SHE + Neutral",
			factions: []string{"SHE", "Neutral"},
			wantIDs:  []string{"NU-0001", "SH-0001", "SH-0002"},
		},
		{
			name:     "存在しない faction は空",
			factions: []string{"Sugar"},
			wantIDs:  []string{},
		},
		{
			name:     "空 faction list は nil",
			factions: nil,
			wantIDs:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCards(t, allSeeds)

			repo := repository.NewPgCardRepository(sharedPg.Pool)
			got, err := repo.FindCardIDsByFactions(context.Background(), tt.factions)
			require.NoError(t, err)
			assert.Equal(t, tt.wantIDs, got)
		})
	}
}
