//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

func TestFindAll(t *testing.T) {
	t.Run("カード一覧の取得", func(t *testing.T) {
		tests := []struct {
			name    string
			seeds   []cardSeed
			wantIDs []string
		}{
			{
				name: "active と inactive が混在するとき、active のみ card_id 昇順で返る",
				seeds: []cardSeed{
					{"SH-0002", "SHE B", "SHE", "Compute", "unlimited", true},
					{"SH-0001", "SHE A", "SHE", "Compute", "unlimited", true},
					{"SH-0099", "SHE Inactive", "SHE", "Compute", "unlimited", false},
				},
				wantIDs: []string{"SH-0001", "SH-0002"},
			},
			{
				name:    "空テーブルのとき、空スライスになる",
				seeds:   nil,
				wantIDs: nil,
			},
			{
				name: "全て inactive のとき、空スライスになる",
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
	})
}
