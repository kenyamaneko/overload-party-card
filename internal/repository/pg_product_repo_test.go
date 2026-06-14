//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

// TestProductFindAll は、プロダクトが product_id 昇順で返ることを検証する。
func TestProductFindAll(t *testing.T) {
	tests := []struct {
		name      string
		products  []productSeed
		wantOrder []string
	}{
		{
			name: "product_id 昇順で返る",
			products: []productSeed{
				{"PD-TST2", "Tenki", "P2"},
				{"PD-TST1", "SHE", "P1"},
			},
			wantOrder: []string{"PD-TST1", "PD-TST2"},
		},
		{
			name:      "空テーブルは空スライス",
			products:  nil,
			wantOrder: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sharedPg.Truncate(t)
			for _, p := range tt.products {
				seedProduct(t, p)
			}

			repo := repository.NewPgProductRepository(sharedPg.Pool)
			got, err := repo.FindAll(context.Background())
			require.NoError(t, err)

			var gotOrder []string
			for _, p := range got {
				gotOrder = append(gotOrder, p.ProductID)
			}
			assert.Equal(t, tt.wantOrder, gotOrder)
		})
	}
}
