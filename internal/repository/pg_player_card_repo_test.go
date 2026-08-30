//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

func TestPlayerCardRepoGetPlayerCards(t *testing.T) {
	t.Run("[所持カードリポジトリ] 所持カード取得", func(t *testing.T) {
		t.Run("指定プレイヤーの所持カードが0件のとき、空の一覧を返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)

			got, err := repo.GetPlayerCards(context.Background(), playerA)

			require.NoError(t, err)
			require.Empty(t, got)
		})

		t.Run("指定プレイヤーの所持カードがあるとき、それらを返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedPlayerCard(t, playerCardSeed{PlayerID: playerA, CardID: "TST-0001", ArtNo: 0, Count: 3})
			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)

			got, err := repo.GetPlayerCards(context.Background(), playerA)

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.Equal(t, "TST-0001", got[0].CardID)
		})

		t.Run("別プレイヤーの所持カードは、返る一覧に含まれない", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedPlayerCard(t, playerCardSeed{PlayerID: playerB, CardID: "TST-0001", ArtNo: 0, Count: 3})
			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)

			got, err := repo.GetPlayerCards(context.Background(), playerA)

			require.NoError(t, err)
			require.Empty(t, got)
		})

		t.Run("card_idが異なる所持カードが複数件あるとき、card_idの昇順で返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedPlayerCard(t, playerCardSeed{PlayerID: playerA, CardID: "TST-0003", ArtNo: 0, Count: 1})
			seedPlayerCard(t, playerCardSeed{PlayerID: playerA, CardID: "TST-0001", ArtNo: 0, Count: 1})
			seedPlayerCard(t, playerCardSeed{PlayerID: playerA, CardID: "TST-0002", ArtNo: 0, Count: 1})
			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)

			got, err := repo.GetPlayerCards(context.Background(), playerA)

			require.NoError(t, err)
			require.Len(t, got, 3)
			require.Equal(t, []string{"TST-0001", "TST-0002", "TST-0003"},
				[]string{got[0].CardID, got[1].CardID, got[2].CardID})
		})

		t.Run("同じcard_idでart_noが異なる所持カードが複数件あるとき、art_noの昇順で返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedPlayerCard(t, playerCardSeed{PlayerID: playerA, CardID: "TST-0001", ArtNo: 2, Count: 1})
			seedPlayerCard(t, playerCardSeed{PlayerID: playerA, CardID: "TST-0001", ArtNo: 0, Count: 1})
			seedPlayerCard(t, playerCardSeed{PlayerID: playerA, CardID: "TST-0001", ArtNo: 1, Count: 1})
			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)

			got, err := repo.GetPlayerCards(context.Background(), playerA)

			require.NoError(t, err)
			require.Len(t, got, 3)
			require.Equal(t, []int64{0, 1, 2}, []int64{got[0].ArtNo, got[1].ArtNo, got[2].ArtNo})
		})

		t.Run("保存したプレイヤーID・カードID・アート番号・枚数の値が、そのまま返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedPlayerCard(t, playerCardSeed{PlayerID: playerA, CardID: "TST-0001", ArtNo: 3, Count: 9})
			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)

			got, err := repo.GetPlayerCards(context.Background(), playerA)

			require.NoError(t, err)
			require.Len(t, got, 1)
			require.Equal(t, &domain.PlayerCard{PlayerID: playerA, CardID: "TST-0001", ArtNo: 3, Count: 9}, got[0])
		})
	})
}

