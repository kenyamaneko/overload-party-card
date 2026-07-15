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

func TestGrantPack(t *testing.T) {
	t.Run("パック付与", func(t *testing.T) {
		t.Run("カードパックを配ると、含まれる各カードが枚数分プレイヤーに付与され、付与数は合計枚数になる", func(t *testing.T) {
			packRepo := &fakeCardPackRepo{pack: &domain.CardPack{
				IsActive: true,
				Cards: []domain.CardPackCard{
					{CardID: "SH-0001", Copies: 3},
					{CardID: "SH-0002", Copies: 1},
				},
			}}
			pcRepo := newInMemoryPlayerCardRepo()

			svc := NewGrantInteractor(packRepo, pcRepo)
			got, err := svc.GrantPack(context.Background(), "player-1", "any")

			require.NoError(t, err)
			assert.Equal(t, 4, got)

			playerCards, err := pcRepo.GetPlayerCards(context.Background(), "player-1")
			require.NoError(t, err)
			assert.ElementsMatch(t, []*domain.PlayerCard{
				{PlayerID: "player-1", CardID: "SH-0001", Count: 3},
				{PlayerID: "player-1", CardID: "SH-0002", Count: 1},
			}, playerCards)
		})

		t.Run("既に所持しているカードを配ると、所持枚数に加算される", func(t *testing.T) {
			packRepo := &fakeCardPackRepo{pack: &domain.CardPack{
				IsActive: true,
				Cards: []domain.CardPackCard{
					{CardID: "SH-0001", Copies: 2},
				},
			}}
			pcRepo := newInMemoryPlayerCardRepo()
			pcRepo.Seed("player-1", []*domain.PlayerCard{{PlayerID: "player-1", CardID: "SH-0001", Count: 5}})

			svc := NewGrantInteractor(packRepo, pcRepo)
			got, err := svc.GrantPack(context.Background(), "player-1", "any")

			require.NoError(t, err)
			assert.Equal(t, 2, got)

			playerCards, err := pcRepo.GetPlayerCards(context.Background(), "player-1")
			require.NoError(t, err)
			assert.ElementsMatch(t, []*domain.PlayerCard{
				{PlayerID: "player-1", CardID: "SH-0001", Count: 7},
			}, playerCards)
		})

		dbDown := errors.New("db down")
		errorCases := []struct {
			name     string
			packRepo *fakeCardPackRepo
			wantErr  error
		}{
			{
				name:     "無効化されたカードパックのとき、ErrPackInactive になり何も付与されない",
				packRepo: &fakeCardPackRepo{pack: &domain.CardPack{IsActive: false, Cards: []domain.CardPackCard{{CardID: "SH-0001", Copies: 3}}}},
				wantErr:  port.ErrPackInactive,
			},
			{
				name:     "存在しないカードパックのとき、ErrNotFound になり何も付与されない",
				packRepo: &fakeCardPackRepo{err: port.ErrNotFound},
				wantErr:  port.ErrNotFound,
			},
			{
				name:     "中身が空のカードパックのとき、ErrEmptyPack になり何も付与されない",
				packRepo: &fakeCardPackRepo{pack: &domain.CardPack{IsActive: true, Cards: nil}},
				wantErr:  port.ErrEmptyPack,
			},
			{
				name:     "カードパックの取得が失敗したとき、そのエラーになり何も付与されない",
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
				playerCards, err := pcRepo.GetPlayerCards(context.Background(), "player-1")
				require.NoError(t, err)
				assert.Empty(t, playerCards)
			})
		}
	})
}
