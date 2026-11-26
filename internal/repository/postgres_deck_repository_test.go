//go:build integration

package repository_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
	"github.com/kenyamaneko/overload-party-card/internal/repository/repositorytest"
)

var noDeckCards = []domain.DeckCardEntry{}

func newDeck(playerID, deckName string) domain.Deck {
	defaultTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return domain.Deck{
		PlayerID:  playerID,
		DeckName:  deckName,
		Faction:   "Tenki",
		ProductID: "TST-2001",
		RoutineID: "TST-1002",
		SpecialID: "TST-1003",
		CreatedAt: defaultTime,
		UpdatedAt: defaultTime,
	}
}

func newDeckCardEntry(cardID string, artNo int64, count int) domain.DeckCardEntry {
	return domain.DeckCardEntry{CardID: cardID, ArtNo: artNo, Count: count}
}

func createDeck(t *testing.T, repo *repository.PostgresDeckRepository, deck domain.Deck, entries []domain.DeckCardEntry) int64 {
	t.Helper()
	deckID, err := repo.Create(t.Context(), deck, entries)
	require.NoError(t, err)
	return deckID
}

func toDeckNames(decks []*domain.Deck) []string {
	deckNames := make([]string, 0, len(decks))
	for _, deck := range decks {
		deckNames = append(deckNames, deck.DeckName)
	}
	return deckNames
}

func toDeckCardEntries(cards []domain.DeckCard) []domain.DeckCardEntry {
	entries := make([]domain.DeckCardEntry, 0, len(cards))
	for _, card := range cards {
		entries = append(entries, domain.DeckCardEntry{CardID: card.CardID, ArtNo: card.ArtNo, Count: card.Count})
	}
	return entries
}

func requireDeckCards(t *testing.T, repo *repository.PostgresDeckRepository, playerID string, deckID int64, want []domain.DeckCardEntry) {
	t.Helper()
	got, err := repo.GetDeckCards(t.Context(), playerID, deckID)
	require.NoError(t, err)
	assert.ElementsMatch(t, want, toDeckCardEntries(got))
}

func requireDeckName(t *testing.T, repo *repository.PostgresDeckRepository, playerID string, deckID int64, want string) {
	t.Helper()
	got, err := repo.FindByID(t.Context(), playerID, deckID)
	require.NoError(t, err)
	assert.Equal(t, want, got.DeckName)
}

func createDeckA(t *testing.T, repo *repository.PostgresDeckRepository) int64 {
	t.Helper()
	deckA := newDeck(repositorytest.PlayerOneID, "テストデッキA")
	deckA.Faction = "Tenki"
	deckA.ProductID = "TST-2001"
	deckA.RoutineID = "TST-1002"
	deckA.SpecialID = "TST-1003"
	deckA.PlaymatNo = repositorytest.ToPointer(int64(5))
	deckA.SleeveNo = repositorytest.ToPointer(int64(7))
	deckA.CreatedAt = repositorytest.ParseTime(t, "2026-01-02T03:04:05Z")
	deckA.UpdatedAt = repositorytest.ParseTime(t, "2026-02-03T04:05:06Z")
	return createDeck(t, repo, deckA, noDeckCards)
}

func createDeckB(t *testing.T, repo *repository.PostgresDeckRepository) int64 {
	t.Helper()
	deckB := newDeck(repositorytest.PlayerOneID, "テストデッキB")
	deckB.Faction = "SHE"
	deckB.ProductID = "TST-2004"
	deckB.RoutineID = "TST-1005"
	deckB.SpecialID = "TST-1006"
	deckB.PlaymatNo = repositorytest.ToPointer(int64(6))
	deckB.SleeveNo = repositorytest.ToPointer(int64(8))
	deckB.CreatedAt = repositorytest.ParseTime(t, "2026-03-04T05:06:07Z")
	deckB.UpdatedAt = repositorytest.ParseTime(t, "2026-04-05T06:07:08Z")
	return createDeck(t, repo, deckB, noDeckCards)
}

type deckAFieldCase struct {
	name  string
	check func(t *testing.T, deckA *domain.Deck, deckAID int64)
}

var deckAFieldCases = []deckAFieldCase{
	{"プレイヤーIDが00000000-0000-0000-0000-000000000001になる", func(t *testing.T, deckA *domain.Deck, _ int64) {
		assert.Equal(t, "00000000-0000-0000-0000-000000000001", deckA.PlayerID)
	}},
	{"デッキIDがデッキAの作成時に返されたデッキIDになる", func(t *testing.T, deckA *domain.Deck, deckAID int64) {
		assert.Equal(t, deckAID, deckA.DeckID)
	}},
	{"デッキ名がテストデッキAになる", func(t *testing.T, deckA *domain.Deck, _ int64) {
		assert.Equal(t, "テストデッキA", deckA.DeckName)
	}},
	{"陣営がTenkiになる", func(t *testing.T, deckA *domain.Deck, _ int64) { assert.Equal(t, "Tenki", deckA.Faction) }},
	{"プロダクトIDがTST-2001になる", func(t *testing.T, deckA *domain.Deck, _ int64) { assert.Equal(t, "TST-2001", deckA.ProductID) }},
	{"ルーチン施策IDがTST-1002になる", func(t *testing.T, deckA *domain.Deck, _ int64) { assert.Equal(t, "TST-1002", deckA.RoutineID) }},
	{"スペシャル施策IDがTST-1003になる", func(t *testing.T, deckA *domain.Deck, _ int64) { assert.Equal(t, "TST-1003", deckA.SpecialID) }},
	{"プレイマット番号が5になる", func(t *testing.T, deckA *domain.Deck, _ int64) {
		assert.Equal(t, repositorytest.ToPointer(int64(5)), deckA.PlaymatNo)
	}},
	{"スリーブ番号が7になる", func(t *testing.T, deckA *domain.Deck, _ int64) {
		assert.Equal(t, repositorytest.ToPointer(int64(7)), deckA.SleeveNo)
	}},
	{"作成日時が2026-01-02T03:04:05Zになる", func(t *testing.T, deckA *domain.Deck, _ int64) {
		repositorytest.RequireTimeEqual(t, repositorytest.ParseTime(t, "2026-01-02T03:04:05Z"), deckA.CreatedAt)
	}},
	{"更新日時が2026-02-03T04:05:06Zになる", func(t *testing.T, deckA *domain.Deck, _ int64) {
		repositorytest.RequireTimeEqual(t, repositorytest.ParseTime(t, "2026-02-03T04:05:06Z"), deckA.UpdatedAt)
	}},
}

