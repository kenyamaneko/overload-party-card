//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

// TestInitiativeFindAll は、施策が initiative_id 昇順で、親 product_id 付きで返ることを検証する。
func TestInitiativeFindAll(t *testing.T) {
	sharedPg.Truncate(t)
	seedProduct(t, productSeed{"PD-TST1", "SHE", "P1"})
	seedProduct(t, productSeed{"PD-TST2", "Tenki", "P2"})
	seedInitiative(t, initiativeSeed{"IN-TST-S1", "PD-TST1", "special", "S1", 200, "", `{"ops":[]}`})
	seedInitiative(t, initiativeSeed{"IN-TST-R1", "PD-TST1", "routine", "R1", 100, "", `{"ops":[]}`})
	seedInitiative(t, initiativeSeed{"IN-TST-R2", "PD-TST2", "routine", "R2", 100, "", `{"ops":[]}`})

	repo := repository.NewPgInitiativeRepository(sharedPg.Pool)
	got, err := repo.FindAll(context.Background())
	require.NoError(t, err)

	var gotIDs []string
	productByID := map[string]string{}
	for _, i := range got {
		gotIDs = append(gotIDs, i.InitiativeID)
		productByID[i.InitiativeID] = i.ProductID
	}

	assert.Equal(t, []string{"IN-TST-R1", "IN-TST-R2", "IN-TST-S1"}, gotIDs)
	assert.Equal(t, "PD-TST1", productByID["IN-TST-R1"])
	assert.Equal(t, "PD-TST2", productByID["IN-TST-R2"])
}

// TestInitiativeFindAll_ExcludesInactive は is_active=false の施策が除外されることを検証する。
func TestInitiativeFindAll_ExcludesInactive(t *testing.T) {
	sharedPg.Truncate(t)
	seedProduct(t, productSeed{"PD-TST1", "SHE", "P1"})
	seedInitiative(t, initiativeSeed{"IN-TST-R1", "PD-TST1", "routine", "R1", 100, "", `{"ops":[]}`})
	seedInactiveInitiative(t, initiativeSeed{"IN-TST-S9", "PD-TST1", "special", "Retired", 0, "", `{"ops":[]}`})

	repo := repository.NewPgInitiativeRepository(sharedPg.Pool)
	got, err := repo.FindAll(context.Background())
	require.NoError(t, err)

	require.Len(t, got, 1)
	assert.Equal(t, "IN-TST-R1", got[0].InitiativeID)
}
