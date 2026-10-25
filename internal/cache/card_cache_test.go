package cache_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/cache/cachetest"
)

func loadCardCache(t *testing.T, cardIDs ...string) *cache.CardCache {
	t.Helper()
	cardCache := cache.NewCardCache()
	require.NoError(t, cardCache.Load(context.Background(), cachetest.NewFakeCardRepository(cardIDs...)))
	return cardCache
}

func TestCardCache(t *testing.T) {
	t.Run("[カード定義キャッシュ]カード定義の読み込み", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("カード定義がTST-0001の1件のとき、読み込むと", func(t *testing.T) {
				t.Run("TST-0001で参照したカード定義のカードIDがTST-0001になる", func(t *testing.T) {
					cardCache := loadCardCache(t, "TST-0001")

					card := cardCache.FindByID("TST-0001")

					require.NotNil(t, card)
					assert.Equal(t, "TST-0001", card.CardID)
				})

				t.Run("保持件数が1件になる", func(t *testing.T) {
					cardCache := loadCardCache(t, "TST-0001")

					assert.Equal(t, 1, cardCache.Count())
				})
			})

			t.Run("カード定義がTST-0001からTST-0003の3件のとき、読み込むと", func(t *testing.T) {
				t.Run("TST-0002で参照したカード定義のカードIDがTST-0002になる", func(t *testing.T) {
					cardCache := loadCardCache(t, "TST-0001", "TST-0002", "TST-0003")

					card := cardCache.FindByID("TST-0002")

					require.NotNil(t, card)
					assert.Equal(t, "TST-0002", card.CardID)
				})

				t.Run("TST-0004ではカード定義が見つからない", func(t *testing.T) {
					cardCache := loadCardCache(t, "TST-0001", "TST-0002", "TST-0003")

					assert.Nil(t, cardCache.FindByID("TST-0004"))
				})

				t.Run("保持件数が3件になる", func(t *testing.T) {
					cardCache := loadCardCache(t, "TST-0001", "TST-0002", "TST-0003")

					assert.Equal(t, 3, cardCache.Count())
				})
			})

			t.Run("TST-0001とTST-0002を読み込み済みで、保管先がTST-0004の1件だけを返すとき、再度読み込むと", func(t *testing.T) {
				t.Run("TST-0004でカード定義を参照できる", func(t *testing.T) {
					cardCache := loadCardCache(t, "TST-0001", "TST-0002")
					require.NoError(t, cardCache.Load(context.Background(), cachetest.NewFakeCardRepository("TST-0004")))

					card := cardCache.FindByID("TST-0004")

					require.NotNil(t, card)
					assert.Equal(t, "TST-0004", card.CardID)
				})

				t.Run("TST-0001ではカード定義が見つからない", func(t *testing.T) {
					cardCache := loadCardCache(t, "TST-0001", "TST-0002")
					require.NoError(t, cardCache.Load(context.Background(), cachetest.NewFakeCardRepository("TST-0004")))

					assert.Nil(t, cardCache.FindByID("TST-0001"))
				})
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("カード定義の保管先がstore-unreachableというエラーを返したとき、読み込みのエラーの原因にstore-unreachableが含まれる", func(t *testing.T) {
				repository := &cachetest.FakeCardRepository{Err: cachetest.ErrStoreUnreachable}

				err := cache.NewCardCache().Load(context.Background(), repository)

				assert.ErrorIs(t, err, cachetest.ErrStoreUnreachable)
			})

			t.Run("カード定義の保管先が0件を返したとき、読み込みはカード定義が0件であることを示すエラーを返す", func(t *testing.T) {
				err := cache.NewCardCache().Load(context.Background(), cachetest.NewFakeCardRepository())

				assert.ErrorIs(t, err, cache.ErrNoCardDefinitions)
			})

			t.Run("TST-0001だけを読み込み済みのキャッシュに再度読み込んで失敗したとき", func(t *testing.T) {
				tests := []struct {
					name       string
					repository *cachetest.FakeCardRepository
				}{
					{
						name:       "保管先が取得に失敗したとき、TST-0001でカード定義を参照できる",
						repository: &cachetest.FakeCardRepository{Err: cachetest.ErrStoreUnreachable},
					},
					{
						name:       "保管先が0件を返したとき、TST-0001でカード定義を参照できる",
						repository: cachetest.NewFakeCardRepository(),
					},
				}
				for _, tt := range tests {
					t.Run(tt.name, func(t *testing.T) {
						cardCache := loadCardCache(t, "TST-0001")
						require.Error(t, cardCache.Load(context.Background(), tt.repository))

						card := cardCache.FindByID("TST-0001")

						require.NotNil(t, card)
						assert.Equal(t, "TST-0001", card.CardID)
					})
				}
			})
		})
	})
}
