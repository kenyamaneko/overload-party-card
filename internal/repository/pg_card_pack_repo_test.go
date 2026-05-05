//go:build integration

package repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

// cardPackSeed は card.card_pack への最小シード入力。
// テストごとに pack の selection / is_active / copies を切り替えるためのフィクスチャ。
type cardPackSeed struct {
	PackID        string
	Description   string
	SelectionJSON string // JSONB として渡す
	CopiesPerCard int
	IsActive      bool
}

func seedCardPack(t *testing.T, s cardPackSeed) {
	t.Helper()
	_, err := sharedPg.Pool.Exec(context.Background(),
		`INSERT INTO card.card_pack (pack_id, description, selection, copies_per_card, is_active)
		 VALUES ($1, $2, $3::jsonb, $4, $5)`,
		s.PackID, s.Description, s.SelectionJSON, s.CopiesPerCard, s.IsActive)
	require.NoError(t, err)
}

// TestPgCardPackRepository_GetPack は card_pack マスターから 1 行を読み出し、
// selection JSONB を domain.Selection sum type にパースする仕様を検証する。
//
// 仕様意図:
//   - 既存 pack 取得: selection / copies / is_active が domain 値に正しく投影される
//   - by_factions / by_card_ids いずれの selection もパースできる
//   - 不在 pack_id: port.ErrNotFound (selection の妥当性は問わない)
//   - 未知 selection.type: port.ErrInvalidPackSelection (握りつぶさず明示エラー)
func TestPgCardPackRepository_GetPack(t *testing.T) {
	tests := []struct {
		name              string
		seeds             []cardPackSeed
		queryPackID       string
		wantErrIs         error
		wantPackID        string
		wantCopies        int
		wantIsActive      bool
		wantSelectionType domain.Selection
	}{
		{
			name: "by_factions の active pack が取得できる",
			seeds: []cardPackSeed{{
				PackID:        "initial_she",
				Description:   "オンボーディング初期パック (SHE)",
				SelectionJSON: `{"type":"by_factions","factions":["SHE","Neutral"]}`,
				CopiesPerCard: 3,
				IsActive:      true,
			}},
			queryPackID:       "initial_she",
			wantPackID:        "initial_she",
			wantCopies:        3,
			wantIsActive:      true,
			wantSelectionType: domain.SelectionByFactions{Factions: []string{"SHE", "Neutral"}},
		},
		{
			name: "by_card_ids の active pack が取得できる",
			seeds: []cardPackSeed{{
				PackID:        "limited_xmas",
				Description:   "限定パック",
				SelectionJSON: `{"type":"by_card_ids","card_ids":["LM-0001","LM-0002"]}`,
				CopiesPerCard: 1,
				IsActive:      true,
			}},
			queryPackID:       "limited_xmas",
			wantPackID:        "limited_xmas",
			wantCopies:        1,
			wantIsActive:      true,
			wantSelectionType: domain.SelectionByCardIDs{CardIDs: []string{"LM-0001", "LM-0002"}},
		},
		{
			name: "is_active=false の pack も読み出せる (運用判定は usecase 側)",
			seeds: []cardPackSeed{{
				PackID:        "retired_pack",
				SelectionJSON: `{"type":"by_factions","factions":["SHE"]}`,
				CopiesPerCard: 3,
				IsActive:      false,
			}},
			queryPackID:       "retired_pack",
			wantPackID:        "retired_pack",
			wantCopies:        3,
			wantIsActive:      false,
			wantSelectionType: domain.SelectionByFactions{Factions: []string{"SHE"}},
		},
		{
			name:        "不在 pack_id は port.ErrNotFound を返す",
			seeds:       nil,
			queryPackID: "missing_pack",
			wantErrIs:   port.ErrNotFound,
		},
		{
			name: "未知 selection.type は port.ErrInvalidPackSelection を返す",
			seeds: []cardPackSeed{{
				PackID:        "weird_pack",
				SelectionJSON: `{"type":"by_rarity","rarities":["legendary"]}`,
				CopiesPerCard: 1,
				IsActive:      true,
			}},
			queryPackID: "weird_pack",
			wantErrIs:   port.ErrInvalidPackSelection,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sharedPg.Truncate(t)
			for _, s := range tt.seeds {
				seedCardPack(t, s)
			}

			repo := repository.NewPgCardPackRepository(sharedPg.Pool)
			got, err := repo.GetPack(context.Background(), tt.queryPackID)

			verifiers := map[bool]func(){
				true: func() {
					require.NoError(t, err)
					require.NotNil(t, got)
					assert.Equal(t, tt.wantPackID, got.PackID)
					assert.Equal(t, tt.wantCopies, got.CopiesPerCard)
					assert.Equal(t, tt.wantIsActive, got.IsActive)
					assert.Equal(t, tt.wantSelectionType, got.Selection)
				},
				false: func() {
					require.Error(t, err)
					assert.True(t, errors.Is(err, tt.wantErrIs), "err=%v, want errors.Is %v", err, tt.wantErrIs)
					assert.Nil(t, got)
				},
			}
			verifiers[tt.wantErrIs == nil]()
		})
	}
}

// TestPgCardPackRepository_GetPack_PreservesSelectionJSON は selection JSONB の
// 往復 (DB 保存値 → domain → 比較) で「factions / card_ids 配列の順序が保たれる」
// ことを保証する。配布順序が pack 定義に依存するため、順序ずれは仕様事故になる。
func TestPgCardPackRepository_GetPack_PreservesSelectionJSON(t *testing.T) {
	sharedPg.Truncate(t)

	rawJSON := `{"type":"by_factions","factions":["Tuners","Sugar","Tenki","SHE","Neutral"]}`
	seedCardPack(t, cardPackSeed{
		PackID:        "ordered_pack",
		SelectionJSON: rawJSON,
		CopiesPerCard: 1,
		IsActive:      true,
	})

	repo := repository.NewPgCardPackRepository(sharedPg.Pool)
	got, err := repo.GetPack(context.Background(), "ordered_pack")
	require.NoError(t, err)

	sel, ok := got.Selection.(domain.SelectionByFactions)
	require.True(t, ok, "selection should be SelectionByFactions, got %T", got.Selection)
	assert.Equal(t, []string{"Tuners", "Sugar", "Tenki", "SHE", "Neutral"}, sel.Factions)

	// DB の selection 列を直接 raw JSON で読み戻して、保存値が壊れていないことも確認。
	var raw json.RawMessage
	err = sharedPg.Pool.QueryRow(context.Background(),
		`SELECT selection FROM card.card_pack WHERE pack_id = $1`, "ordered_pack",
	).Scan(&raw)
	require.NoError(t, err)
	assert.JSONEq(t, rawJSON, string(raw))
}
