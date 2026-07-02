package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

// controlInitiativesJSON は既知施策 2 件の制御フィクスチャ。生成データの並びに
// 依存せず「ID で正しい施策 1 件を引ける」ことを固定する。2 件は kind / name が
// 異なり、返った施策が問い合わせ ID に対応する 1 件であることを判別できる。
const controlInitiativesJSON = `[
	{"initiative_id":"IN-TST-0001","product_id":"PD-TST-0001","kind":"routine","name":"制御施策A","is_active":true},
	{"initiative_id":"IN-TST-0002","product_id":"PD-TST-0001","kind":"special","name":"制御施策B","is_active":true}
]`

// TestInitiativeFindByID は、ID で問い合わせた施策が返ること・未知 ID で nil が返ることを検証します。
func TestInitiativeFindByID(t *testing.T) {
	ic := NewInitiativeCache()
	require.NoError(t, ic.LoadFromBytes([]byte(controlInitiativesJSON)))

	cases := []struct {
		name string
		id   string
		want *domain.Initiative
	}{
		{
			name: "既知 ID は対応する施策を返す",
			id:   "IN-TST-0001",
			want: &domain.Initiative{
				InitiativeID: "IN-TST-0001",
				ProductID:    "PD-TST-0001",
				Kind:         "routine",
				Name:         "制御施策A",
				IsActive:     true,
			},
		},
		{
			name: "未知 ID は nil を返す",
			id:   "IN-TST-9999",
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ic.FindByID(tc.id))
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
