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

// TestGetPack は指定 pack_id の card_pack 行と card_pack_cards 行を結合して
// domain.CardPack を組み立てて返すこと、card_id の昇順で内包カードが並ぶこと
// を確認する。Select 関数の主要責務はこの集約の正確性。
func TestGetPack(t *testing.T) {
	sharedPg.Truncate(t)
	seedCardPack(t, cardPackSeed{
		PackID:   "p1",
		IsActive: true,
		Cards: []domain.CardPackCard{
			{CardID: "SH-0002", Copies: 1},
			{CardID: "SH-0001", Copies: 3},
		},
	})

	repo := repository.NewPgCardPackRepository(sharedPg.Pool)
	got, err := repo.GetPack(context.Background(), "p1")

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "p1", got.PackID)
	assert.True(t, got.IsActive)
	assert.Equal(t, []domain.CardPackCard{
		{CardID: "SH-0001", Copies: 3},
		{CardID: "SH-0002", Copies: 1},
	}, got.Cards)
}

// TestGetPack_OnlyReturnsRequestedPack は指定 pack_id 以外の行が混入しない
// ことを確認する。同一テーブルに複数 pack を入れて、要求した pack の cards
// だけが返ることを assert する。
func TestGetPack_OnlyReturnsRequestedPack(t *testing.T) {
	sharedPg.Truncate(t)
	seedCardPack(t, cardPackSeed{
		PackID:   "wanted",
		IsActive: true,
		Cards:    []domain.CardPackCard{{CardID: "SH-0001", Copies: 3}},
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
	assert.Equal(t, []domain.CardPackCard{{CardID: "SH-0001", Copies: 3}}, got.Cards)
}

// TestGetPack_NotFound は対象 pack_id が DB に存在しないとき port.ErrNotFound が
// 返ることを固定する (呼び出し側の errors.Is で分岐できるようにするため)。
func TestGetPack_NotFound(t *testing.T) {
	sharedPg.Truncate(t)

	repo := repository.NewPgCardPackRepository(sharedPg.Pool)
	got, err := repo.GetPack(context.Background(), "missing")

	require.Error(t, err)
	assert.True(t, errors.Is(err, port.ErrNotFound))
	assert.Nil(t, got)
}

// TestGetPack_NoCards は card_pack 行はあるが card_pack_cards 行が 0 件の場合、
// 空 Cards を持つ CardPack が返ることを固定する (運用判定は usecase 側の責務)。
func TestGetPack_NoCards(t *testing.T) {
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
	assert.Equal(t, "empty", got.PackID)
	assert.Empty(t, got.Cards)
}
