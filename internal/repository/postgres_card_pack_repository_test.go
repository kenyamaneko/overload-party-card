//go:build integration

package repository_test

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

func insertCardPack(t *testing.T, pool *pgxpool.Pool, packID, description string, isActive bool) {
	t.Helper()
	_, err := pool.Exec(t.Context(), `
		INSERT INTO card.card_pack (pack_id, description, is_active)
		VALUES ($1, $2, $3)`,
		packID, description, isActive,
	)
	require.NoError(t, err)
}

func insertCardPackCard(t *testing.T, pool *pgxpool.Pool, packID, cardID string, copies int) {
	t.Helper()
	_, err := pool.Exec(t.Context(), `
		INSERT INTO card.card_pack_cards (pack_id, card_id, copies)
		VALUES ($1, $2, $3)`,
		packID, cardID, copies,
	)
	require.NoError(t, err)
}

func toPackCardIDs(pack *domain.CardPack) []string {
	cardIDs := make([]string, 0, len(pack.Cards))
	for _, card := range pack.Cards {
		cardIDs = append(cardIDs, card.CardID)
	}
	return cardIDs
}

func TestPostgresCardPackRepositoryGetPack(t *testing.T) {
	t.Run("[配布パック]配布パックの取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("パックTST-1001の内包カードがTST-0003、TST-0001、TST-0002の順に保存されているとき、パックIDがTST-1001で取得すると、内包カードがTST-0001、TST-0002、TST-0003の順に返る", func(t *testing.T) {
				pool := preparePool(t)
				insertCardPack(t, pool, "TST-1001", "テスト用パック", true)
				insertCardPackCard(t, pool, "TST-1001", "TST-0003", 1)
				insertCardPackCard(t, pool, "TST-1001", "TST-0001", 1)
				insertCardPackCard(t, pool, "TST-1001", "TST-0002", 1)

				got, err := repository.NewPostgresCardPackRepository(pool).GetPack(t.Context(), "TST-1001")

				require.NoError(t, err)
				assert.Equal(t, []string{"TST-0001", "TST-0002", "TST-0003"}, toPackCardIDs(got))
			})

			t.Run("パックTST-1001にカードIDがTST-0002、配布枚数が4の内包カードが保存されているとき、パックIDがTST-1001で取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got domain.CardPackCard)
				}{
					{"内包カードのカードIDがTST-0002になる", func(t *testing.T, got domain.CardPackCard) { assert.Equal(t, "TST-0002", got.CardID) }},
					{"内包カードの配布枚数が4になる", func(t *testing.T, got domain.CardPackCard) { assert.Equal(t, 4, got.Copies) }},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						pool := preparePool(t)
						insertCardPack(t, pool, "TST-1001", "テスト用パック", true)
						insertCardPackCard(t, pool, "TST-1001", "TST-0002", 4)

						got, err := repository.NewPostgresCardPackRepository(pool).GetPack(t.Context(), "TST-1001")

						require.NoError(t, err)
						require.Len(t, got.Cards, 1)
						tc.check(t, got.Cards[0])
					})
				}
			})

			t.Run("パックTST-1001がカードTST-0001を、パックTST-1002がカードTST-0002を内包しているとき、パックIDがTST-1001で取得すると、内包カードはTST-0001の1件だけが返る", func(t *testing.T) {
				pool := preparePool(t)
				insertCardPack(t, pool, "TST-1001", "テスト用パック", true)
				insertCardPack(t, pool, "TST-1002", "テスト用パック乙", true)
				insertCardPackCard(t, pool, "TST-1001", "TST-0001", 1)
				insertCardPackCard(t, pool, "TST-1002", "TST-0002", 1)

				got, err := repository.NewPostgresCardPackRepository(pool).GetPack(t.Context(), "TST-1001")

				require.NoError(t, err)
				assert.Equal(t, []string{"TST-0001"}, toPackCardIDs(got))
			})

			t.Run("パックIDがTST-1001、説明がテスト用パック、有効フラグがtrueのパックが保存されているとき、パックIDがTST-1001で取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got *domain.CardPack)
				}{
					{"パックIDがTST-1001になる", func(t *testing.T, got *domain.CardPack) { assert.Equal(t, "TST-1001", got.PackID) }},
					{"説明がテスト用パックになる", func(t *testing.T, got *domain.CardPack) { assert.Equal(t, "テスト用パック", got.Description) }},
					{"有効フラグがtrueになる", func(t *testing.T, got *domain.CardPack) { assert.True(t, got.IsActive) }},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						pool := preparePool(t)
						insertCardPack(t, pool, "TST-1001", "テスト用パック", true)

						got, err := repository.NewPostgresCardPackRepository(pool).GetPack(t.Context(), "TST-1001")

						require.NoError(t, err)
						tc.check(t, got)
					})
				}
			})

			t.Run("パックTST-1001が有効フラグfalseで保存されているとき、パックIDがTST-1001で取得すると、パックの有効フラグがfalseで返る", func(t *testing.T) {
				pool := preparePool(t)
				insertCardPack(t, pool, "TST-1001", "テスト用パック", false)

				got, err := repository.NewPostgresCardPackRepository(pool).GetPack(t.Context(), "TST-1001")

				require.NoError(t, err)
				assert.False(t, got.IsActive)
			})

			t.Run("パックTST-1001が内包カードTST-0001の1件だけで保存されているとき、パックIDがTST-1001で取得すると、内包カードはTST-0001の1件で返る", func(t *testing.T) {
				pool := preparePool(t)
				insertCardPack(t, pool, "TST-1001", "テスト用パック", true)
				insertCardPackCard(t, pool, "TST-1001", "TST-0001", 1)

				got, err := repository.NewPostgresCardPackRepository(pool).GetPack(t.Context(), "TST-1001")

				require.NoError(t, err)
				assert.Equal(t, []string{"TST-0001"}, toPackCardIDs(got))
			})

			t.Run("パックTST-1001が内包カード0件で保存されているとき、パックIDがTST-1001で取得すると、内包カードの一覧は空になる", func(t *testing.T) {
				pool := preparePool(t)
				insertCardPack(t, pool, "TST-1001", "テスト用パック", true)

				got, err := repository.NewPostgresCardPackRepository(pool).GetPack(t.Context(), "TST-1001")

				require.NoError(t, err)
				assert.Len(t, got.Cards, 0)
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("パックTST-1001だけが保存されているとき、保存されていないパックIDのTST-1009で取得すると、存在しないことを示すエラーが返る", func(t *testing.T) {
				pool := preparePool(t)
				insertCardPack(t, pool, "TST-1001", "テスト用パック", true)

				_, err := repository.NewPostgresCardPackRepository(pool).GetPack(t.Context(), "TST-1009")

				assert.ErrorIs(t, err, port.ErrNotFound)
			})
		})
	})
}
