//go:build integration

package repository_test

import (
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
	"github.com/kenyamaneko/overload-party-card/internal/repository/repositorytest"
)

type initiativeRow struct {
	InitiativeID string
	ProductID    string
	Kind         string
	Name         string
	InsightCost  int64
	EffectText   string
	Effect       string
	IsActive     bool
}

func newInitiativeRow(initiativeID string) initiativeRow {
	return initiativeRow{
		InitiativeID: initiativeID,
		ProductID:    "TST-2001",
		Kind:         "routine",
		Name:         "テスト施策",
		InsightCost:  3,
		EffectText:   "テスト効果",
		Effect:       "{}",
		IsActive:     true,
	}
}

func insertInitiative(t *testing.T, pool *pgxpool.Pool, row initiativeRow) {
	t.Helper()
	_, err := pool.Exec(t.Context(), `
		INSERT INTO card.initiatives (initiative_id, product_id, kind, name, insight_cost, effect_text, effect, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7::text::jsonb, $8)`,
		row.InitiativeID, row.ProductID, row.Kind, row.Name, row.InsightCost, row.EffectText, row.Effect, row.IsActive,
	)
	require.NoError(t, err)
}

func toInitiativeIDs(initiatives []*domain.Initiative) []string {
	initiativeIDs := make([]string, 0, len(initiatives))
	for _, initiative := range initiatives {
		initiativeIDs = append(initiativeIDs, initiative.InitiativeID)
	}
	return initiativeIDs
}

func TestPostgresInitiativeRepositoryFindAll(t *testing.T) {
	t.Run("[施策]有効な施策の一覧取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("TST-1002は有効フラグがtrue、TST-1003は有効フラグがfalseの施策が保存されているとき、取得すると、TST-1002だけが返る", func(t *testing.T) {
				pool := preparePool(t)
				repositorytest.InsertProduct(t, pool, repositorytest.NewProductRow("TST-2001"))
				active := newInitiativeRow("TST-1002")
				active.IsActive = true
				inactive := newInitiativeRow("TST-1003")
				inactive.IsActive = false
				insertInitiative(t, pool, active)
				insertInitiative(t, pool, inactive)

				got, err := repository.NewPostgresInitiativeRepository(pool).FindAll(t.Context())

				require.NoError(t, err)
				assert.Equal(t, []string{"TST-1002"}, toInitiativeIDs(got))
			})

			t.Run("施策がTST-1003、TST-1001、TST-1002の順に保存されているとき、取得すると、TST-1001、TST-1002、TST-1003の順に返る", func(t *testing.T) {
				pool := preparePool(t)
				repositorytest.InsertProduct(t, pool, repositorytest.NewProductRow("TST-2001"))
				insertInitiative(t, pool, newInitiativeRow("TST-1003"))
				insertInitiative(t, pool, newInitiativeRow("TST-1001"))
				insertInitiative(t, pool, newInitiativeRow("TST-1002"))

				got, err := repository.NewPostgresInitiativeRepository(pool).FindAll(t.Context())

				require.NoError(t, err)
				assert.Equal(t, []string{"TST-1001", "TST-1002", "TST-1003"}, toInitiativeIDs(got))
			})

			t.Run("施策が1件も保存されていないとき、取得すると、空の一覧が返る", func(t *testing.T) {
				pool := preparePool(t)

				got, err := repository.NewPostgresInitiativeRepository(pool).FindAll(t.Context())

				require.NoError(t, err)
				assert.Len(t, got, 0)
			})

			t.Run("施策IDがTST-1002、プロダクトIDがTST-2001、区分がroutine、施策名がテスト施策、施策の使用コストが3、効果テキストがテスト効果、効果定義が{\"trigger\":\"on_use\"}、有効フラグがtrueの有効な施策が保存されているとき、取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got *domain.Initiative)
				}{
					{"施策IDがTST-1002になる", func(t *testing.T, got *domain.Initiative) { assert.Equal(t, "TST-1002", got.InitiativeID) }},
					{"プロダクトIDがTST-2001になる", func(t *testing.T, got *domain.Initiative) { assert.Equal(t, "TST-2001", got.ProductID) }},
					{"区分がroutineになる", func(t *testing.T, got *domain.Initiative) { assert.Equal(t, "routine", got.Kind) }},
					{"施策名がテスト施策になる", func(t *testing.T, got *domain.Initiative) { assert.Equal(t, "テスト施策", got.Name) }},
					{"施策の使用コストが3になる", func(t *testing.T, got *domain.Initiative) { assert.Equal(t, int64(3), got.InsightCost) }},
					{"効果テキストがテスト効果になる", func(t *testing.T, got *domain.Initiative) { assert.Equal(t, "テスト効果", got.EffectText) }},
					{"効果定義が{\"trigger\":\"on_use\"}になる", func(t *testing.T, got *domain.Initiative) {
						effect, err := json.Marshal(got.Effect)
						require.NoError(t, err)
						assert.JSONEq(t, `{"trigger":"on_use"}`, string(effect))
					}},
					{"有効フラグがtrueになる", func(t *testing.T, got *domain.Initiative) { assert.True(t, got.IsActive) }},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						pool := preparePool(t)
						repositorytest.InsertProduct(t, pool, repositorytest.NewProductRow("TST-2001"))
						row := newInitiativeRow("TST-1002")
						row.ProductID = "TST-2001"
						row.Kind = "routine"
						row.Name = "テスト施策"
						row.InsightCost = 3
						row.EffectText = "テスト効果"
						row.Effect = `{"trigger":"on_use"}`
						row.IsActive = true
						insertInitiative(t, pool, row)

						got, err := repository.NewPostgresInitiativeRepository(pool).FindAll(t.Context())

						require.NoError(t, err)
						require.Len(t, got, 1)
						tc.check(t, got[0])
					})
				}
			})

			t.Run("区分がspecialの有効な施策が保存されているとき、取得すると、区分がspecialになる", func(t *testing.T) {
				pool := preparePool(t)
				repositorytest.InsertProduct(t, pool, repositorytest.NewProductRow("TST-2001"))
				row := newInitiativeRow("TST-1003")
				row.Kind = "special"
				row.IsActive = true
				insertInitiative(t, pool, row)

				got, err := repository.NewPostgresInitiativeRepository(pool).FindAll(t.Context())

				require.NoError(t, err)
				require.Len(t, got, 1)
				assert.Equal(t, "special", got[0].Kind)
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("DBに接続できないとき、取得すると、DBに接続できないことを示すエラーが返る", func(t *testing.T) {
				pool := repositorytest.NewUnreachablePool(t)

				_, err := repository.NewPostgresInitiativeRepository(pool).FindAll(t.Context())

				repositorytest.RequireConnectionError(t, err)
			})
		})
	})
}
