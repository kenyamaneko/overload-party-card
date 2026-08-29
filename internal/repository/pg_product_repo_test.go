//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

func TestProductFindAll(t *testing.T) {
	t.Run("[プロダクトリポジトリ]プロダクト一覧の取得", func(t *testing.T) {
		orderingCases := []struct {
			name      string
			products  []productSeed
			wantOrder []string
		}{
			{
				name: "複数プロダクトのとき、product_id昇順で返る",
				products: []productSeed{
					{"PD-TST2", "Tenki", "P2"},
					{"PD-TST1", "SHE", "P1"},
				},
				wantOrder: []string{"PD-TST1", "PD-TST2"},
			},
			{
				name:      "空テーブルのとき、空スライスになる",
				products:  nil,
				wantOrder: nil,
			},
		}

		for _, tt := range orderingCases {
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

		t.Run("is_active=falseのプロダクトがあるとき、除外される", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedProduct(t, productSeed{"PD-TST1", "SHE", "P1"})
			seedInactiveProduct(t, productSeed{"PD-TST9", "SHE", "Retired"})

			repo := repository.NewPgProductRepository(sharedPg.Pool)
			got, err := repo.FindAll(context.Background())
			require.NoError(t, err)

			require.Len(t, got, 1)
			assert.Equal(t, "PD-TST1", got[0].ProductID)
		})
	})
}
