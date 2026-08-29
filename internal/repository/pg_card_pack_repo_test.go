//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

func TestCardPackRepoGetPack(t *testing.T) {
	t.Run("[配布パックリポジトリ] 配布パック取得", func(t *testing.T) {
		t.Run("指定したpack_idの行が存在しないとき、port.ErrNotFoundを返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgCardPackRepository(sharedPg.Pool)

			_, err := repo.GetPack(context.Background(), "tst_pack_missing")

			require.ErrorIs(t, err, port.ErrNotFound)
		})

		t.Run("指定したpack_idの行が存在し、配布対象カードが1件以上登録されているとき、そのパック定義と配布対象カード一覧を返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCardPack(t, cardPackSeed{PackID: "tst_pack_a", IsActive: true})
			seedCardPackCard(t, "tst_pack_a", "TST-0001", 3)
			repo := repository.NewPgCardPackRepository(sharedPg.Pool)

			got, err := repo.GetPack(context.Background(), "tst_pack_a")

			require.NoError(t, err)
			require.Equal(t, "tst_pack_a", got.PackID)
			require.Len(t, got.Cards, 1)
		})

		t.Run("指定したpack_idの行が存在し、配布対象カードが0件のとき、配布対象カードが0件のパック定義を返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCardPack(t, cardPackSeed{PackID: "tst_pack_empty", IsActive: true})
			repo := repository.NewPgCardPackRepository(sharedPg.Pool)

			got, err := repo.GetPack(context.Background(), "tst_pack_empty")

			require.NoError(t, err)
			require.Empty(t, got.Cards)
		})

		t.Run("配布対象カードが複数件あるとき、card_idの昇順で返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCardPack(t, cardPackSeed{PackID: "tst_pack_a", IsActive: true})
			seedCardPackCard(t, "tst_pack_a", "TST-0003", 1)
			seedCardPackCard(t, "tst_pack_a", "TST-0001", 1)
			seedCardPackCard(t, "tst_pack_a", "TST-0002", 1)
			repo := repository.NewPgCardPackRepository(sharedPg.Pool)

			got, err := repo.GetPack(context.Background(), "tst_pack_a")

			require.NoError(t, err)
			require.Len(t, got.Cards, 3)
			require.Equal(t, []string{"TST-0001", "TST-0002", "TST-0003"},
				[]string{got.Cards[0].CardID, got.Cards[1].CardID, got.Cards[2].CardID})
		})

		t.Run("is_active=falseのパック行を指定しても、port.ErrNotFoundにならずパック定義を取得できる", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCardPack(t, cardPackSeed{PackID: "tst_pack_a", IsActive: false})
			repo := repository.NewPgCardPackRepository(sharedPg.Pool)

			got, err := repo.GetPack(context.Background(), "tst_pack_a")

			require.NoError(t, err)
			require.False(t, got.IsActive)
		})

		t.Run("保存したDescription/IsActive/CreatedAt/UpdatedAtの値が、そのまま返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			createdAt := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
			updatedAt := time.Date(2026, 2, 5, 10, 0, 0, 0, time.UTC)
			seedFullCardPack(t, fullCardPackSeed{
				PackID: "tst_pack_a", Description: "テスト説明", IsActive: true,
				CreatedAt: createdAt, UpdatedAt: updatedAt,
			})
			repo := repository.NewPgCardPackRepository(sharedPg.Pool)

			got, err := repo.GetPack(context.Background(), "tst_pack_a")

			require.NoError(t, err)
			require.Equal(t, "テスト説明", got.Description)
			require.True(t, got.IsActive)
			require.True(t, createdAt.Equal(got.CreatedAt))
			require.True(t, updatedAt.Equal(got.UpdatedAt))
		})

		t.Run("配布対象カードのCardID/Copiesが、保存した値のまま返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCardPack(t, cardPackSeed{PackID: "tst_pack_a", IsActive: true})
			seedCardPackCard(t, "tst_pack_a", "TST-0001", 7)
			repo := repository.NewPgCardPackRepository(sharedPg.Pool)

			got, err := repo.GetPack(context.Background(), "tst_pack_a")

			require.NoError(t, err)
			require.Len(t, got.Cards, 1)
			require.Equal(t, "TST-0001", got.Cards[0].CardID)
			require.Equal(t, 7, got.Cards[0].Copies)
		})
	})
}
