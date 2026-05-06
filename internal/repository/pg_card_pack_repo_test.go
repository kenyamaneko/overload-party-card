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
	PackID        string
	SelectionJSON string
	CopiesPerCard int
	IsActive      bool
}

func seedCardPack(t *testing.T, s cardPackSeed) {
	t.Helper()
	_, err := sharedPg.Pool.Exec(context.Background(),
		`INSERT INTO card.card_pack (pack_id, description, selection, copies_per_card, is_active)
		 VALUES ($1, '', $2::jsonb, $3, $4)`,
		s.PackID, s.SelectionJSON, s.CopiesPerCard, s.IsActive)
	require.NoError(t, err)
}

// TestGetPack は selection JSONB が domain.Selection sum type に正しく投影される
// ことを確認する。selection 種別ごとに投影先 domain 型が変わるのが Select 関数の
// 主要な責務であり、ここではその対応関係だけをケース化する。
func TestGetPack(t *testing.T) {
	tests := []struct {
		name          string
		seedJSON      string
		wantSelection domain.Selection
	}{
		{
			name:          "by_factions は SelectionByFactions に投影される",
			seedJSON:      `{"type":"by_factions","factions":["SHE","Neutral"]}`,
			wantSelection: domain.SelectionByFactions{Factions: []string{"SHE", "Neutral"}},
		},
		{
			name:          "by_card_ids は SelectionByCardIDs に投影される",
			seedJSON:      `{"type":"by_card_ids","card_ids":["LM-0001","LM-0002"]}`,
			wantSelection: domain.SelectionByCardIDs{CardIDs: []string{"LM-0001", "LM-0002"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCardPack(t, cardPackSeed{
				PackID:        "p1",
				SelectionJSON: tt.seedJSON,
				CopiesPerCard: 1,
				IsActive:      true,
			})

			repo := repository.NewPgCardPackRepository(sharedPg.Pool)
			got, err := repo.GetPack(context.Background(), "p1")

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "p1", got.PackID)
			assert.Equal(t, tt.wantSelection, got.Selection)
		})
	}
}

// TestGetPack_OnlyReturnsRequestedPack は指定 pack_id 以外の行が混入しないことを
// 確認する。同一テーブルに複数 pack を入れて、要求した pack だけが返ることを assert する。
func TestGetPack_OnlyReturnsRequestedPack(t *testing.T) {
	sharedPg.Truncate(t)
	seedCardPack(t, cardPackSeed{
		PackID:        "wanted",
		SelectionJSON: `{"type":"by_factions","factions":["SHE"]}`,
		CopiesPerCard: 1,
		IsActive:      true,
	})
	seedCardPack(t, cardPackSeed{
		PackID:        "other",
		SelectionJSON: `{"type":"by_factions","factions":["Tenki"]}`,
		CopiesPerCard: 1,
		IsActive:      true,
	})

	repo := repository.NewPgCardPackRepository(sharedPg.Pool)
	got, err := repo.GetPack(context.Background(), "wanted")

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "wanted", got.PackID)
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

// TestGetPack_InvalidSelection は selection JSONB の type discriminator が
// 既知でない値の場合、port.ErrInvalidPackSelection を返すことを固定する。
// pack マスターの不整合を呼び出し側で検出可能にするための契約。
func TestGetPack_InvalidSelection(t *testing.T) {
	sharedPg.Truncate(t)
	seedCardPack(t, cardPackSeed{
		PackID:        "weird",
		SelectionJSON: `{"type":"by_rarity","rarities":["legendary"]}`,
		CopiesPerCard: 1,
		IsActive:      true,
	})

	repo := repository.NewPgCardPackRepository(sharedPg.Pool)
	got, err := repo.GetPack(context.Background(), "weird")

	require.Error(t, err)
	assert.True(t, errors.Is(err, port.ErrInvalidPackSelection))
	assert.Nil(t, got)
}