type deckPreparation func(t *testing.T, repo *repository.PostgresDeckRepository) (playerID string, deckID int64)

func prepareAbsentDeck(t *testing.T, _ *repository.PostgresDeckRepository) (string, int64) {
	t.Helper()
	return repositorytest.PlayerOneID, repositorytest.AbsentDeckID
}

func prepareDeckOfOtherPlayer(t *testing.T, repo *repository.PostgresDeckRepository) (string, int64) {
	t.Helper()
	deckID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキ"), noDeckCards)
	return repositorytest.PlayerTwoID, deckID
}

func TestPostgresDeckRepositoryCreate(t *testing.T) {
	t.Run("[デッキ]デッキの作成", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("デッキ名がテストデッキ、陣営がTenki、プロダクトIDがTST-2001、ルーチン施策IDがTST-1002、スペシャル施策IDがTST-1003を指定して作成したとき、返されたデッキIDで取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got *domain.Deck)
				}{
					{"デッキ名がテストデッキになる", func(t *testing.T, got *domain.Deck) { assert.Equal(t, "テストデッキ", got.DeckName) }},
					{"陣営がTenkiになる", func(t *testing.T, got *domain.Deck) { assert.Equal(t, "Tenki", got.Faction) }},
					{"プロダクトIDがTST-2001になる", func(t *testing.T, got *domain.Deck) { assert.Equal(t, "TST-2001", got.ProductID) }},
					{"ルーチン施策IDがTST-1002になる", func(t *testing.T, got *domain.Deck) { assert.Equal(t, "TST-1002", got.RoutineID) }},
					{"スペシャル施策IDがTST-1003になる", func(t *testing.T, got *domain.Deck) { assert.Equal(t, "TST-1003", got.SpecialID) }},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						deck := newDeck(repositorytest.PlayerOneID, "テストデッキ")
						deck.Faction = "Tenki"
						deck.ProductID = "TST-2001"
						deck.RoutineID = "TST-1002"
						deck.SpecialID = "TST-1003"

						deckID, err := repo.Create(t.Context(), deck, noDeckCards)
						require.NoError(t, err)
						got, err := repo.FindByID(t.Context(), repositorytest.PlayerOneID, deckID)

						require.NoError(t, err)
						tc.check(t, got)
					})
				}
			})

			t.Run("プレイマット番号が5、スリーブ番号が7を指定して作成したとき、返されたデッキIDで取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got *domain.Deck)
				}{
					{"プレイマット番号が5になる", func(t *testing.T, got *domain.Deck) {
						assert.Equal(t, repositorytest.ToPointer(int64(5)), got.PlaymatNo)
					}},
					{"スリーブ番号が7になる", func(t *testing.T, got *domain.Deck) {
						assert.Equal(t, repositorytest.ToPointer(int64(7)), got.SleeveNo)
					}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						deck := newDeck(repositorytest.PlayerOneID, "テストデッキ")
						deck.PlaymatNo = repositorytest.ToPointer(int64(5))
						deck.SleeveNo = repositorytest.ToPointer(int64(7))

						deckID, err := repo.Create(t.Context(), deck, noDeckCards)
						require.NoError(t, err)
						got, err := repo.FindByID(t.Context(), repositorytest.PlayerOneID, deckID)

						require.NoError(t, err)
						tc.check(t, got)
					})
				}
			})

			t.Run("プレイマット番号とスリーブ番号を未設定で作成したとき、返されたデッキIDで取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got *domain.Deck)
				}{
					{"プレイマット番号が未設定で返る", func(t *testing.T, got *domain.Deck) { assert.Nil(t, got.PlaymatNo) }},
					{"スリーブ番号が未設定で返る", func(t *testing.T, got *domain.Deck) { assert.Nil(t, got.SleeveNo) }},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						deck := newDeck(repositorytest.PlayerOneID, "テストデッキ")
						deck.PlaymatNo = nil
						deck.SleeveNo = nil

						deckID, err := repo.Create(t.Context(), deck, noDeckCards)
						require.NoError(t, err)
						got, err := repo.FindByID(t.Context(), repositorytest.PlayerOneID, deckID)

						require.NoError(t, err)
						tc.check(t, got)
					})
				}
			})

			t.Run("作成日時が2026-01-02T03:04:05Z、更新日時が2026-02-03T04:05:06Zを指定して作成したとき、返されたデッキIDで取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got *domain.Deck)
				}{
					{"作成日時が2026-01-02T03:04:05Zになる", func(t *testing.T, got *domain.Deck) {
						repositorytest.RequireTimeEqual(t, repositorytest.ParseTime(t, "2026-01-02T03:04:05Z"), got.CreatedAt)
					}},
					{"更新日時が2026-02-03T04:05:06Zになる", func(t *testing.T, got *domain.Deck) {
						repositorytest.RequireTimeEqual(t, repositorytest.ParseTime(t, "2026-02-03T04:05:06Z"), got.UpdatedAt)
					}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						deck := newDeck(repositorytest.PlayerOneID, "テストデッキ")
						deck.CreatedAt = repositorytest.ParseTime(t, "2026-01-02T03:04:05Z")
						deck.UpdatedAt = repositorytest.ParseTime(t, "2026-02-03T04:05:06Z")

						deckID, err := repo.Create(t.Context(), deck, noDeckCards)
						require.NoError(t, err)
						got, err := repo.FindByID(t.Context(), repositorytest.PlayerOneID, deckID)

						require.NoError(t, err)
						tc.check(t, got)
					})
				}
			})

			t.Run("デッキを作成し、返されたデッキIDでカード構成を取得するとき", func(t *testing.T) {
				cases := []struct {
					name    string
					entries []domain.DeckCardEntry
				}{
					{
						"カード構成にTST-0001・アート番号0・枚数2の1件を指定すると、TST-0001・アート番号0・枚数2の1件が返る",
						[]domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)},
					},
					{
						"カード構成にTST-0001・アート番号0・枚数2、TST-0001・アート番号1・枚数3、TST-0002・アート番号0・枚数1の3件を指定すると、TST-0001・アート番号0・枚数2、TST-0001・アート番号1・枚数3、TST-0002・アート番号0・枚数1の3件が返る",
						[]domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2), newDeckCardEntry("TST-0001", 1, 3), newDeckCardEntry("TST-0002", 0, 1)},
					},
					{
						"カード構成にTST-0001・アート番号0・枚数2、TST-0001・アート番号1・枚数3の2件を指定すると、TST-0001・アート番号0・枚数2、TST-0001・アート番号1・枚数3の2件が返る",
						[]domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2), newDeckCardEntry("TST-0001", 1, 3)},
					},
					{
						"カード構成にTST-0001・アート番号0・枚数2、TST-0002・アート番号0・枚数3の2件を指定すると、TST-0001・アート番号0・枚数2、TST-0002・アート番号0・枚数3の2件が返る",
						[]domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2), newDeckCardEntry("TST-0002", 0, 3)},
					},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))

						deckID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキ"), tc.entries)

						requireDeckCards(t, repo, repositorytest.PlayerOneID, deckID, tc.entries)
					})
				}
			})

			t.Run("デッキ名がテストデッキ、カード構成が0件で作成したとき", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64)
				}{
					{"返されたデッキIDで取得すると、デッキ名がテストデッキになる", func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64) {
						requireDeckName(t, repo, repositorytest.PlayerOneID, deckID, "テストデッキ")
					}},
					{"返されたデッキIDでカード構成を取得すると、空の一覧が返る", func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64) {
						requireDeckCards(t, repo, repositorytest.PlayerOneID, deckID, []domain.DeckCardEntry{})
					}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))

						deckID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキ"), noDeckCards)

						tc.check(t, repo, deckID)
					})
				}
			})

			t.Run("プレイヤー00000000-0000-0000-0000-000000000001が、デッキ名がテストデッキ、カード構成がTST-0001・アート番号0・枚数2のデッキを続けて2回作成したとき、返される2つのデッキIDは互いに異なる", func(t *testing.T) {
				repo := repository.NewPostgresDeckRepository(preparePool(t))
				entries := []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)}

				firstDeckID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキ"), entries)
				secondDeckID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキ"), entries)

				assert.NotEqual(t, firstDeckID, secondDeckID)
			})
		})

		t.Run("異常系", func(t *testing.T) {
			duplicatedEntries := []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 1), newDeckCardEntry("TST-0001", 0, 2)}

			t.Run("TST-0001・アート番号0・枚数1とTST-0001・アート番号0・枚数2を含むカード構成で、デッキ名がテストデッキのデッキを作成するとき", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, repo *repository.PostgresDeckRepository, createErr error)
				}{
					{"作成すると、同じカードIDとアート番号の組が重複していることを示すエラーが返る", func(t *testing.T, _ *repository.PostgresDeckRepository, createErr error) {
						repositorytest.RequireSQLState(t, createErr, repositorytest.SQLStateUniqueViolation)
					}},
					{"作成したあと、プレイヤーのデッキ一覧を取得すると、デッキ名がテストデッキのデッキは含まれない", func(t *testing.T, repo *repository.PostgresDeckRepository, _ error) {
						got, err := repo.FindByPlayerID(t.Context(), repositorytest.PlayerOneID)
						require.NoError(t, err)
						assert.NotContains(t, toDeckNames(got), "テストデッキ")
					}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))

						_, createErr := repo.Create(t.Context(), newDeck(repositorytest.PlayerOneID, "テストデッキ"), duplicatedEntries)

						tc.check(t, repo, createErr)
					})
				}
			})

			t.Run("TST-0001・アート番号0・枚数1とTST-0001・アート番号0・枚数2を含むカード構成での作成に失敗したあと、TST-0001・アート番号0・枚数1だけのカード構成でデッキ名がテストデッキのデッキを作成し、プレイヤーのデッキ一覧を取得すると、デッキ名がテストデッキのデッキが1件だけ含まれる", func(t *testing.T) {
				repo := repository.NewPostgresDeckRepository(preparePool(t))
				_, createErr := repo.Create(t.Context(), newDeck(repositorytest.PlayerOneID, "テストデッキ"), duplicatedEntries)
				repositorytest.RequireSQLState(t, createErr, repositorytest.SQLStateUniqueViolation)

				createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキ"), []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 1)})
				got, err := repo.FindByPlayerID(t.Context(), repositorytest.PlayerOneID)

				require.NoError(t, err)
				assert.Equal(t, []string{"テストデッキ"}, toDeckNames(got))
			})
		})
	})
}

