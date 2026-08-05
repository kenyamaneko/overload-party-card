//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

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
				name: "activeとinactiveが混在するとき、activeのみcard_id昇順で返る",
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
				name: "全てinactiveのとき、空スライスになる",
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

		t.Run("全項目を持つカードを1行投入すると、全フィールドが投入値どおりに取得される", func(t *testing.T) {
			sharedPg.Truncate(t)
			ts := time.Now().UTC().Truncate(time.Microsecond)
			seedFullCard(t, fullCardSeed{
				CardID: "TST-0001", CardName: "Full Card", ResourceLabel: "vCPU", Faction: "SHE",
				CardType: "Compute", Subtype: "VM", Resizable: true, Elastic: true,
				Stats:       `{"throughput":10,"availability":1,"maintenance_cost":1,"sla_penalty":1}`,
				EffectText:  "テスト効果",
				Effects:     `{"ops":[]}`,
				Restriction: "limited", IsActive: true, CreatedAt: ts, UpdatedAt: ts,
			})

			repo := repository.NewPgCardRepository(sharedPg.Pool)
			got, err := repo.FindAll(context.Background())
			require.NoError(t, err)
			require.Len(t, got, 1)

			c := got[0]
			assert.Equal(t, "TST-0001", c.CardID)
			assert.Equal(t, "Full Card", c.CardName)
			assert.Equal(t, "vCPU", c.ResourceLabel)
			assert.Equal(t, "SHE", c.Faction)
			assert.Equal(t, "Compute", c.CardType)
			require.NotNil(t, c.Subtype)
			assert.Equal(t, "VM", *c.Subtype)
			assert.True(t, c.Resizable)
			assert.True(t, c.Elastic)
			assert.JSONEq(t, `{"throughput":10,"availability":1,"maintenance_cost":1,"sla_penalty":1}`, string(c.Stats))
			require.NotNil(t, c.EffectText)
			assert.Equal(t, "テスト効果", *c.EffectText)
			assert.JSONEq(t, `{"ops":[]}`, string(c.Effects))
			assert.Equal(t, "limited", c.Restriction)
			assert.True(t, c.IsActive)
			assert.True(t, ts.Equal(c.CreatedAt))
			assert.True(t, ts.Equal(c.UpdatedAt))
		})
	})
}
