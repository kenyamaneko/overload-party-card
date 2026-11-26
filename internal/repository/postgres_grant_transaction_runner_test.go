//go:build integration

package repository_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
	"github.com/kenyamaneko/overload-party-card/internal/repository/repositorytest"
)

var errAfterGrant = errors.New("追加後の失敗")

type grantTransactionOutcome struct {
	pool       *pgxpool.Pool
	err        error
	isInserted bool
}

func runGrantTransaction(t *testing.T, pool *pgxpool.Pool, failureAfterAdd error) grantTransactionOutcome {
	t.Helper()
	outcome := grantTransactionOutcome{pool: pool}
	outcome.err = repository.NewPostgresGrantTransactionRunner(pool).RunInTransaction(
		t.Context(),
		func(repositories port.GrantRepositories) error {
			isInserted, err := repositories.ProcessedEvents.Insert(t.Context(), "11111111-1111-1111-1111-111111111111", repositorytest.EventTypeForTest)
			require.NoError(t, err)
			outcome.isInserted = isInserted
			require.NoError(t, repositories.PlayerCards.AddCards(t.Context(), []domain.PlayerCard{
				{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 0, Count: 3},
			}))
			return failureAfterAdd
		},
	)
	return outcome
}

func requireGrantedCard(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	got, err := repository.NewPostgresPlayerCardRepository(pool).GetPlayerCards(t.Context(), repositorytest.PlayerOneID)
	require.NoError(t, err)
	want := []domain.PlayerCard{{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 0, Count: 3}}
	assert.Equal(t, want, repositorytest.ToPlayerCards(got))
}

func TestPostgresGrantTransactionRunnerRunInTransaction(t *testing.T) {
	t.Run("[カード配布]処理済みイベントの記録と配布カードの追加の同時確定", func(t *testing.T) {
		t.Run("イベントID11111111-1111-1111-1111-111111111111の記録と、プレイヤー00000000-0000-0000-0000-000000000001へのTST-0001・アート番号0・配布枚数3の追加を、この順に行うとき", func(t *testing.T) {
			t.Run("正常系", func(t *testing.T) {
				t.Run("追加のあとに失敗が起きなかったとき", func(t *testing.T) {
					cases := []struct {
						name  string
						check func(t *testing.T, outcome grantTransactionOutcome)
					}{
						{"DBに保存されている処理済みイベントの件数は1件になる", func(t *testing.T, outcome grantTransactionOutcome) {
							assert.Equal(t, 1, repositorytest.CountProcessedEvents(t, outcome.pool))
						}},
						{"記録したイベントID11111111-1111-1111-1111-111111111111を、再度記録すると、重複であることを示す結果が返る", func(t *testing.T, outcome grantTransactionOutcome) {
							isInserted, err := repository.NewPostgresProcessedEventRepository(outcome.pool).Insert(t.Context(), "11111111-1111-1111-1111-111111111111", repositorytest.EventTypeForTest)
							require.NoError(t, err)
							assert.False(t, isInserted)
						}},
						{"所持カードを取得すると、TST-0001・アート番号0・枚数3の1件だけが返る", func(t *testing.T, outcome grantTransactionOutcome) {
							requireGrantedCard(t, outcome.pool)
						}},
					}
					for _, tc := range cases {
						t.Run(tc.name, func(t *testing.T) {
							outcome := runGrantTransaction(t, preparePool(t), nil)

							require.NoError(t, outcome.err)
							tc.check(t, outcome)
						})
					}
				})
			})

			t.Run("異常系", func(t *testing.T) {
				t.Run("追加のあとに「追加後の失敗」が起きたとき", func(t *testing.T) {
					cases := []struct {
						name  string
						check func(t *testing.T, outcome grantTransactionOutcome)
					}{
						{"呼び出し元に返るエラーの原因が「追加後の失敗」になる", func(t *testing.T, outcome grantTransactionOutcome) {
							assert.ErrorIs(t, outcome.err, errAfterGrant)
						}},
						{"DBに保存されている処理済みイベントの件数は0件になる", func(t *testing.T, outcome grantTransactionOutcome) {
							assert.Equal(t, 0, repositorytest.CountProcessedEvents(t, outcome.pool))
						}},
						{"所持カードを取得すると、空の一覧が返る", func(t *testing.T, outcome grantTransactionOutcome) {
							got, err := repository.NewPostgresPlayerCardRepository(outcome.pool).GetPlayerCards(t.Context(), repositorytest.PlayerOneID)
							require.NoError(t, err)
							assert.Len(t, got, 0)
						}},
					}
					for _, tc := range cases {
						t.Run(tc.name, func(t *testing.T) {
							outcome := runGrantTransaction(t, preparePool(t), errAfterGrant)

							tc.check(t, outcome)
						})
					}
				})

				t.Run("追加のあとに「追加後の失敗」が起きて記録も追加も残らなかったあと、同じイベントIDの記録と同じ追加を再度行ったとき", func(t *testing.T) {
					cases := []struct {
						name  string
						check func(t *testing.T, retry grantTransactionOutcome)
					}{
						{"再度の記録は、新規であることを示す結果が返る", func(t *testing.T, retry grantTransactionOutcome) {
							assert.True(t, retry.isInserted)
						}},
						{"所持カードを取得すると、TST-0001・アート番号0・枚数3の1件だけが返る", func(t *testing.T, retry grantTransactionOutcome) {
							requireGrantedCard(t, retry.pool)
						}},
					}
					for _, tc := range cases {
						t.Run(tc.name, func(t *testing.T) {
							pool := preparePool(t)
							failed := runGrantTransaction(t, pool, errAfterGrant)
							require.ErrorIs(t, failed.err, errAfterGrant)

							retry := runGrantTransaction(t, pool, nil)

							require.NoError(t, retry.err)
							tc.check(t, retry)
						})
					}
				})
			})
		})
	})
}