func TestPostgresDeckRepositoryFindByPlayerID(t *testing.T) {
	t.Run("[デッキ]プレイヤーのデッキ一覧の取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("プレイヤー00000000-0000-0000-0000-000000000001がデッキ名テストデッキA、プレイヤー00000000-0000-0000-0000-000000000002がデッキ名テストデッキBのデッキを持つとき、プレイヤー00000000-0000-0000-0000-000000000001で取得すると、デッキ名がテストデッキAの1件だけが返る", func(t *testing.T) {
				repo := repository.NewPostgresDeckRepository(preparePool(t))
				createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキA"), noDeckCards)
				createDeck(t, repo, newDeck(repositorytest.PlayerTwoID, "テストデッキB"), noDeckCards)

				got, err := repo.FindByPlayerID(t.Context(), repositorytest.PlayerOneID)

				require.NoError(t, err)
				assert.Equal(t, []string{"テストデッキA"}, toDeckNames(got))
			})

			t.Run("更新日時が2026-02-01T00:00:00Zのテストデッキ中、2026-03-01T00:00:00Zのテストデッキ新、2026-01-01T00:00:00Zのテストデッキ古の順に作成したとき、取得すると、テストデッキ新、テストデッキ中、テストデッキ古の順に返る", func(t *testing.T) {
				repo := repository.NewPostgresDeckRepository(preparePool(t))
				middle := newDeck(repositorytest.PlayerOneID, "テストデッキ中")
				middle.UpdatedAt = repositorytest.ParseTime(t, "2026-02-01T00:00:00Z")
				newest := newDeck(repositorytest.PlayerOneID, "テストデッキ新")
				newest.UpdatedAt = repositorytest.ParseTime(t, "2026-03-01T00:00:00Z")
				oldest := newDeck(repositorytest.PlayerOneID, "テストデッキ古")
				oldest.UpdatedAt = repositorytest.ParseTime(t, "2026-01-01T00:00:00Z")
				createDeck(t, repo, middle, noDeckCards)
				createDeck(t, repo, newest, noDeckCards)
				createDeck(t, repo, oldest, noDeckCards)

				got, err := repo.FindByPlayerID(t.Context(), repositorytest.PlayerOneID)

				require.NoError(t, err)
				assert.Equal(t, []string{"テストデッキ新", "テストデッキ中", "テストデッキ古"}, toDeckNames(got))
			})

			t.Run("デッキが無いとき、取得すると、空の一覧が返る", func(t *testing.T) {
				repo := repository.NewPostgresDeckRepository(preparePool(t))

				got, err := repo.FindByPlayerID(t.Context(), repositorytest.PlayerOneID)

				require.NoError(t, err)
				assert.Len(t, got, 0)
			})

			t.Run("プレイヤーIDが00000000-0000-0000-0000-000000000001、デッキ名がテストデッキA、陣営がTenki、プロダクトIDがTST-2001、ルーチン施策IDがTST-1002、スペシャル施策IDがTST-1003、プレイマット番号が5、スリーブ番号が7、作成日時が2026-01-02T03:04:05Z、更新日時が2026-02-03T04:05:06ZのデッキAと、同じプレイヤーIDで、デッキ名がテストデッキB、陣営がSHE、プロダクトIDがTST-2004、ルーチン施策IDがTST-1005、スペシャル施策IDがTST-1006、プレイマット番号が6、スリーブ番号が8、作成日時が2026-03-04T05:06:07Z、更新日時が2026-04-05T06:07:08ZのデッキBを、この順に作成して保存してあるとき、取得すると、返った2件のうちデッキAは", func(t *testing.T) {
				for _, tc := range deckAFieldCases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						deckAID := createDeckA(t, repo)
						createDeckB(t, repo)

						got, err := repo.FindByPlayerID(t.Context(), repositorytest.PlayerOneID)

						require.NoError(t, err)
						require.Len(t, got, 2)
						decksByName := map[string]*domain.Deck{got[0].DeckName: got[0], got[1].DeckName: got[1]}
						require.Contains(t, decksByName, "テストデッキA")
						tc.check(t, decksByName["テストデッキA"], deckAID)
					})
				}
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("DBに接続できないとき、取得すると、DBに接続できないことを示すエラーが返る", func(t *testing.T) {
				repo := repository.NewPostgresDeckRepository(repositorytest.NewUnreachablePool(t))

				_, err := repo.FindByPlayerID(t.Context(), repositorytest.PlayerOneID)

				repositorytest.RequireConnectionError(t, err)
			})
		})
	})
}

