//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

// newTestDeck は DeckRepo テスト用の domain.Deck を、PlaymatNo/SleeveNo と
// CreatedAt/UpdatedAt 以外は固定のダミー値で組み立てる。
// product_id / routine_id / special_id は FK を持たないため実在データ不要。
func newTestDeck(playerID, deckName string, playmatNo, sleeveNo *int64) domain.Deck {
	fixedAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	return domain.Deck{
		PlayerID:  playerID,
		DeckName:  deckName,
		Faction:   "SHE",
		ProductID: "PD-TST01",
		RoutineID: "IN-TST01",
		SpecialID: "IN-TST02",
		PlaymatNo: playmatNo,
		SleeveNo:  sleeveNo,
		CreatedAt: fixedAt,
		UpdatedAt: fixedAt,
	}
}

func TestDeckRepoCreate(t *testing.T) {
	t.Run("[デッキリポジトリ] デッキ作成", func(t *testing.T) {
		t.Run("デッキを複数作成すると、それぞれ異なるdeck_idが採番される", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deckA := newTestDeck(playerA, "デッキA", nil, nil)
			deckB := newTestDeck(playerA, "デッキB", nil, nil)

			idA, err := repo.Create(context.Background(), deckA, nil)
			require.NoError(t, err)
			idB, err := repo.Create(context.Background(), deckB, nil)
			require.NoError(t, err)

			require.NotEqual(t, idA, idB)
		})

		t.Run("作成したデッキを取得し直すと、指定した項目の値が、そのまま返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", int64Ptr(12), int64Ptr(34))

			deckID, err := repo.Create(context.Background(), deck, nil)
			require.NoError(t, err)

			got, err := repo.FindByID(context.Background(), playerA, deckID)
			require.NoError(t, err)
			require.Equal(t, "デッキA", got.DeckName)
			require.Equal(t, "SHE", got.Faction)
			require.Equal(t, "PD-TST01", got.ProductID)
			require.Equal(t, "IN-TST01", got.RoutineID)
			require.Equal(t, "IN-TST02", got.SpecialID)
			require.NotNil(t, got.PlaymatNo)
			require.Equal(t, int64(12), *got.PlaymatNo)
			require.NotNil(t, got.SleeveNo)
			require.Equal(t, int64(34), *got.SleeveNo)
		})

		t.Run("デッキを作成すると、呼び出し側が指定した作成日時・更新日時の値が、そのまま保存される", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)
			deck.CreatedAt = time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
			deck.UpdatedAt = time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)

			deckID, err := repo.Create(context.Background(), deck, nil)
			require.NoError(t, err)

			got, err := repo.FindByID(context.Background(), playerA, deckID)
			require.NoError(t, err)
			require.True(t, deck.CreatedAt.Equal(got.CreatedAt))
			require.True(t, deck.UpdatedAt.Equal(got.UpdatedAt))
		})

		t.Run("プレイマット番号・スリーブ番号を未設定でデッキを作成すると、取得した値も未設定になる", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)

			deckID, err := repo.Create(context.Background(), deck, nil)
			require.NoError(t, err)

			got, err := repo.FindByID(context.Background(), playerA, deckID)
			require.NoError(t, err)
			require.Nil(t, got.PlaymatNo)
			require.Nil(t, got.SleeveNo)
		})

		t.Run("構成カードを指定せずにデッキを作成すると、そのデッキの構成カードは0件になる", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)

			deckID, err := repo.Create(context.Background(), deck, []domain.DeckCardEntry{})
			require.NoError(t, err)

			cards, err := repo.GetDeckCards(context.Background(), playerA, deckID)
			require.NoError(t, err)
			require.Empty(t, cards)
		})

		t.Run("構成カードを指定してデッキを作成すると、その構成カードが指定した値のまま取得できる", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)
			entries := []domain.DeckCardEntry{
				{CardID: "TST-0001", ArtNo: 0, Count: 2},
				{CardID: "TST-0002", ArtNo: 0, Count: 1},
			}

			deckID, err := repo.Create(context.Background(), deck, entries)
			require.NoError(t, err)

			cards, err := repo.GetDeckCards(context.Background(), playerA, deckID)
			require.NoError(t, err)
			require.ElementsMatch(t, []domain.DeckCard{
				{PlayerID: playerA, DeckID: deckID, CardID: "TST-0001", ArtNo: 0, Count: 2},
				{PlayerID: playerA, DeckID: deckID, CardID: "TST-0002", ArtNo: 0, Count: 1},
			}, cards)
		})

		t.Run("同じカードID・アート番号の組を複数含む構成カードを指定してデッキを作成しようとすると、エラーを返し、デッキは作成されない", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)
			entries := []domain.DeckCardEntry{
				{CardID: "TST-0001", ArtNo: 0, Count: 2},
				{CardID: "TST-0001", ArtNo: 0, Count: 1},
			}

			_, err := repo.Create(context.Background(), deck, entries)
			require.Error(t, err)

			decks, err := repo.FindByPlayerID(context.Background(), playerA)
			require.NoError(t, err)
			require.Empty(t, decks)
		})

		t.Run("同じカードID・アート番号の組を複数含む構成でのデッキ作成に失敗した後、重複のない構成で改めて作成すると、デッキが作成される", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)
			duplicateEntries := []domain.DeckCardEntry{
				{CardID: "TST-0001", ArtNo: 0, Count: 2},
				{CardID: "TST-0001", ArtNo: 0, Count: 1},
			}
			_, err := repo.Create(context.Background(), deck, duplicateEntries)
			require.Error(t, err)

			validEntries := []domain.DeckCardEntry{
				{CardID: "TST-0001", ArtNo: 0, Count: 2},
			}
			deckID, err := repo.Create(context.Background(), deck, validEntries)

			require.NoError(t, err)
			got, err := repo.FindByID(context.Background(), playerA, deckID)
			require.NoError(t, err)
			require.Equal(t, "デッキA", got.DeckName)
		})
	})
}

