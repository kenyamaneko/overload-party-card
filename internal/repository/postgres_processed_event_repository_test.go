//go:build integration

package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/repository"
	"github.com/kenyamaneko/overload-party-card/internal/repository/repositorytest"
)

func TestPostgresProcessedEventRepositoryInsert(t *testing.T) {
	t.Run("[処理済みイベント]処理済みイベントの記録", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("イベントID11111111-1111-1111-1111-111111111111を記録するとき", func(t *testing.T) {
				cases := []struct {
					name             string
					recordedEventIDs []string
				}{
					{"何も記録されていなければ、新規であることを示す結果が返る", nil},
					{"イベントID22222222-2222-2222-2222-222222222222だけが記録済みなら、新規であることを示す結果が返る", []string{"22222222-2222-2222-2222-222222222222"}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresProcessedEventRepository(preparePool(t))
						for _, recordedEventID := range tc.recordedEventIDs {
							_, err := repo.Insert(t.Context(), recordedEventID, repositorytest.EventTypeForTest)
							require.NoError(t, err)
						}

						isInserted, err := repo.Insert(t.Context(), "11111111-1111-1111-1111-111111111111", repositorytest.EventTypeForTest)

						require.NoError(t, err)
						assert.True(t, isInserted)
					})
				}
			})

			t.Run("イベントID11111111-1111-1111-1111-111111111111が記録済みのとき、そのイベントIDで記録すると、重複であることを示す結果が返る", func(t *testing.T) {
				repo := repository.NewPostgresProcessedEventRepository(preparePool(t))
				_, err := repo.Insert(t.Context(), "11111111-1111-1111-1111-111111111111", repositorytest.EventTypeForTest)
				require.NoError(t, err)

				isInserted, err := repo.Insert(t.Context(), "11111111-1111-1111-1111-111111111111", repositorytest.EventTypeForTest)

				require.NoError(t, err)
				assert.False(t, isInserted)
			})

			t.Run("イベントID11111111-1111-1111-1111-111111111111で2回続けて記録したとき、DBに保存されている処理済みイベントの件数は1件になる", func(t *testing.T) {
				pool := preparePool(t)
				repo := repository.NewPostgresProcessedEventRepository(pool)

				_, firstErr := repo.Insert(t.Context(), "11111111-1111-1111-1111-111111111111", repositorytest.EventTypeForTest)
				_, secondErr := repo.Insert(t.Context(), "11111111-1111-1111-1111-111111111111", repositorytest.EventTypeForTest)

				require.NoError(t, firstErr)
				require.NoError(t, secondErr)
				assert.Equal(t, 1, repositorytest.CountProcessedEvents(t, pool))
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("イベントIDがinvalid-event-idのとき、記録すると、UUID形式の誤りを示すエラーが返る", func(t *testing.T) {
				repo := repository.NewPostgresProcessedEventRepository(preparePool(t))

				_, err := repo.Insert(t.Context(), "invalid-event-id", repositorytest.EventTypeForTest)

				repositorytest.RequireSQLState(t, err, repositorytest.SQLStateInvalidTextRepresentation)
			})
		})
	})
}