func TestPostgresDeckRepositoryFindByID(t *testing.T) {
	t.Run("[デッキ]デッキの取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("プレイヤーIDが00000000-0000-0000-0000-000000000001、デッキ名がテストデッキA、陣営がTenki、プロダクトIDがTST-2001、ルーチン施策IDがTST-1002、スペシャル施策IDがTST-1003、プレイマット番号が5、スリーブ番号が7、作成日時が2026-01-02T03:04:05Z、更新日時が2026-02-03T04:05:06ZのデッキAと、同じプレイヤーIDで、デッキ名がテストデッキB、陣営がSHE、プロダクトIDがTST-2004、ルーチン施策IDがTST-1005、スペシャル施策IDがTST-1006、プレイマット番号が6、スリーブ番号が8、作成日時が2026-03-04T05:06:07Z、更新日時が2026-04-05T06:07:08ZのデッキBを、この順に作成して保存してあるとき、プレイヤーID00000000-0000-0000-0000-000000000001とデッキAのデッキIDで取得すると", func(t *testing.T) {
				for _, tc := range deckAFieldCases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						deckAID := createDeckA(t, repo)
						createDeckB(t, repo)

						got, err := repo.FindByID(t.Context(), repositorytest.PlayerOneID, deckAID)

						require.NoError(t, err)
						tc.check(t, got, deckAID)
					})
				}
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("デッキを取得するとき", func(t *testing.T) {
				cases := []struct {
					name    string
					prepare deckPreparation
				}{
					{"保存されていないデッキID999999を指定すると、存在しないことを示すエラーが返る", prepareAbsentDeck},
					{"プレイヤー00000000-0000-0000-0000-000000000001が持つデッキのデッキIDを、持ち主ではないプレイヤー00000000-0000-0000-0000-000000000002のプレイヤーIDで指定すると、存在しないことを示すエラーが返る", prepareDeckOfOtherPlayer},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						playerID, deckID := tc.prepare(t, repo)

						_, err := repo.FindByID(t.Context(), playerID, deckID)

						assert.ErrorIs(t, err, port.ErrNotFound)
					})
				}
			})
		})
	})
}