func TestDeckRepoFindByPlayerID(t *testing.T) {
	t.Run("[デッキリポジトリ] プレイヤーのデッキ一覧取得", func(t *testing.T) {
		t.Run("指定プレイヤーのデッキが0件のとき、空の一覧を返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)

			got, err := repo.FindByPlayerID(context.Background(), playerA)

			require.NoError(t, err)
			require.Empty(t, got)
		})

		t.Run("指定プレイヤーのデッキがあるとき、それらを返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)
			deckID, err := repo.Create(context.Background(), deck, nil)
			require.NoError(t, err)

			got, err := repo.FindByPlayerID(context.Background(), playerA)

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.Equal(t, deckID, got[0].DeckID)
		})

		t.Run("別プレイヤーのデッキは、返る一覧に含まれない", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerB, "デッキB", nil, nil)
			_, err := repo.Create(context.Background(), deck, nil)
			require.NoError(t, err)

			got, err := repo.FindByPlayerID(context.Background(), playerA)

			require.NoError(t, err)
			require.Empty(t, got)
		})

		t.Run("updated_atが異なる複数のデッキがあるとき、updated_atの降順で返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deckOld := newTestDeck(playerA, "デッキOld", nil, nil)
			deckOld.UpdatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			deckMid := newTestDeck(playerA, "デッキMid", nil, nil)
			deckMid.UpdatedAt = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
			deckNew := newTestDeck(playerA, "デッキNew", nil, nil)
			deckNew.UpdatedAt = time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

			idOld, err := repo.Create(context.Background(), deckOld, nil)
			require.NoError(t, err)
			idNew, err := repo.Create(context.Background(), deckNew, nil)
			require.NoError(t, err)
			idMid, err := repo.Create(context.Background(), deckMid, nil)
			require.NoError(t, err)

			got, err := repo.FindByPlayerID(context.Background(), playerA)

			require.NoError(t, err)
			require.Len(t, got, 3)
			require.Equal(t, []int64{idNew, idMid, idOld},
				[]int64{got[0].DeckID, got[1].DeckID, got[2].DeckID})
		})
	})
}

func TestDeckRepoFindByID(t *testing.T) {
	t.Run("[デッキリポジトリ] デッキ取得", func(t *testing.T) {
		t.Run("指定したplayer_id・deck_idの組に対応するデッキが存在するとき、そのデッキを返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			deckID := insertDeck(t, playerA, "デッキA")
			repo := repository.NewPgDeckRepository(sharedPg.Pool)

			got, err := repo.FindByID(context.Background(), playerA, deckID)

			require.NoError(t, err)
			require.Equal(t, playerA, got.PlayerID)
			require.Equal(t, deckID, got.DeckID)
		})

		t.Run("指定したdeck_idのデッキが存在しないとき、port.ErrNotFoundを返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)

			_, err := repo.FindByID(context.Background(), playerA, 999999)

			require.ErrorIs(t, err, port.ErrNotFound)
		})

		t.Run("指定したdeck_idのデッキが別プレイヤーのものであるとき、port.ErrNotFoundを返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			deckID := insertDeck(t, playerA, "デッキA")
			repo := repository.NewPgDeckRepository(sharedPg.Pool)

			_, err := repo.FindByID(context.Background(), playerB, deckID)

			require.ErrorIs(t, err, port.ErrNotFound)
		})
	})
}

