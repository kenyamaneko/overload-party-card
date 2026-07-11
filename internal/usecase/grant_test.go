package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// fakeCardPackRepo は CardPackRepo のテスト用スタブ。
type fakeCardPackRepo struct {
	pack *domain.CardPack
	err  error
}

func (f *fakeCardPackRepo) GetPack(_ context.Context, _ string) (*domain.CardPack, error) {
	return f.pack, f.err
}

// assertNoCardsGranted は、指定プレイヤーへ 1 枚も配布されていないこと (所持
// コレクションが空のまま) を検証します。ガード条件が配布を回避したことを、呼び出し
// フラグでなく永続状態の不変条件として固定します。
func assertNoCardsGranted(t *testing.T, repo *inMemoryPlayerCardRepo, playerID string) {
	t.Helper()
	owned, err := repo.GetPlayerCards(context.Background(), playerID)
	require.NoError(t, err)
	assert.Empty(t, owned)
}

func TestGrantPack(t *testing.T) {
	t.Run("パック付与", func(t *testing.T) {
		t.Run("有効な pack のとき、pack の全カードが所持枚数として永続化され付与コピー総数を返す", func(t *testing.T) {
			// pack 内でカードごとに枚数が異なるケースで、写し漏れを検出する。
			packRepo := &fakeCardPackRepo{pack: &domain.CardPack{
				IsActive: true,
				Cards: []domain.CardPackCard{
					{CardID: "TST-0001", Copies: 3},
					{CardID: "TST-0002", Copies: 1},
				},
			}}
			pcRepo := newInMemoryPlayerCardRepo()

			svc := NewGrantInteractor(packRepo, pcRepo)
			got, err := svc.GrantPack(context.Background(), "player-1", "any")

			require.NoError(t, err)
			assert.Equal(t, 4, got) // 3 + 1 コピー

			owned, err := pcRepo.GetPlayerCards(context.Background(), "player-1")
			require.NoError(t, err)
			counts := make(map[string]int, len(owned))
			for _, pc := range owned {
				counts[pc.CardID] = pc.Count
			}
			assert.Equal(t, map[string]int{"TST-0001": 3, "TST-0002": 1}, counts)
		})

		t.Run("既に所持しているカードを含む pack を配布したとき、既存枚数に加算され今回配布したコピー総数を返す", func(t *testing.T) {
			packRepo := &fakeCardPackRepo{pack: &domain.CardPack{
				IsActive: true,
				Cards:    []domain.CardPackCard{{CardID: "TST-0001", Copies: 2}},
			}}
			pcRepo := newInMemoryPlayerCardRepo()
			pcRepo.Seed("player-1", []*domain.PlayerCard{
				{PlayerID: "player-1", CardID: "TST-0001", ArtNo: 0, Count: 5},
			})

			svc := NewGrantInteractor(packRepo, pcRepo)
			got, err := svc.GrantPack(context.Background(), "player-1", "any")

			require.NoError(t, err)
			assert.Equal(t, 2, got) // 今回配布分のみ

			owned, err := pcRepo.GetPlayerCards(context.Background(), "player-1")
			require.NoError(t, err)
			require.Len(t, owned, 1)
			assert.Equal(t, "TST-0001", owned[0].CardID)
			assert.Equal(t, 7, owned[0].Count) // 既存 5 + 配布 2
		})

		dbDown := errors.New("db down")
		errorCases := []struct {
			name     string
			packRepo *fakeCardPackRepo
			wantErr  error
		}{
			{
				name:     "is_active=false の pack のとき、ErrPackInactive になりカードを配布しない",
				packRepo: &fakeCardPackRepo{pack: &domain.CardPack{IsActive: false, Cards: []domain.CardPackCard{{CardID: "TST-0001", Copies: 3}}}},
				wantErr:  port.ErrPackInactive,
			},
			{
				name:     "pack が存在しないとき、ErrNotFound を伝播しカードを配布しない",
				packRepo: &fakeCardPackRepo{err: port.ErrNotFound},
				wantErr:  port.ErrNotFound,
			},
			{
				name:     "内包カードが 0 件のとき、ErrEmptyPack になりカードを配布しない",
				packRepo: &fakeCardPackRepo{pack: &domain.CardPack{IsActive: true, Cards: nil}},
				wantErr:  port.ErrEmptyPack,
			},
			{
				name:     "pack repo が任意エラーのとき、それを伝播しカードを配布しない",
				packRepo: &fakeCardPackRepo{err: dbDown},
				wantErr:  dbDown,
			},
		}

		for _, tc := range errorCases {
			t.Run(tc.name, func(t *testing.T) {
				pcRepo := newInMemoryPlayerCardRepo()
				svc := NewGrantInteractor(tc.packRepo, pcRepo)
				_, err := svc.GrantPack(context.Background(), "player-1", "any")

				require.ErrorIs(t, err, tc.wantErr)
				assertNoCardsGranted(t, pcRepo, "player-1")
			})
		}
	})
}