func TestPostgresDeckRepositoryGetDeckCards(t *testing.T) {
	t.Run("[デッキ]デッキのカード構成の取得", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("プレイヤー00000000-0000-0000-0000-000000000001のデッキAにカードIDがTST-0001、アート番号が1、枚数が4のカード構成があり、同じプレイヤーのデッキBにカードIDがTST-0002、アート番号が0、枚数が3のカード構成があるとき、デッキAのカード構成を取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got domain.DeckCard, deckAID int64)
				}{
					{"カード構成のプレイヤーIDが00000000-0000-0000-0000-000000000001になる", func(t *testing.T, got domain.DeckCard, _ int64) {
						assert.Equal(t, "00000000-0000-0000-0000-000000000001", got.PlayerID)
					}},
					{"カード構成のデッキIDがデッキAの作成時に返されたデッキIDになる", func(t *testing.T, got domain.DeckCard, deckAID int64) {
						assert.Equal(t, deckAID, got.DeckID)
					}},
					{"カード構成のカードIDがTST-0001になる", func(t *testing.T, got domain.DeckCard, _ int64) { assert.Equal(t, "TST-0001", got.CardID) }},
					{"カード構成のアート番号が1になる", func(t *testing.T, got domain.DeckCard, _ int64) { assert.Equal(t, int64(1), got.ArtNo) }},
					{"カード構成の枚数が4になる", func(t *testing.T, got domain.DeckCard, _ int64) { assert.Equal(t, 4, got.Count) }},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						deckAID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキA"), []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 1, 4)})
						createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキB"), []domain.DeckCardEntry{newDeckCardEntry("TST-0002", 0, 3)})

						got, err := repo.GetDeckCards(t.Context(), repositorytest.PlayerOneID, deckAID)

						require.NoError(t, err)
						require.Len(t, got, 1)
						tc.check(t, got[0], deckAID)
					})
				}
			})

			t.Run("デッキ名がテストデッキAでカード構成がTST-0001・アート番号0・枚数2のデッキと、デッキ名がテストデッキBでカード構成がTST-0002・アート番号0・枚数3のデッキが同じプレイヤーにあるとき、テストデッキAのデッキIDで取得すると、TST-0001・アート番号0・枚数2の1件だけが返る", func(t *testing.T) {
				repo := repository.NewPostgresDeckRepository(preparePool(t))
				deckAID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキA"), []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)})
				createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキB"), []domain.DeckCardEntry{newDeckCardEntry("TST-0002", 0, 3)})

				requireDeckCards(t, repo, repositorytest.PlayerOneID, deckAID, []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)})
			})

			t.Run("カード構成が0件のデッキだけがあるとき、そのデッキのカード構成を取得すると、空の一覧が返る", func(t *testing.T) {
				repo := repository.NewPostgresDeckRepository(preparePool(t))
				deckID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキ"), noDeckCards)

				requireDeckCards(t, repo, repositorytest.PlayerOneID, deckID, []domain.DeckCardEntry{})
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("プレイヤー00000000-0000-0000-0000-000000000001のデッキにTST-0001・アート番号0・枚数2のカード構成があるとき、カード構成を取得すると", func(t *testing.T) {
				cases := []struct {
					name   string
					target func(ownedDeckID int64) (playerID string, deckID int64)
				}{
					{"保存されていないデッキID999999を指定すると、空の一覧が返る", func(int64) (string, int64) {
						return repositorytest.PlayerOneID, repositorytest.AbsentDeckID
					}},
					{"プレイヤー00000000-0000-0000-0000-000000000001が持つデッキのデッキIDを、持ち主ではないプレイヤー00000000-0000-0000-0000-000000000002のプレイヤーIDで指定すると、空の一覧が返る", func(ownedDeckID int64) (string, int64) {
						return repositorytest.PlayerTwoID, ownedDeckID
					}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						ownedDeckID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキ"), []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)})
						playerID, deckID := tc.target(ownedDeckID)

						requireDeckCards(t, repo, playerID, deckID, []domain.DeckCardEntry{})
					})
				}
			})
		})
	})
}