func TestDeckRepoGetDeckCards(t *testing.T) {
	t.Run("[デッキリポジトリ] デッキ構成カード取得", func(t *testing.T) {
		t.Run("指定デッキの構成カードが0件のとき、空の一覧を返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			deckID := insertDeck(t, playerA, "デッキA")
			repo := repository.NewPgDeckRepository(sharedPg.Pool)

			got, err := repo.GetDeckCards(context.Background(), playerA, deckID)

			require.NoError(t, err)
			require.Empty(t, got)
		})

		t.Run("指定デッキの構成カードがあるとき、それらを返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			deckID := insertDeck(t, playerA, "デッキA")
			seedDeckCard(t, deckCardSeed{PlayerID: playerA, DeckID: deckID, CardID: "TST-0001", ArtNo: 0, Count: 2})
			seedDeckCard(t, deckCardSeed{PlayerID: playerA, DeckID: deckID, CardID: "TST-0002", ArtNo: 0, Count: 1})
			repo := repository.NewPgDeckRepository(sharedPg.Pool)

			got, err := repo.GetDeckCards(context.Background(), playerA, deckID)

			require.NoError(t, err)
			require.ElementsMatch(t, []domain.DeckCard{
				{PlayerID: playerA, DeckID: deckID, CardID: "TST-0001", ArtNo: 0, Count: 2},
				{PlayerID: playerA, DeckID: deckID, CardID: "TST-0002", ArtNo: 0, Count: 1},
			}, got)
		})

		t.Run("別デッキ・別プレイヤーの構成カードは、返る一覧に含まれない", func(t *testing.T) {
			sharedPg.Truncate(t)
			deckIDA := insertDeck(t, playerA, "デッキA")
			deckIDOther := insertDeck(t, playerA, "デッキ別")
			deckIDB := insertDeck(t, playerB, "デッキB")
			seedDeckCard(t, deckCardSeed{PlayerID: playerA, DeckID: deckIDA, CardID: "TST-0001", ArtNo: 0, Count: 2})
			seedDeckCard(t, deckCardSeed{PlayerID: playerA, DeckID: deckIDOther, CardID: "TST-0002", ArtNo: 0, Count: 1})
			seedDeckCard(t, deckCardSeed{PlayerID: playerB, DeckID: deckIDB, CardID: "TST-0003", ArtNo: 0, Count: 1})
			repo := repository.NewPgDeckRepository(sharedPg.Pool)

			got, err := repo.GetDeckCards(context.Background(), playerA, deckIDA)

			require.NoError(t, err)
			require.Equal(t, []domain.DeckCard{
				{PlayerID: playerA, DeckID: deckIDA, CardID: "TST-0001", ArtNo: 0, Count: 2},
			}, got)
		})
	})
}

