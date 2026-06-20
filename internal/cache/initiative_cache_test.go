package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gencache "github.com/kenyamaneko/overload-party-card/data/cache"
)

func loadTestInitiativeCache(t *testing.T) *InitiativeCache {
	t.Helper()
	ic := NewInitiativeCache()
	err := ic.LoadFromBytes(gencache.InitiativesJSON)
	require.NoError(t, err, "LoadFromBytes failed")
	return ic
}

// TestInitiativeFindByID は、ID で施策を引けること・未知 ID で nil が返ることを検証します。
func TestInitiativeFindByID(t *testing.T) {
	ic := loadTestInitiativeCache(t)
	existingID := ic.All()[0].InitiativeID

	tests := []struct {
		name         string
		initiativeID string
		wantNil      bool
	}{
		{"existing id returns an initiative", existingID, false},
		{"unknown id returns nil", "IN-NOPE", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantNil, ic.FindByID(tt.initiativeID) == nil)
		})
	}
}

// TestInitiativeLoadFromBytes_Empty は、0 件 JSON がマスター欠落としてエラーに
// なることを検証します。
func TestInitiativeLoadFromBytes_Empty(t *testing.T) {
	ic := NewInitiativeCache()
	err := ic.LoadFromBytes([]byte(`[]`))
	assert.Error(t, err)
}
