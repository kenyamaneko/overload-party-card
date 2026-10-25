//go:build integration

package repository_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
	"github.com/kenyamaneko/overload-party-card/internal/repository/repositorytest"
)

type cardRow struct {
	CardID        string
	CardName      string
	ResourceLabel string
	Faction       string
	CardType      string
	Subtype       *string
	IsResizable   bool
	IsElastic     bool
	Stats         string
	EffectText    *string
	Effects       *string
	Restriction   string
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func newCardRow(cardID string) cardRow {
	return cardRow{
		CardID:        cardID,
		CardName:      "テストカード",
		ResourceLabel: "テストラベル",
		Faction:       "Tenki",
		CardType:      "Attachment",
		Stats:         "{}",
		Restriction:   "unlimited",
		IsActive:      true,
		CreatedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func insertCardDefinition(t *testing.T, pool *pgxpool.Pool, row cardRow) {
	t.Helper()
	_, err := pool.Exec(t.Context(), `
		INSERT INTO card.card_definitions (
			card_id, card_name, resource_label, faction, card_type, subtype, resizable, elastic,
			stats, effect_text, effects, restriction, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::text::jsonb, $10, $11::text::jsonb, $12, $13, $14, $15)`,
		row.CardID, row.CardName, row.ResourceLabel, row.Faction, row.CardType, row.Subtype,
		row.IsResizable, row.IsElastic, row.Stats, row.EffectText, row.Effects, row.Restriction,
		row.IsActive, row.CreatedAt, row.UpdatedAt,
	)
	require.NoError(t, err)
}

func toCardIDs(cards []*domain.Card) []string {
	cardIDs := make([]string, 0, len(cards))
	for _, card := range cards {
		cardIDs = append(cardIDs, card.CardID)
	}
	return cardIDs
}

func TestPostgresCardRepositoryFindAll(t *testing.T) {
	t.Run("[カード定義]有効なカード定義の一覧取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("TST-0001は有効フラグがtrue、TST-0002は有効フラグがfalseのカード定義が保存されているとき、取得すると、TST-0001だけが返る", func(t *testing.T) {
				pool := preparePool(t)
				active := newCardRow("TST-0001")
				active.IsActive = true
				inactive := newCardRow("TST-0002")
				inactive.IsActive = false
				insertCardDefinition(t, pool, active)
				insertCardDefinition(t, pool, inactive)

				got, err := repository.NewPostgresCardRepository(pool).FindAll(t.Context())

				require.NoError(t, err)
				assert.Equal(t, []string{"TST-0001"}, toCardIDs(got))
			})

			t.Run("カード定義がTST-0003、TST-0001、TST-0002の順に保存されているとき、取得すると、TST-0001、TST-0002、TST-0003の順に返る", func(t *testing.T) {
				pool := preparePool(t)
				insertCardDefinition(t, pool, newCardRow("TST-0003"))
				insertCardDefinition(t, pool, newCardRow("TST-0001"))
				insertCardDefinition(t, pool, newCardRow("TST-0002"))

				got, err := repository.NewPostgresCardRepository(pool).FindAll(t.Context())

				require.NoError(t, err)
				assert.Equal(t, []string{"TST-0001", "TST-0002", "TST-0003"}, toCardIDs(got))
			})

			t.Run("カード定義が1件も保存されていないとき、取得すると、空の一覧が返る", func(t *testing.T) {
				pool := preparePool(t)

				got, err := repository.NewPostgresCardRepository(pool).FindAll(t.Context())

				require.NoError(t, err)
				assert.Len(t, got, 0)
			})

			t.Run("カードIDがTST-0001、カード名がテストカード、リソースラベルがテストラベル、陣営がTenki、カードタイプがAttachment、Resizable属性がtrue、Elastic属性がfalse、ステータスが{\"throughput\":300}、制限区分がsemi_limited、有効フラグがtrue、作成日時が2026-01-02T03:04:05Z、更新日時が2026-02-03T04:05:06Zの有効なカード定義が保存されているとき、取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got *domain.Card)
				}{
					{"カードIDがTST-0001になる", func(t *testing.T, got *domain.Card) { assert.Equal(t, "TST-0001", got.CardID) }},
					{"カード名がテストカードになる", func(t *testing.T, got *domain.Card) { assert.Equal(t, "テストカード", got.CardName) }},
					{"リソースラベルがテストラベルになる", func(t *testing.T, got *domain.Card) { assert.Equal(t, "テストラベル", got.ResourceLabel) }},
					{"陣営がTenkiになる", func(t *testing.T, got *domain.Card) { assert.Equal(t, "Tenki", got.Faction) }},
					{"カードタイプがAttachmentになる", func(t *testing.T, got *domain.Card) { assert.Equal(t, "Attachment", got.CardType) }},
					{"Resizable属性がtrueになる", func(t *testing.T, got *domain.Card) { assert.True(t, got.IsResizable) }},
					{"Elastic属性がfalseになる", func(t *testing.T, got *domain.Card) { assert.False(t, got.IsElastic) }},
					{"ステータスが{\"throughput\":300}になる", func(t *testing.T, got *domain.Card) {
						assert.Equal(t, domain.CardStats{Throughput: repositorytest.ToPointer(int64(300))}, got.Stats)
					}},
					{"制限区分がsemi_limitedになる", func(t *testing.T, got *domain.Card) { assert.Equal(t, "semi_limited", got.Restriction) }},
					{"有効フラグがtrueになる", func(t *testing.T, got *domain.Card) { assert.True(t, got.IsActive) }},
					{"作成日時が2026-01-02T03:04:05Zになる", func(t *testing.T, got *domain.Card) {
						repositorytest.RequireTimeEqual(t, repositorytest.ParseTime(t, "2026-01-02T03:04:05Z"), got.CreatedAt)
					}},
					{"更新日時が2026-02-03T04:05:06Zになる", func(t *testing.T, got *domain.Card) {
						repositorytest.RequireTimeEqual(t, repositorytest.ParseTime(t, "2026-02-03T04:05:06Z"), got.UpdatedAt)
					}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						pool := preparePool(t)
						row := newCardRow("TST-0001")
						row.CardName = "テストカード"
						row.ResourceLabel = "テストラベル"
						row.Faction = "Tenki"
						row.CardType = "Attachment"
						row.IsResizable = true
						row.IsElastic = false
						row.Stats = `{"throughput":300}`
						row.Restriction = "semi_limited"
						row.IsActive = true
						row.CreatedAt = repositorytest.ParseTime(t, "2026-01-02T03:04:05Z")
						row.UpdatedAt = repositorytest.ParseTime(t, "2026-02-03T04:05:06Z")
						insertCardDefinition(t, pool, row)

						got, err := repository.NewPostgresCardRepository(pool).FindAll(t.Context())

						require.NoError(t, err)
						require.Len(t, got, 1)
						tc.check(t, got[0])
					})
				}
			})

			t.Run("カードIDがTST-0002、Resizable属性がfalse、Elastic属性がtrue、制限区分がlimited、有効フラグがtrueの有効なカード定義が保存されているとき、取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got *domain.Card)
				}{
					{"カードIDがTST-0002になる", func(t *testing.T, got *domain.Card) { assert.Equal(t, "TST-0002", got.CardID) }},
					{"Resizable属性がfalseになる", func(t *testing.T, got *domain.Card) { assert.False(t, got.IsResizable) }},
					{"Elastic属性がtrueになる", func(t *testing.T, got *domain.Card) { assert.True(t, got.IsElastic) }},
					{"制限区分がlimitedになる", func(t *testing.T, got *domain.Card) { assert.Equal(t, "limited", got.Restriction) }},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						pool := preparePool(t)
						row := newCardRow("TST-0002")
						row.IsResizable = false
						row.IsElastic = true
						row.Restriction = "limited"
						row.IsActive = true
						insertCardDefinition(t, pool, row)

						got, err := repository.NewPostgresCardRepository(pool).FindAll(t.Context())

						require.NoError(t, err)
						require.Len(t, got, 1)
						tc.check(t, got[0])
					})
				}
			})

			t.Run("サブタイプがVM、効果テキストがテスト効果、効果定義が[{\"trigger\":\"on_deploy\"}]の有効なカード定義が保存されているとき、取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got *domain.Card)
				}{
					{"サブタイプがVMになる", func(t *testing.T, got *domain.Card) {
						assert.Equal(t, repositorytest.ToPointer("VM"), got.Subtype)
					}},
					{"効果テキストがテスト効果になる", func(t *testing.T, got *domain.Card) {
						assert.Equal(t, repositorytest.ToPointer("テスト効果"), got.EffectText)
					}},
					{"効果定義が[{\"trigger\":\"on_deploy\"}]になる", func(t *testing.T, got *domain.Card) {
						effects, err := json.Marshal(got.Effects)
						require.NoError(t, err)
						assert.JSONEq(t, `[{"trigger":"on_deploy"}]`, string(effects))
					}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						pool := preparePool(t)
						row := newCardRow("TST-0001")
						row.Subtype = repositorytest.ToPointer("VM")
						row.EffectText = repositorytest.ToPointer("テスト効果")
						row.Effects = repositorytest.ToPointer(`[{"trigger":"on_deploy"}]`)
						row.IsActive = true
						insertCardDefinition(t, pool, row)

						got, err := repository.NewPostgresCardRepository(pool).FindAll(t.Context())

						require.NoError(t, err)
						require.Len(t, got, 1)
						tc.check(t, got[0])
					})
				}
			})

			t.Run("サブタイプ・効果テキスト・効果定義がNULLの有効なカード定義が保存されているとき、取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got *domain.Card)
				}{
					{"サブタイプが未設定で返る", func(t *testing.T, got *domain.Card) { assert.Nil(t, got.Subtype) }},
					{"効果テキストが未設定で返る", func(t *testing.T, got *domain.Card) { assert.Nil(t, got.EffectText) }},
					{"効果定義が未設定で返る", func(t *testing.T, got *domain.Card) { assert.Nil(t, got.Effects) }},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						pool := preparePool(t)
						row := newCardRow("TST-0001")
						row.Subtype = nil
						row.EffectText = nil
						row.Effects = nil
						row.IsActive = true
						insertCardDefinition(t, pool, row)

						got, err := repository.NewPostgresCardRepository(pool).FindAll(t.Context())

						require.NoError(t, err)
						require.Len(t, got, 1)
						tc.check(t, got[0])
					})
				}
			})

			t.Run("有効なカード定義を取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					stats string
					want  domain.CardStats
				}{
					{"ステータスが{\"throughput_max\":500}のとき、ステータスのthroughput_maxが500になる", `{"throughput_max":500}`, domain.CardStats{ThroughputMax: repositorytest.ToPointer(int64(500))}},
					{"ステータスが{\"yield\":120}のとき、ステータスのyieldが120になる", `{"yield":120}`, domain.CardStats{Yield: repositorytest.ToPointer(int64(120))}},
					{"ステータスが{\"yield_max\":240}のとき、ステータスのyield_maxが240になる", `{"yield_max":240}`, domain.CardStats{YieldMax: repositorytest.ToPointer(int64(240))}},
					{"ステータスが{\"availability\":99}のとき、ステータスのavailabilityが99になる", `{"availability":99}`, domain.CardStats{Availability: repositorytest.ToPointer(int64(99))}},
					{"ステータスが{\"maintenance_cost\":30}のとき、ステータスのmaintenance_costが30になる", `{"maintenance_cost":30}`, domain.CardStats{MaintenanceCost: repositorytest.ToPointer(int64(30))}},
					{"ステータスが{\"sla_penalty\":15}のとき、ステータスのsla_penaltyが15になる", `{"sla_penalty":15}`, domain.CardStats{SLAPenalty: repositorytest.ToPointer(int64(15))}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						pool := preparePool(t)
						row := newCardRow("TST-0001")
						row.Stats = tc.stats
						row.IsActive = true
						insertCardDefinition(t, pool, row)

						got, err := repository.NewPostgresCardRepository(pool).FindAll(t.Context())

						require.NoError(t, err)
						require.Len(t, got, 1)
						assert.Equal(t, tc.want, got[0].Stats)
					})
				}
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("DBに接続できないとき、取得すると、DBに接続できないことを示すエラーが返る", func(t *testing.T) {
				pool := repositorytest.NewUnreachablePool(t)

				_, err := repository.NewPostgresCardRepository(pool).FindAll(t.Context())

				repositorytest.RequireConnectionError(t, err)
			})

			t.Run("カードIDがTST-0001、有効フラグがtrue、ステータスが{\"unknown_stat\":1}のカード定義が保存されているとき、取得すると、エラー内容にunknown_statを含み、ステータスに未知の項目があることを示すエラーが返る", func(t *testing.T) {
				pool := preparePool(t)
				row := newCardRow("TST-0001")
				row.IsActive = true
				row.Stats = `{"unknown_stat":1}`
				insertCardDefinition(t, pool, row)

				_, err := repository.NewPostgresCardRepository(pool).FindAll(t.Context())

				assert.ErrorContains(t, err, "unknown_stat")
			})
		})
	})
}
