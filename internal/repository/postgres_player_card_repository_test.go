//go:build integration

package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
	"github.com/kenyamaneko/overload-party-card/internal/repository/repositorytest"
)

type playerCardKey struct {
	CardID string
	ArtNo  int64
}

func toPlayerCardCounts(cards []*domain.PlayerCard) map[playerCardKey]int {
	counts := make(map[playerCardKey]int, len(cards))
	for _, card := range cards {
		counts[playerCardKey{CardID: card.CardID, ArtNo: card.ArtNo}] = card.Count
	}
	return counts
}

func addPlayerCard(t *testing.T, repo *repository.PostgresPlayerCardRepository, card domain.PlayerCard) {
	t.Helper()
	require.NoError(t, repo.AddCards(t.Context(), []domain.PlayerCard{card}))
}

func TestPostgresPlayerCardRepositoryGetPlayerCards(t *testing.T) {
	t.Run("[所持カード]所持カードの取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("プレイヤー00000000-0000-0000-0000-000000000001がTST-0001・アート番号0・枚数2、プレイヤー00000000-0000-0000-0000-000000000002がTST-0002・アート番号0・枚数3を所持しているとき、プレイヤー00000000-0000-0000-0000-000000000001で取得すると、TST-0001・アート番号0・枚数2の1件だけが返る", func(t *testing.T) {
				repo := repository.NewPostgresPlayerCardRepository(preparePool(t))
				addPlayerCard(t, repo, domain.PlayerCard{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 0, Count: 2})
				addPlayerCard(t, repo, domain.PlayerCard{PlayerID: repositorytest.PlayerTwoID, CardID: "TST-0002", ArtNo: 0, Count: 3})

				got, err := repo.GetPlayerCards(t.Context(), repositorytest.PlayerOneID)

				require.NoError(t, err)
				want := []domain.PlayerCard{{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 0, Count: 2}}
				assert.Equal(t, want, repositorytest.ToPlayerCards(got))
			})

			t.Run("所持カードが「TST-0002・アート番号0」「TST-0001・アート番号1」「TST-0001・アート番号0」の順に保存されているとき、取得すると、TST-0001・アート番号0、TST-0001・アート番号1、TST-0002・アート番号0の順に返る", func(t *testing.T) {
				repo := repository.NewPostgresPlayerCardRepository(preparePool(t))
				addPlayerCard(t, repo, domain.PlayerCard{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0002", ArtNo: 0, Count: 1})
				addPlayerCard(t, repo, domain.PlayerCard{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 1, Count: 1})
				addPlayerCard(t, repo, domain.PlayerCard{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 0, Count: 1})

				got, err := repo.GetPlayerCards(t.Context(), repositorytest.PlayerOneID)

				require.NoError(t, err)
				want := []domain.PlayerCard{
					{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 0, Count: 1},
					{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 1, Count: 1},
					{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0002", ArtNo: 0, Count: 1},
				}
				assert.Equal(t, want, repositorytest.ToPlayerCards(got))
			})

			t.Run("所持カードが無いとき、取得すると、空の一覧が返る", func(t *testing.T) {
				repo := repository.NewPostgresPlayerCardRepository(preparePool(t))

				got, err := repo.GetPlayerCards(t.Context(), repositorytest.PlayerOneID)

				require.NoError(t, err)
				assert.Len(t, got, 0)
			})

			t.Run("プレイヤーIDが00000000-0000-0000-0000-000000000001、カードIDがTST-0001、アート番号が1、枚数が4の所持カードがあるとき、取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got *domain.PlayerCard)
				}{
					{"プレイヤーIDが00000000-0000-0000-0000-000000000001になる", func(t *testing.T, got *domain.PlayerCard) {
						assert.Equal(t, "00000000-0000-0000-0000-000000000001", got.PlayerID)
					}},
					{"カードIDがTST-0001になる", func(t *testing.T, got *domain.PlayerCard) { assert.Equal(t, "TST-0001", got.CardID) }},
					{"アート番号が1になる", func(t *testing.T, got *domain.PlayerCard) { assert.Equal(t, int64(1), got.ArtNo) }},
					{"枚数が4になる", func(t *testing.T, got *domain.PlayerCard) { assert.Equal(t, 4, got.Count) }},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresPlayerCardRepository(preparePool(t))
						addPlayerCard(t, repo, domain.PlayerCard{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 1, Count: 4})

						got, err := repo.GetPlayerCards(t.Context(), repositorytest.PlayerOneID)

						require.NoError(t, err)
						require.Len(t, got, 1)
						tc.check(t, got[0])
					})
				}
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("DBに接続できないとき、取得すると、DBに接続できないことを示すエラーが返る", func(t *testing.T) {
				pool := repositorytest.NewUnreachablePool(t)

				_, err := repository.NewPostgresPlayerCardRepository(pool).GetPlayerCards(t.Context(), repositorytest.PlayerOneID)

				repositorytest.RequireConnectionError(t, err)
			})
		})
	})
}