func TestPlayerCardRepoAddCards(t *testing.T) {
	t.Run("[所持カードリポジトリ] 所持カード加算", func(t *testing.T) {
		t.Run("空のカード一覧を渡したとき、所持カードの枚数は変化せず、加算したコピー総数として0を返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedPlayerCard(t, playerCardSeed{PlayerID: playerA, CardID: "TST-0001", ArtNo: 0, Count: 5})
			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)

			total, err := repo.AddCards(context.Background(), playerA, []domain.CardPackCard{})

			require.NoError(t, err)
			require.Equal(t, 0, total)
			count, ok := fetchPlayerCardCount(t, playerA, "TST-0001", 0)
			require.True(t, ok)
			require.Equal(t, 5, count)
		})

		t.Run("指定したカードをアート番号0で所持していないプレイヤーに対して加算すると、指定した枚数でアート番号0の所持カード行が新規に作られる", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)

			_, err := repo.AddCards(context.Background(), playerA, []domain.CardPackCard{{CardID: "TST-0001", Copies: 4}})

			require.NoError(t, err)
			count, ok := fetchPlayerCardCount(t, playerA, "TST-0001", 0)
			require.True(t, ok)
			require.Equal(t, 4, count)
		})

		t.Run("指定したカードを既にアート番号0で所持しているプレイヤーに対して加算すると、既存の所持枚数に指定した枚数が加算される", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedPlayerCard(t, playerCardSeed{PlayerID: playerA, CardID: "TST-0001", ArtNo: 0, Count: 5})
			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)

			_, err := repo.AddCards(context.Background(), playerA, []domain.CardPackCard{{CardID: "TST-0001", Copies: 3}})

			require.NoError(t, err)
			count, ok := fetchPlayerCardCount(t, playerA, "TST-0001", 0)
			require.True(t, ok)
			require.Equal(t, 8, count)
		})

		t.Run("指定したカードをアート番号0以外で所持しているプレイヤーに対して加算すると、そのアート番号の所持枚数は変化せず、別途アート番号0の所持カード行が新規に作られる", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedPlayerCard(t, playerCardSeed{PlayerID: playerA, CardID: "TST-0001", ArtNo: 5, Count: 2})
			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)

			_, err := repo.AddCards(context.Background(), playerA, []domain.CardPackCard{{CardID: "TST-0001", Copies: 3}})

			require.NoError(t, err)
			artNo5Count, ok := fetchPlayerCardCount(t, playerA, "TST-0001", 5)
			require.True(t, ok)
			require.Equal(t, 2, artNo5Count)
			artNo0Count, ok := fetchPlayerCardCount(t, playerA, "TST-0001", 0)
			require.True(t, ok)
			require.Equal(t, 3, artNo0Count)
		})

		t.Run("複数カードを一括で加算したとき、それぞれのカードの枚数が個別に加算される", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)

			_, err := repo.AddCards(context.Background(), playerA, []domain.CardPackCard{
				{CardID: "TST-0001", Copies: 2},
				{CardID: "TST-0002", Copies: 5},
			})

			require.NoError(t, err)
			count1, ok := fetchPlayerCardCount(t, playerA, "TST-0001", 0)
			require.True(t, ok)
			require.Equal(t, 2, count1)
			count2, ok := fetchPlayerCardCount(t, playerA, "TST-0002", 0)
			require.True(t, ok)
			require.Equal(t, 5, count2)
		})

		t.Run("複数カードを一括で加算したとき、戻り値は各カードの加算枚数の合計値になる", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)

			total, err := repo.AddCards(context.Background(), playerA, []domain.CardPackCard{
				{CardID: "TST-0001", Copies: 2},
				{CardID: "TST-0002", Copies: 5},
			})

			require.NoError(t, err)
			require.Equal(t, 7, total)
		})

		t.Run("別プレイヤーの所持カードを加算しても、別プレイヤーの所持枚数は変化しない", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedPlayerCard(t, playerCardSeed{PlayerID: playerB, CardID: "TST-0001", ArtNo: 0, Count: 5})
			repo := repository.NewPgPlayerCardRepository(sharedPg.Pool)

			_, err := repo.AddCards(context.Background(), playerA, []domain.CardPackCard{{CardID: "TST-0001", Copies: 3}})

			require.NoError(t, err)
			count, ok := fetchPlayerCardCount(t, playerB, "TST-0001", 0)
			require.True(t, ok)
			require.Equal(t, 5, count)
		})
	})
}