func TestDeckRepoUpdate(t *testing.T) {
	t.Run("[デッキリポジトリ] デッキ更新", func(t *testing.T) {
		t.Run("存在するデッキを更新すると、指定した項目が更新後の値になる", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)
			deckID, err := repo.Create(context.Background(), deck, nil)
			require.NoError(t, err)

			updated := deck
			updated.DeckID = deckID
			updated.DeckName = "デッキA改"
			updated.Faction = "Tenki"
			updated.ProductID = "PD-TST02"
			updated.RoutineID = "IN-TST03"
			updated.SpecialID = "IN-TST04"
			updated.PlaymatNo = int64Ptr(56)
			updated.SleeveNo = int64Ptr(78)

			err = repo.Update(context.Background(), updated, nil)
			require.NoError(t, err)

			got, err := repo.FindByID(context.Background(), playerA, deckID)
			require.NoError(t, err)
			require.Equal(t, "デッキA改", got.DeckName)
			require.Equal(t, "Tenki", got.Faction)
			require.Equal(t, "PD-TST02", got.ProductID)
			require.Equal(t, "IN-TST03", got.RoutineID)
			require.Equal(t, "IN-TST04", got.SpecialID)
			require.NotNil(t, got.PlaymatNo)
			require.Equal(t, int64(56), *got.PlaymatNo)
			require.NotNil(t, got.SleeveNo)
			require.Equal(t, int64(78), *got.SleeveNo)
		})

		t.Run("呼び出し側が指定したupdated_atの値は保存に使われず、更新時点の日時に置き換わる", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)
			deckID, err := repo.Create(context.Background(), deck, nil)
			require.NoError(t, err)

			updated := deck
			updated.DeckID = deckID
			updated.UpdatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

			err = repo.Update(context.Background(), updated, nil)
			require.NoError(t, err)

			got, err := repo.FindByID(context.Background(), playerA, deckID)
			require.NoError(t, err)
			require.WithinDuration(t, time.Now(), got.UpdatedAt, 5*time.Minute)
		})

		t.Run("更新前の構成カードは、新しく指定した構成カードの内容で完全に置き換わる", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)
			deckID, err := repo.Create(context.Background(), deck, []domain.DeckCardEntry{
				{CardID: "TST-0001", ArtNo: 0, Count: 2},
			})
			require.NoError(t, err)

			updated := deck
			updated.DeckID = deckID
			err = repo.Update(context.Background(), updated, []domain.DeckCardEntry{
				{CardID: "TST-0002", ArtNo: 0, Count: 3},
			})
			require.NoError(t, err)

			cards, err := repo.GetDeckCards(context.Background(), playerA, deckID)
			require.NoError(t, err)
			require.Equal(t, []domain.DeckCard{
				{PlayerID: playerA, DeckID: deckID, CardID: "TST-0002", ArtNo: 0, Count: 3},
			}, cards)
		})

		t.Run("構成カードを指定せずに更新すると、構成カードは全件削除され0件になる", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)
			deckID, err := repo.Create(context.Background(), deck, []domain.DeckCardEntry{
				{CardID: "TST-0001", ArtNo: 0, Count: 2},
			})
			require.NoError(t, err)

			updated := deck
			updated.DeckID = deckID
			err = repo.Update(context.Background(), updated, []domain.DeckCardEntry{})
			require.NoError(t, err)

			cards, err := repo.GetDeckCards(context.Background(), playerA, deckID)
			require.NoError(t, err)
			require.Empty(t, cards)
		})

		t.Run("設定されていたプレイマット番号・スリーブ番号を未設定にして更新すると、取得した値も未設定になる", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", int64Ptr(1), int64Ptr(2))
			deckID, err := repo.Create(context.Background(), deck, nil)
			require.NoError(t, err)

			updated := deck
			updated.DeckID = deckID
			updated.PlaymatNo = nil
			updated.SleeveNo = nil
			err = repo.Update(context.Background(), updated, nil)
			require.NoError(t, err)

			got, err := repo.FindByID(context.Background(), playerA, deckID)
			require.NoError(t, err)
			require.Nil(t, got.PlaymatNo)
			require.Nil(t, got.SleeveNo)
		})

		t.Run("存在するデッキに対し、同じカードID・アート番号の組を複数含む構成で更新しようとすると、エラーを返し、更新前の構成カードがそのまま取得できる", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)
			deckID, err := repo.Create(context.Background(), deck, []domain.DeckCardEntry{
				{CardID: "TST-0001", ArtNo: 0, Count: 2},
			})
			require.NoError(t, err)

			updated := deck
			updated.DeckID = deckID
			err = repo.Update(context.Background(), updated, []domain.DeckCardEntry{
				{CardID: "TST-0002", ArtNo: 0, Count: 1},
				{CardID: "TST-0002", ArtNo: 0, Count: 4},
			})
			require.Error(t, err)

			cards, err := repo.GetDeckCards(context.Background(), playerA, deckID)
			require.NoError(t, err)
			require.Equal(t, []domain.DeckCard{
				{PlayerID: playerA, DeckID: deckID, CardID: "TST-0001", ArtNo: 0, Count: 2},
			}, cards)
		})

		t.Run("同じカードID・アート番号の組を複数含む構成での更新に失敗した後、重複のない構成で改めて更新すると、その構成に置き換わる", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)
			deckID, err := repo.Create(context.Background(), deck, []domain.DeckCardEntry{
				{CardID: "TST-0001", ArtNo: 0, Count: 2},
			})
			require.NoError(t, err)
			updated := deck
			updated.DeckID = deckID
			duplicateErr := repo.Update(context.Background(), updated, []domain.DeckCardEntry{
				{CardID: "TST-0002", ArtNo: 0, Count: 1},
				{CardID: "TST-0002", ArtNo: 0, Count: 4},
			})
			require.Error(t, duplicateErr)

			err = repo.Update(context.Background(), updated, []domain.DeckCardEntry{
				{CardID: "TST-0002", ArtNo: 0, Count: 4},
			})

			require.NoError(t, err)
			cards, err := repo.GetDeckCards(context.Background(), playerA, deckID)
			require.NoError(t, err)
			require.Equal(t, []domain.DeckCard{
				{PlayerID: playerA, DeckID: deckID, CardID: "TST-0002", ArtNo: 0, Count: 4},
			}, cards)
		})

		t.Run("存在しないplayer_id・deck_idの組に対して更新しようとすると、構成カードが0件のときport.ErrNotFoundを返し、更新は行われない", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)
			deck.DeckID = 999999

			err := repo.Update(context.Background(), deck, []domain.DeckCardEntry{})

			require.ErrorIs(t, err, port.ErrNotFound)
			require.Equal(t, 0, countRows(t, "card.deck_cards"))
		})

		t.Run("存在しないplayer_id・deck_idの組に対して更新しようとすると、構成カードが1件以上のときport.ErrNotFoundを返し、更新は行われない", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)
			deck.DeckID = 999999

			err := repo.Update(context.Background(), deck, []domain.DeckCardEntry{
				{CardID: "TST-0001", ArtNo: 0, Count: 1},
			})

			require.ErrorIs(t, err, port.ErrNotFound)
			require.Equal(t, 0, countRows(t, "card.deck_cards"))
		})
	})
}