func TestPostgresDeckRepositoryUpdate(t *testing.T) {
	t.Run("[デッキ]デッキの更新", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("デッキ名がテストデッキ、陣営がTenki、プロダクトIDがTST-2001、ルーチン施策IDがTST-1002、スペシャル施策IDがTST-1003のデッキを、デッキ名がテストデッキ改、陣営がSHE、プロダクトIDがTST-2004、ルーチン施策IDがTST-1005、スペシャル施策IDがTST-1006に変えて更新したとき、デッキを取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got *domain.Deck)
				}{
					{"デッキ名がテストデッキ改になる", func(t *testing.T, got *domain.Deck) { assert.Equal(t, "テストデッキ改", got.DeckName) }},
					{"陣営がSHEになる", func(t *testing.T, got *domain.Deck) { assert.Equal(t, "SHE", got.Faction) }},
					{"プロダクトIDがTST-2004になる", func(t *testing.T, got *domain.Deck) { assert.Equal(t, "TST-2004", got.ProductID) }},
					{"ルーチン施策IDがTST-1005になる", func(t *testing.T, got *domain.Deck) { assert.Equal(t, "TST-1005", got.RoutineID) }},
					{"スペシャル施策IDがTST-1006になる", func(t *testing.T, got *domain.Deck) { assert.Equal(t, "TST-1006", got.SpecialID) }},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						original := newDeck(repositorytest.PlayerOneID, "テストデッキ")
						original.Faction = "Tenki"
						original.ProductID = "TST-2001"
						original.RoutineID = "TST-1002"
						original.SpecialID = "TST-1003"
						deckID := createDeck(t, repo, original, noDeckCards)
						changed := newDeck(repositorytest.PlayerOneID, "テストデッキ改")
						changed.DeckID = deckID
						changed.Faction = "SHE"
						changed.ProductID = "TST-2004"
						changed.RoutineID = "TST-1005"
						changed.SpecialID = "TST-1006"

						require.NoError(t, repo.Update(t.Context(), changed, noDeckCards))
						got, err := repo.FindByID(t.Context(), repositorytest.PlayerOneID, deckID)

						require.NoError(t, err)
						tc.check(t, got)
					})
				}
			})

			t.Run("プレイマット番号が5、スリーブ番号が7のデッキを、2つとも未設定にして更新したとき、デッキを取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got *domain.Deck)
				}{
					{"プレイマット番号が未設定で返る", func(t *testing.T, got *domain.Deck) { assert.Nil(t, got.PlaymatNo) }},
					{"スリーブ番号が未設定で返る", func(t *testing.T, got *domain.Deck) { assert.Nil(t, got.SleeveNo) }},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						original := newDeck(repositorytest.PlayerOneID, "テストデッキ")
						original.PlaymatNo = repositorytest.ToPointer(int64(5))
						original.SleeveNo = repositorytest.ToPointer(int64(7))
						deckID := createDeck(t, repo, original, noDeckCards)
						changed := newDeck(repositorytest.PlayerOneID, "テストデッキ")
						changed.DeckID = deckID
						changed.PlaymatNo = nil
						changed.SleeveNo = nil

						require.NoError(t, repo.Update(t.Context(), changed, noDeckCards))
						got, err := repo.FindByID(t.Context(), repositorytest.PlayerOneID, deckID)

						require.NoError(t, err)
						tc.check(t, got)
					})
				}
			})

			t.Run("プレイマット番号とスリーブ番号が未設定のデッキを、プレイマット番号3、スリーブ番号9を指定して更新したとき、デッキを取得すると", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, got *domain.Deck)
				}{
					{"プレイマット番号が3になる", func(t *testing.T, got *domain.Deck) {
						assert.Equal(t, repositorytest.ToPointer(int64(3)), got.PlaymatNo)
					}},
					{"スリーブ番号が9になる", func(t *testing.T, got *domain.Deck) {
						assert.Equal(t, repositorytest.ToPointer(int64(9)), got.SleeveNo)
					}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						original := newDeck(repositorytest.PlayerOneID, "テストデッキ")
						original.PlaymatNo = nil
						original.SleeveNo = nil
						deckID := createDeck(t, repo, original, noDeckCards)
						changed := newDeck(repositorytest.PlayerOneID, "テストデッキ")
						changed.DeckID = deckID
						changed.PlaymatNo = repositorytest.ToPointer(int64(3))
						changed.SleeveNo = repositorytest.ToPointer(int64(9))

						require.NoError(t, repo.Update(t.Context(), changed, noDeckCards))
						got, err := repo.FindByID(t.Context(), repositorytest.PlayerOneID, deckID)

						require.NoError(t, err)
						tc.check(t, got)
					})
				}
			})

			t.Run("作成日時が2026-01-02T03:04:05Zのデッキを更新したとき、デッキを取得すると、作成日時は2026-01-02T03:04:05Zのままになる", func(t *testing.T) {
				repo := repository.NewPostgresDeckRepository(preparePool(t))
				original := newDeck(repositorytest.PlayerOneID, "テストデッキ")
				original.CreatedAt = repositorytest.ParseTime(t, "2026-01-02T03:04:05Z")
				deckID := createDeck(t, repo, original, noDeckCards)
				changed := newDeck(repositorytest.PlayerOneID, "テストデッキ改")
				changed.DeckID = deckID

				require.NoError(t, repo.Update(t.Context(), changed, noDeckCards))
				got, err := repo.FindByID(t.Context(), repositorytest.PlayerOneID, deckID)

				require.NoError(t, err)
				repositorytest.RequireTimeEqual(t, repositorytest.ParseTime(t, "2026-01-02T03:04:05Z"), got.CreatedAt)
			})

			t.Run("TST-0001・アート番号0・枚数2のカード構成を持つデッキを更新したあと、カード構成を取得すると", func(t *testing.T) {
				cases := []struct {
					name     string
					replaced []domain.DeckCardEntry
				}{
					{
						"カード構成をTST-0002・アート番号0・枚数1の1件に差し替えると、TST-0002・アート番号0・枚数1の1件が返る",
						[]domain.DeckCardEntry{newDeckCardEntry("TST-0002", 0, 1)},
					},
					{
						"カード構成をTST-0002・アート番号0・枚数1、TST-0002・アート番号1・枚数4、TST-0003・アート番号0・枚数3の3件に差し替えると、TST-0002・アート番号0・枚数1、TST-0002・アート番号1・枚数4、TST-0003・アート番号0・枚数3の3件が返る",
						[]domain.DeckCardEntry{newDeckCardEntry("TST-0002", 0, 1), newDeckCardEntry("TST-0002", 1, 4), newDeckCardEntry("TST-0003", 0, 3)},
					},
					{"カード構成を0件にすると、空の一覧が返る", []domain.DeckCardEntry{}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						deckID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキ"), []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)})
						changed := newDeck(repositorytest.PlayerOneID, "テストデッキ")
						changed.DeckID = deckID

						require.NoError(t, repo.Update(t.Context(), changed, tc.replaced))

						requireDeckCards(t, repo, repositorytest.PlayerOneID, deckID, tc.replaced)
					})
				}
			})

			t.Run("デッキ名がテストデッキAでカード構成がTST-0001・アート番号0・枚数2のデッキと、デッキ名がテストデッキBでカード構成がTST-0002・アート番号0・枚数3のデッキが同じプレイヤーにあるとき、テストデッキAをデッキ名テストデッキ改、カード構成TST-0003・アート番号0・枚数1に更新したあと", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, repo *repository.PostgresDeckRepository, deckBID int64)
				}{
					{"テストデッキBを取得すると、デッキ名がテストデッキBのまま返る", func(t *testing.T, repo *repository.PostgresDeckRepository, deckBID int64) {
						requireDeckName(t, repo, repositorytest.PlayerOneID, deckBID, "テストデッキB")
					}},
					{"テストデッキBのカード構成を取得すると、TST-0002・アート番号0・枚数3の1件のまま返る", func(t *testing.T, repo *repository.PostgresDeckRepository, deckBID int64) {
						requireDeckCards(t, repo, repositorytest.PlayerOneID, deckBID, []domain.DeckCardEntry{newDeckCardEntry("TST-0002", 0, 3)})
					}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						deckAID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキA"), []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)})
						deckBID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキB"), []domain.DeckCardEntry{newDeckCardEntry("TST-0002", 0, 3)})
						changed := newDeck(repositorytest.PlayerOneID, "テストデッキ改")
						changed.DeckID = deckAID

						require.NoError(t, repo.Update(t.Context(), changed, []domain.DeckCardEntry{newDeckCardEntry("TST-0003", 0, 1)}))

						tc.check(t, repo, deckBID)
					})
				}
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("デッキを更新するとき", func(t *testing.T) {
				cases := []struct {
					name    string
					prepare deckPreparation
				}{
					{"保存されていないデッキID999999を指定すると、存在しないことを示すエラーが返る", prepareAbsentDeck},
					{"プレイヤー00000000-0000-0000-0000-000000000001が持つデッキのデッキIDを、持ち主ではないプレイヤー00000000-0000-0000-0000-000000000002のプレイヤーIDで指定すると、存在しないことを示すエラーが返る", prepareDeckOfOtherPlayer},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						playerID, deckID := tc.prepare(t, repo)
						changed := newDeck(playerID, "テストデッキ改")
						changed.DeckID = deckID

						err := repo.Update(t.Context(), changed, noDeckCards)

						assert.ErrorIs(t, err, port.ErrNotFound)
					})
				}
			})

			t.Run("プレイヤー00000000-0000-0000-0000-000000000001が持つ、デッキ名がテストデッキでカード構成がTST-0001・アート番号0・枚数2のデッキを、持ち主ではないプレイヤー00000000-0000-0000-0000-000000000002のプレイヤーIDで、デッキ名テストデッキ改、カード構成TST-0003・アート番号0・枚数1に更新したとき", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64)
				}{
					{"持ち主のプレイヤーIDでデッキを取得すると、デッキ名がテストデッキのまま返る", func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64) {
						requireDeckName(t, repo, repositorytest.PlayerOneID, deckID, "テストデッキ")
					}},
					{"持ち主のプレイヤーIDでカード構成を取得すると、TST-0001・アート番号0・枚数2の1件のまま返る", func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64) {
						requireDeckCards(t, repo, repositorytest.PlayerOneID, deckID, []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)})
					}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						deckID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキ"), []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)})
						changed := newDeck(repositorytest.PlayerTwoID, "テストデッキ改")
						changed.DeckID = deckID

						_ = repo.Update(t.Context(), changed, []domain.DeckCardEntry{newDeckCardEntry("TST-0003", 0, 1)})

						tc.check(t, repo, deckID)
					})
				}
			})

			t.Run("デッキ名がテストデッキでカード構成がTST-0001・アート番号0・枚数2のデッキを、TST-0002・アート番号0・枚数1とTST-0002・アート番号0・枚数3を含むカード構成で、デッキ名テストデッキ改に更新するとき", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64, updateErr error)
				}{
					{"更新すると、同じカードIDとアート番号の組が重複していることを示すエラーが返る", func(t *testing.T, _ *repository.PostgresDeckRepository, _ int64, updateErr error) {
						repositorytest.RequireSQLState(t, updateErr, repositorytest.SQLStateUniqueViolation)
					}},
					{"更新したあと、カード構成を取得すると、TST-0001・アート番号0・枚数2の1件のまま返る", func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64, _ error) {
						requireDeckCards(t, repo, repositorytest.PlayerOneID, deckID, []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)})
					}},
					{"更新したあと、デッキを取得すると、デッキ名がテストデッキのまま返る", func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64, _ error) {
						requireDeckName(t, repo, repositorytest.PlayerOneID, deckID, "テストデッキ")
					}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						deckID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキ"), []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)})
						changed := newDeck(repositorytest.PlayerOneID, "テストデッキ改")
						changed.DeckID = deckID

						updateErr := repo.Update(t.Context(), changed, []domain.DeckCardEntry{newDeckCardEntry("TST-0002", 0, 1), newDeckCardEntry("TST-0002", 0, 3)})

						tc.check(t, repo, deckID, updateErr)
					})
				}
			})

			t.Run("TST-0002・アート番号0・枚数1とTST-0002・アート番号0・枚数3を含むカード構成での更新に失敗したあと、TST-0002・アート番号0・枚数1だけのカード構成でデッキ名テストデッキ改に更新したとき", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64)
				}{
					{"デッキを取得すると、デッキ名がテストデッキ改になる", func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64) {
						requireDeckName(t, repo, repositorytest.PlayerOneID, deckID, "テストデッキ改")
					}},
					{"カード構成を取得すると、TST-0002・アート番号0・枚数1の1件が返る", func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64) {
						requireDeckCards(t, repo, repositorytest.PlayerOneID, deckID, []domain.DeckCardEntry{newDeckCardEntry("TST-0002", 0, 1)})
					}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						deckID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキ"), []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)})
						changed := newDeck(repositorytest.PlayerOneID, "テストデッキ改")
						changed.DeckID = deckID
						failedErr := repo.Update(t.Context(), changed, []domain.DeckCardEntry{newDeckCardEntry("TST-0002", 0, 1), newDeckCardEntry("TST-0002", 0, 3)})
						repositorytest.RequireSQLState(t, failedErr, repositorytest.SQLStateUniqueViolation)

						require.NoError(t, repo.Update(t.Context(), changed, []domain.DeckCardEntry{newDeckCardEntry("TST-0002", 0, 1)}))

						tc.check(t, repo, deckID)
					})
				}
			})
		})
	})
}

