//go:build integration

package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

type cardPackSeed struct {
	PackID   string
	IsActive bool
	Cards    []domain.CardPackCard
}

func seedCardPack(t *testing.T, s cardPackSeed) {
	t.Helper()
	_, err := sharedPg.Pool.Exec(context.Background(),
		`INSERT INTO card.card_pack (pack_id, description, is_active)
		 VALUES ($1, '', $2)`,
		s.PackID, s.IsActive)
	require.NoError(t, err)
	for _, c := range s.Cards {
		_, err := sharedPg.Pool.Exec(context.Background(),
			`INSERT INTO card.card_pack_cards (pack_id, card_id, copies)
			 VALUES ($1, $2, $3)`,
			s.PackID, c.CardID, c.Copies)
		require.NoError(t, err)
	}
}

func TestGetPack(t *testing.T) {
	t.Run("card pack の取得", func(t *testing.T) {
		t.Run("該当 pack があるとき、card_id 昇順の Cards 付きで返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCardPack(t, cardPackSeed{
				PackID:   "wanted",
				IsActive: true,
				Cards: []domain.CardPackCard{
					{CardID: "SH-0002", Copies: 1},
					{CardID: "SH-0001", Copies: 3},
				},
			})
			seedCardPack(t, cardPackSeed{
				PackID:   "other",
				IsActive: true,
				Cards:    []domain.CardPackCard{{CardID: "TK-0001", Copies: 3}},
			})

			repo := repository.NewPgCardPackRepository(sharedPg.Pool)
			got, err := repo.GetPack(context.Background(), "wanted")

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "wanted", got.PackID)
			assert.True(t, got.IsActive)
			assert.Equal(t, []domain.CardPackCard{
				{CardID: "SH-0001", Copies: 3},
				{CardID: "SH-0002", Copies: 1},
			}, got.Cards)
		})

		t.Run("存在しない pack_id のとき、ErrNotFound になり nil を返す", func(t *testing.T) {
			sharedPg.Truncate(t)

			repo := repository.NewPgCardPackRepository(sharedPg.Pool)
			got, err := repo.GetPack(context.Background(), "missing")

			require.Error(t, err)
			assert.True(t, errors.Is(err, port.ErrNotFound))
			assert.Nil(t, got)
		})

		t.Run("card_pack_cards が 0 件のとき、空 Cards の CardPack を返す", func(t *testing.T) {
			// 空 pack を弾くかどうかの運用判定は usecase 側の責務。
			sharedPg.Truncate(t)
			seedCardPack(t, cardPackSeed{
				PackID:   "empty",
				IsActive: true,
				Cards:    nil,
			})

			repo := repository.NewPgCardPackRepository(sharedPg.Pool)
			got, err := repo.GetPack(context.Background(), "empty")

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Empty(t, got.Cards)
		})

		t.Run("運用停止中の pack を取得すると、停止状態のまま内包カード付きで返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCardPack(t, cardPackSeed{
				PackID:   "inactive-pack",
				IsActive: false,
				Cards:    []domain.CardPackCard{{CardID: "TST-0001", Copies: 2}},
			})

			repo := repository.NewPgCardPackRepository(sharedPg.Pool)
			got, err := repo.GetPack(context.Background(), "inactive-pack")

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.False(t, got.IsActive)
			assert.Equal(t, []domain.CardPackCard{{CardID: "TST-0001", Copies: 2}}, got.Cards)
		})
	})
}
