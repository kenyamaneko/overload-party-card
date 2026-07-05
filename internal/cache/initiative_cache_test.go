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

func TestInitiativeFindByID(t *testing.T) {
	t.Run("施策の ID 検索", func(t *testing.T) {
		ic := loadTestInitiativeCache(t)
		existingID := ic.All()[0].InitiativeID

		tests := []struct {
			name         string
			initiativeID string
			wantNil      bool
		}{
			{"既知の ID のとき、施策を引ける", existingID, false},
			{"未知の ID のとき、nil を返す", "IN-NOPE", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.wantNil, ic.FindByID(tt.initiativeID) == nil)
			})
		}
	})
}

func TestInitiativeLoadFromBytes(t *testing.T) {
	t.Run("施策キャッシュのロード", func(t *testing.T) {
		t.Run("0 件 JSON のとき、マスター欠落としてエラーになる", func(t *testing.T) {
			ic := NewInitiativeCache()
			err := ic.LoadFromBytes([]byte(`[]`))
			assert.Error(t, err)
		})
	})
}
