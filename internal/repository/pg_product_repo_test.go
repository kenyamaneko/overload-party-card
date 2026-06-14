//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

// TestProductFindAll は、プロダクトが product_id 昇順・施策が initiative_id 昇順で
// 親プロダクトに紐づいて組み立てられることを検証する。
func TestProductFindAll(t *testing.T) {
	tests := []struct {
		name        string
		products    []productSeed
		initiatives []initiativeSeed
		wantOrder   []string
		wantByID    map[string][]string
	}{
		{
			name: "product_id 昇順・施策は initiative_id 昇順で組み立てられる",
			products: []productSeed{
				{"PD-TST2", "Tenki", "P2"},
				{"PD-TST1", "SHE", "P1"},
			},
			initiatives: []initiativeSeed{
				{"IN-TST-S1", "PD-TST1", "special", "S1", 200, "", `{"ops":[]}`},
				{"IN-TST-R1", "PD-TST1", "routine", "R1", 100, "", `{"ops":[]}`},
				{"IN-TST-R2", "PD-TST2", "routine", "R2", 100, "", `{"ops":[]}`},
			},
			wantOrder: []string{"PD-TST1", "PD-TST2"},
			wantByID: map[string][]string{
				"PD-TST1": {"IN-TST-R1", "IN-TST-S1"},
				"PD-TST2": {"IN-TST-R2"},
			},
		},
		{
			name:      "空テーブルは空スライス",
			products:  nil,
			wantOrder: nil,
			wantByID:  map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sharedPg.Truncate(t)
			for _, p := range tt.products {
				seedProduct(t, p)
			}
			for _, i := range tt.initiatives {
				seedInitiative(t, i)
			}

			repo := repository.NewPgProductRepository(sharedPg.Pool)
			got, err := repo.FindAll(context.Background())
			require.NoError(t, err)

			var gotOrder []string
			gotByID := map[string][]string{}
			for _, p := range got {
				gotOrder = append(gotOrder, p.ProductID)
				var ids []string
				for _, in := range p.Initiatives {
					ids = append(ids, in.InitiativeID)
				}
				gotByID[p.ProductID] = ids
			}

			assert.Equal(t, tt.wantOrder, gotOrder)
			for pid, wantIDs := range tt.wantByID {
				assert.Equal(t, wantIDs, gotByID[pid], "initiatives of %s", pid)
			}
		})
	}
}

// TestProductFindAll_EffectPreserved は、施策の effect JSONB が往復で保持されることを検証する。
func TestProductFindAll_EffectPreserved(t *testing.T) {
	sharedPg.Truncate(t)
	seedProduct(t, productSeed{"PD-TST1", "SHE", "P1"})
	seedInitiative(t, initiativeSeed{"IN-TST-R1", "PD-TST1", "routine", "R1", 150,
		"効果説明", `{"ops":[{"gain_budget":{"target":"myself","amount":50}}]}`})

	repo := repository.NewPgProductRepository(sharedPg.Pool)
	got, err := repo.FindAll(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].Initiatives, 1)

	in := got[0].Initiatives[0]
	assert.Equal(t, int64(150), in.InsightCost)
	assert.Equal(t, "効果説明", in.EffectText)
	assert.JSONEq(t, `{"ops":[{"gain_budget":{"target":"myself","amount":50}}]}`, string(in.Effect))
}
