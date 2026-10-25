package cache_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/cache/cachetest"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

func loadInitiativeCache(t *testing.T, initiativeIDs ...string) *cache.InitiativeCache {
	t.Helper()
	initiativeCache := cache.NewInitiativeCache()
	require.NoError(t, initiativeCache.Load(context.Background(), cachetest.NewFakeInitiativeRepository(initiativeIDs...)))
	return initiativeCache
}

func listInitiativeIDs(initiatives []domain.Initiative) []string {
	initiativeIDs := make([]string, 0, len(initiatives))
	for _, initiative := range initiatives {
		initiativeIDs = append(initiativeIDs, initiative.InitiativeID)
	}
	return initiativeIDs
}

func TestInitiativeCache(t *testing.T) {
	t.Run("[施策定義キャッシュ]施策定義の読み込み", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("施策定義がTST-0001の1件のとき、読み込むと", func(t *testing.T) {
				t.Run("TST-0001で参照した施策定義の施策IDがTST-0001になる", func(t *testing.T) {
					initiativeCache := loadInitiativeCache(t, "TST-0001")

					initiative := initiativeCache.FindByID("TST-0001")

					require.NotNil(t, initiative)
					assert.Equal(t, "TST-0001", initiative.InitiativeID)
				})

				t.Run("保持件数が1件になる", func(t *testing.T) {
					initiativeCache := loadInitiativeCache(t, "TST-0001")

					assert.Equal(t, 1, initiativeCache.Count())
				})
			})

			t.Run("施策定義がTST-0001からTST-0003の3件のとき、読み込むと", func(t *testing.T) {
				t.Run("TST-0002で参照した施策定義の施策IDがTST-0002になる", func(t *testing.T) {
					initiativeCache := loadInitiativeCache(t, "TST-0001", "TST-0002", "TST-0003")

					initiative := initiativeCache.FindByID("TST-0002")

					require.NotNil(t, initiative)
					assert.Equal(t, "TST-0002", initiative.InitiativeID)
				})

				t.Run("TST-0004では施策定義が見つからない", func(t *testing.T) {
					initiativeCache := loadInitiativeCache(t, "TST-0001", "TST-0002", "TST-0003")

					assert.Nil(t, initiativeCache.FindByID("TST-0004"))
				})

				t.Run("保持件数が3件になる", func(t *testing.T) {
					initiativeCache := loadInitiativeCache(t, "TST-0001", "TST-0002", "TST-0003")

					assert.Equal(t, 3, initiativeCache.Count())
				})
			})

			t.Run("施策定義がTST-0003、TST-0001、TST-0002の順で取得できるとき、読み込むと一覧の施策IDの並びがTST-0001、TST-0002、TST-0003になる", func(t *testing.T) {
				initiativeCache := loadInitiativeCache(t, "TST-0003", "TST-0001", "TST-0002")

				assert.Equal(t, []string{"TST-0001", "TST-0002", "TST-0003"}, listInitiativeIDs(initiativeCache.FindAll()))
			})

			t.Run("TST-0001とTST-0002を読み込み済みで、保管先がTST-0004の1件だけを返すとき、再度読み込むと", func(t *testing.T) {
				t.Run("一覧の施策IDがTST-0004だけになる", func(t *testing.T) {
					initiativeCache := loadInitiativeCache(t, "TST-0001", "TST-0002")
					require.NoError(t, initiativeCache.Load(context.Background(), cachetest.NewFakeInitiativeRepository("TST-0004")))

					assert.Equal(t, []string{"TST-0004"}, listInitiativeIDs(initiativeCache.FindAll()))
				})

				t.Run("TST-0001でもTST-0002でも施策定義が見つからない", func(t *testing.T) {
					initiativeCache := loadInitiativeCache(t, "TST-0001", "TST-0002")
					require.NoError(t, initiativeCache.Load(context.Background(), cachetest.NewFakeInitiativeRepository("TST-0004")))

					assert.Nil(t, initiativeCache.FindByID("TST-0001"))
					assert.Nil(t, initiativeCache.FindByID("TST-0002"))
				})
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("施策定義の保管先がstore-unreachableというエラーを返したとき、読み込みのエラーの原因にstore-unreachableが含まれる", func(t *testing.T) {
				repository := &cachetest.FakeInitiativeRepository{Err: cachetest.ErrStoreUnreachable}

				err := cache.NewInitiativeCache().Load(context.Background(), repository)

				assert.ErrorIs(t, err, cachetest.ErrStoreUnreachable)
			})

			t.Run("施策定義の保管先が0件を返したとき、読み込みは施策定義が0件であることを示すエラーを返す", func(t *testing.T) {
				err := cache.NewInitiativeCache().Load(context.Background(), cachetest.NewFakeInitiativeRepository())

				assert.ErrorIs(t, err, cache.ErrNoInitiativeDefinitions)
			})

			t.Run("TST-0001とTST-0002を読み込み済みのキャッシュに再度読み込んで失敗したとき", func(t *testing.T) {
				tests := []struct {
					name       string
					repository *cachetest.FakeInitiativeRepository
				}{
					{
						name:       "保管先が取得に失敗したとき、一覧の施策IDの並びがTST-0001、TST-0002になる",
						repository: &cachetest.FakeInitiativeRepository{Err: cachetest.ErrStoreUnreachable},
					},
					{
						name:       "保管先が0件を返したとき、一覧の施策IDの並びがTST-0001、TST-0002になる",
						repository: cachetest.NewFakeInitiativeRepository(),
					},
				}
				for _, tt := range tests {
					t.Run(tt.name, func(t *testing.T) {
						initiativeCache := loadInitiativeCache(t, "TST-0001", "TST-0002")
						require.Error(t, initiativeCache.Load(context.Background(), tt.repository))

						assert.Equal(t, []string{"TST-0001", "TST-0002"}, listInitiativeIDs(initiativeCache.FindAll()))
					})
				}
			})
		})
	})
}