func TestPostgresDeckRepositoryDelete(t *testing.T) {
	t.Run("[デッキ]デッキの削除", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("TST-0001・アート番号0・枚数2のカード構成を持つデッキを削除したあと、同じデッキIDで", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64)
				}{
					{"取得すると、存在しないことを示すエラーになる", func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64) {
						_, err := repo.FindByID(t.Context(), repositorytest.PlayerOneID, deckID)
						assert.ErrorIs(t, err, port.ErrNotFound)
					}},
					{"カード構成を取得すると、空の一覧が返る", func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64) {
						requireDeckCards(t, repo, repositorytest.PlayerOneID, deckID, []domain.DeckCardEntry{})
					}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						deckID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキ"), []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)})

						require.NoError(t, repo.Delete(t.Context(), repositorytest.PlayerOneID, deckID))

						tc.check(t, repo, deckID)
					})
				}
			})

			t.Run("デッキ名がテストデッキAでカード構成がTST-0001・アート番号0・枚数2のデッキと、デッキ名がテストデッキBでカード構成がTST-0002・アート番号0・枚数3のデッキが同じプレイヤーにあるとき、テストデッキAを削除したあと", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, repo *repository.PostgresDeckRepository, deckBID int64)
				}{
					{"テストデッキBを取得すると、デッキ名がテストデッキBのまま返る", func(t *testing.T, repo *repository.PostgresDeckRepository, deckBID int64) {
						requireDeckName(t, repo, repositorytest.PlayerOneID, deckBID, "テストデッキB")
					}},
					{"テストデッキBのカード構成を取得すると、TST-0002・アート番号0・枚数3の1件のまま返る", func(t *testing.T, repo *repository.PostgresDeckRepository, deckBID int64) {
						requireDeckCards(t, repo, repositorytest.PlayerOneID, deckBID, []domain.DeckCardEntry{newDeckCardEntry("TST-0002", 0, 3)})
					}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						deckAID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキA"), []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)})
						deckBID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキB"), []domain.DeckCardEntry{newDeckCardEntry("TST-0002", 0, 3)})

						require.NoError(t, repo.Delete(t.Context(), repositorytest.PlayerOneID, deckAID))

						tc.check(t, repo, deckBID)
					})
				}
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("デッキを削除するとき", func(t *testing.T) {
				cases := []struct {
					name    string
					prepare deckPreparation
				}{
					{"保存されていないデッキID999999を指定すると、存在しないことを示すエラーが返る", prepareAbsentDeck},
					{"プレイヤー00000000-0000-0000-0000-000000000001が持つデッキのデッキIDを、持ち主ではないプレイヤー00000000-0000-0000-0000-000000000002のプレイヤーIDで指定すると、存在しないことを示すエラーが返る", prepareDeckOfOtherPlayer},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						playerID, deckID := tc.prepare(t, repo)

						err := repo.Delete(t.Context(), playerID, deckID)

						assert.ErrorIs(t, err, port.ErrNotFound)
					})
				}
			})

			t.Run("プレイヤー00000000-0000-0000-0000-000000000001が持つ、デッキ名がテストデッキでカード構成がTST-0001・アート番号0・枚数2のデッキを、持ち主ではないプレイヤー00000000-0000-0000-0000-000000000002のプレイヤーIDで削除したとき", func(t *testing.T) {
				cases := []struct {
					name  string
					check func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64)
				}{
					{"持ち主のプレイヤーIDでデッキを取得すると、デッキ名がテストデッキのまま返る", func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64) {
						requireDeckName(t, repo, repositorytest.PlayerOneID, deckID, "テストデッキ")
					}},
					{"持ち主のプレイヤーIDでカード構成を取得すると、TST-0001・アート番号0・枚数2の1件のまま返る", func(t *testing.T, repo *repository.PostgresDeckRepository, deckID int64) {
						requireDeckCards(t, repo, repositorytest.PlayerOneID, deckID, []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)})
					}},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						repo := repository.NewPostgresDeckRepository(preparePool(t))
						deckID := createDeck(t, repo, newDeck(repositorytest.PlayerOneID, "テストデッキ"), []domain.DeckCardEntry{newDeckCardEntry("TST-0001", 0, 2)})

						_ = repo.Delete(t.Context(), repositorytest.PlayerTwoID, deckID)

						tc.check(t, repo, deckID)
					})
				}
			})
		})
	})
}