func TestPostgresPlayerCardRepositoryAddCards(t *testing.T) {
	t.Run("[所持カード]配布カードの追加", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("プレイヤー00000000-0000-0000-0000-000000000001がTST-0001・アート番号0・枚数2を所持しているとき、配布カード0件で追加したあと、所持カードを取得すると、TST-0001・アート番号0・枚数2の1件だけが返る", func(t *testing.T) {
				repo := repository.NewPostgresPlayerCardRepository(preparePool(t))
				addPlayerCard(t, repo, domain.PlayerCard{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 0, Count: 2})

				require.NoError(t, repo.AddCards(t.Context(), []domain.PlayerCard{}))
				got, err := repo.GetPlayerCards(t.Context(), repositorytest.PlayerOneID)

				require.NoError(t, err)
				want := []domain.PlayerCard{{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 0, Count: 2}}
				assert.Equal(t, want, repositorytest.ToPlayerCards(got))
			})

			t.Run("プレイヤー00000000-0000-0000-0000-000000000001に配布カードを追加したあと、プレイヤー00000000-0000-0000-0000-000000000001の所持カードを取得すると", func(t *testing.T) {
				t.Run("TST-0001を所持していないとき、TST-0001・アート番号2・配布枚数3を追加すると", func(t *testing.T) {
					cases := []struct {
						name  string
						check func(t *testing.T, got []*domain.PlayerCard)
					}{
						{"TST-0001のアート番号が2で返る", func(t *testing.T, got []*domain.PlayerCard) {
							assert.Contains(t, toPlayerCardCounts(got), playerCardKey{CardID: "TST-0001", ArtNo: 2})
						}},
						{"TST-0001の枚数が3で返る", func(t *testing.T, got []*domain.PlayerCard) {
							assert.Equal(t, 3, toPlayerCardCounts(got)[playerCardKey{CardID: "TST-0001", ArtNo: 2}])
						}},
					}
					for _, tc := range cases {
						t.Run(tc.name, func(t *testing.T) {
							repo := repository.NewPostgresPlayerCardRepository(preparePool(t))

							addPlayerCard(t, repo, domain.PlayerCard{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 2, Count: 3})
							got, err := repo.GetPlayerCards(t.Context(), repositorytest.PlayerOneID)

							require.NoError(t, err)
							tc.check(t, got)
						})
					}
				})

				t.Run("TST-0001のアート番号0を枚数2で所持しているとき、TST-0001・アート番号0・配布枚数3を追加すると、TST-0001のアート番号0の枚数は5になる", func(t *testing.T) {
					repo := repository.NewPostgresPlayerCardRepository(preparePool(t))
					addPlayerCard(t, repo, domain.PlayerCard{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 0, Count: 2})

					addPlayerCard(t, repo, domain.PlayerCard{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 0, Count: 3})
					got, err := repo.GetPlayerCards(t.Context(), repositorytest.PlayerOneID)

					require.NoError(t, err)
					assert.Equal(t, 5, toPlayerCardCounts(got)[playerCardKey{CardID: "TST-0001", ArtNo: 0}])
				})

				t.Run("TST-0001のアート番号1を枚数4で所持しているとき、TST-0001・アート番号0・配布枚数2を追加すると", func(t *testing.T) {
					cases := []struct {
						name  string
						check func(t *testing.T, got []*domain.PlayerCard)
					}{
						{"TST-0001のアート番号1の枚数は4のまま返る", func(t *testing.T, got []*domain.PlayerCard) {
							assert.Equal(t, 4, toPlayerCardCounts(got)[playerCardKey{CardID: "TST-0001", ArtNo: 1}])
						}},
						{"TST-0001のアート番号0が枚数2で返る", func(t *testing.T, got []*domain.PlayerCard) {
							assert.Equal(t, 2, toPlayerCardCounts(got)[playerCardKey{CardID: "TST-0001", ArtNo: 0}])
						}},
					}
					for _, tc := range cases {
						t.Run(tc.name, func(t *testing.T) {
							repo := repository.NewPostgresPlayerCardRepository(preparePool(t))
							addPlayerCard(t, repo, domain.PlayerCard{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 1, Count: 4})

							addPlayerCard(t, repo, domain.PlayerCard{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 0, Count: 2})
							got, err := repo.GetPlayerCards(t.Context(), repositorytest.PlayerOneID)

							require.NoError(t, err)
							tc.check(t, got)
						})
					}
				})

				t.Run("TST-0001もTST-0002も所持していないとき、TST-0001・アート番号0・配布枚数2とTST-0002・アート番号0・配布枚数3を一度に追加すると、TST-0001の枚数が2、TST-0002の枚数が3で返る", func(t *testing.T) {
					repo := repository.NewPostgresPlayerCardRepository(preparePool(t))

					require.NoError(t, repo.AddCards(t.Context(), []domain.PlayerCard{
						{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 0, Count: 2},
						{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0002", ArtNo: 0, Count: 3},
					}))
					got, err := repo.GetPlayerCards(t.Context(), repositorytest.PlayerOneID)

					require.NoError(t, err)
					want := map[playerCardKey]int{
						{CardID: "TST-0001", ArtNo: 0}: 2,
						{CardID: "TST-0002", ArtNo: 0}: 3,
					}
					assert.Equal(t, want, toPlayerCardCounts(got))
				})
			})

			t.Run("プレイヤー00000000-0000-0000-0000-000000000002がTST-0001を枚数7で所持しているとき、プレイヤー00000000-0000-0000-0000-000000000001に、TST-0001・アート番号0・配布枚数3を追加したあと、プレイヤー00000000-0000-0000-0000-000000000002の所持カードを取得すると、TST-0001の枚数は7のまま返る", func(t *testing.T) {
				repo := repository.NewPostgresPlayerCardRepository(preparePool(t))
				addPlayerCard(t, repo, domain.PlayerCard{PlayerID: repositorytest.PlayerTwoID, CardID: "TST-0001", ArtNo: 0, Count: 7})

				addPlayerCard(t, repo, domain.PlayerCard{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 0, Count: 3})
				got, err := repo.GetPlayerCards(t.Context(), repositorytest.PlayerTwoID)

				require.NoError(t, err)
				assert.Equal(t, 7, toPlayerCardCounts(got)[playerCardKey{CardID: "TST-0001", ArtNo: 0}])
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("配布カードが1件のとき", func(t *testing.T) {
				t.Run("プレイヤーIDがinvalid-player-idのとき、追加すると、UUID形式の誤りを示すエラーが返る", func(t *testing.T) {
					repo := repository.NewPostgresPlayerCardRepository(preparePool(t))

					err := repo.AddCards(t.Context(), []domain.PlayerCard{{PlayerID: "invalid-player-id", CardID: "TST-0001", ArtNo: 0, Count: 1}})

					repositorytest.RequireSQLState(t, err, repositorytest.SQLStateInvalidTextRepresentation)
				})

				t.Run("DBに接続できないとき、追加すると、DBに接続できないことを示すエラーが返る", func(t *testing.T) {
					repo := repository.NewPostgresPlayerCardRepository(repositorytest.NewUnreachablePool(t))

					err := repo.AddCards(t.Context(), []domain.PlayerCard{{PlayerID: repositorytest.PlayerOneID, CardID: "TST-0001", ArtNo: 0, Count: 1}})

					repositorytest.RequireConnectionError(t, err)
				})
			})
		})
	})
}