func TestDeckRepoDelete(t *testing.T) {
	t.Run("[デッキリポジトリ] デッキ削除", func(t *testing.T) {
		t.Run("存在するデッキを削除すると、そのデッキは取得できなくなる", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)
			deckID, err := repo.Create(context.Background(), deck, nil)
			require.NoError(t, err)

			err = repo.Delete(context.Background(), playerA, deckID)
			require.NoError(t, err)

			_, err = repo.FindByID(context.Background(), playerA, deckID)
			require.ErrorIs(t, err, port.ErrNotFound)
		})

		t.Run("デッキを削除すると、その構成カードも0件になる", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deck := newTestDeck(playerA, "デッキA", nil, nil)
			deckID, err := repo.Create(context.Background(), deck, []domain.DeckCardEntry{
				{CardID: "TST-0001", ArtNo: 0, Count: 2},
			})
			require.NoError(t, err)

			err = repo.Delete(context.Background(), playerA, deckID)
			require.NoError(t, err)

			cards, err := repo.GetDeckCards(context.Background(), playerA, deckID)
			require.NoError(t, err)
			require.Empty(t, cards)
		})

		t.Run("別プレイヤーのデッキを削除しても、別プレイヤーのデッキ・構成カードは削除されず取得できたままになる", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)
			deckA := newTestDeck(playerA, "デッキA", nil, nil)
			deckIDA, err := repo.Create(context.Background(), deckA, nil)
			require.NoError(t, err)
			deckB := newTestDeck(playerB, "デッキB", nil, nil)
			deckIDB, err := repo.Create(context.Background(), deckB, []domain.DeckCardEntry{
				{CardID: "TST-0001", ArtNo: 0, Count: 2},
			})
			require.NoError(t, err)

			err = repo.Delete(context.Background(), playerA, deckIDA)
			require.NoError(t, err)

			got, err := repo.FindByID(context.Background(), playerB, deckIDB)
			require.NoError(t, err)
			require.Equal(t, deckIDB, got.DeckID)
			cards, err := repo.GetDeckCards(context.Background(), playerB, deckIDB)
			require.NoError(t, err)
			require.Len(t, cards, 1)
		})

		t.Run("存在しないplayer_id・deck_idの組に対して削除を実行すると、port.ErrNotFoundを返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgDeckRepository(sharedPg.Pool)

			err := repo.Delete(context.Background(), playerA, 999999)

			require.ErrorIs(t, err, port.ErrNotFound)
		})
	})
}
