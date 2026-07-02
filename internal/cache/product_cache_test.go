package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

// controlProductsJSON は既知プロダクト 2 件の制御フィクスチャ。生成データの並びに
// 依存せず「ID で正しいプロダクト 1 件を引ける」ことを固定する。2 件は faction が
// 異なり、返ったプロダクトが問い合わせ ID に対応する 1 件であることを判別できる。
const controlProductsJSON = `[
	{"product_id":"PD-TST-0001","faction":"SHE","product_name":"制御プロダクトA","is_active":true},
	{"product_id":"PD-TST-0002","faction":"Tuners","product_name":"制御プロダクトB","is_active":true}
]`

// TestProductFindByID は、ID で問い合わせたプロダクトが返ること・未知 ID で nil が返ることを検証します。
func TestProductFindByID(t *testing.T) {
	pc := NewProductCache()
	require.NoError(t, pc.LoadFromBytes([]byte(controlProductsJSON)))

	cases := []struct {
		name string
		id   string
		want *domain.Product
	}{
		{
			name: "既知 ID は対応するプロダクトを返す",
			id:   "PD-TST-0002",
			want: &domain.Product{
				ProductID:   "PD-TST-0002",
				Faction:     "Tuners",
				ProductName: "制御プロダクトB",
				IsActive:    true,
			},
		},
		{
			name: "未知 ID は nil を返す",
			id:   "PD-TST-9999",
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, pc.FindByID(tc.id))
		})
	}
}

// TestProductLoadFromBytes_Empty は、0 件 JSON がマスター欠落としてエラーに
// なることを検証します。
func TestProductLoadFromBytes_Empty(t *testing.T) {
	pc := NewProductCache()
	err := pc.LoadFromBytes([]byte(`[]`))
	assert.Error(t, err)
}
