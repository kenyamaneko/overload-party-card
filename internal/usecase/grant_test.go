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

// fakeGrantPlayerCardRepo は PlayerCardRepo の最小スタブ。guard 系ケースで
// AddCards が呼ばれないことだけを記録する。
type fakeGrantPlayerCardRepo struct {
	called bool
}

func (f *fakeGrantPlayerCardRepo) GetPlayerCards(_ context.Context, _ string) ([]*domain.PlayerCard, error) {
	return nil, nil
}
func (f *fakeGrantPlayerCardRepo) AddCards(_ context.Context, _ string, _ []domain.CardPackCard) (int, error) {
	f.called = true
	return 0, nil
}

func TestGrantPack(t *testing.T) {
	t.Run("パック付与", func(t *testing.T) {
		t.Run("有効な pack のとき、pack の全カードがプレイヤーの所持へ加算され、付与枚数は copies 合計になる", func(t *testing.T) {
			// pack 内でカードごとに枚数が異なるケースで、写し漏れを検出する。
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

		t.Run("既に所持しているカードを含む pack を配布するとき、既存枚数へ加算される", func(t *testing.T) {
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

		// いずれのエラーでも配布は起きない (AddCards は呼ばれない)。
		dbDown := errors.New("db down")
		errorCases := []struct {
			name     string
			packRepo *fakeCardPackRepo
			wantErr  error
		}{
			{
				name:     "is_active=false の pack のとき、ErrPackInactive になり AddCards を呼ばない",
				packRepo: &fakeCardPackRepo{pack: &domain.CardPack{IsActive: false, Cards: []domain.CardPackCard{{CardID: "SH-0001", Copies: 3}}}},
				wantErr:  port.ErrPackInactive,
			},
			{
				name:     "pack が存在しないとき、ErrNotFound を伝播し AddCards を呼ばない",
				packRepo: &fakeCardPackRepo{err: port.ErrNotFound},
				wantErr:  port.ErrNotFound,
			},
			{
				name:     "内包カードが 0 件のとき、ErrEmptyPack になり AddCards を呼ばない",
				packRepo: &fakeCardPackRepo{pack: &domain.CardPack{IsActive: true, Cards: nil}},
				wantErr:  port.ErrEmptyPack,
			},
			{
				name:     "pack repo が任意エラーのとき、それを伝播し AddCards を呼ばない",
				packRepo: &fakeCardPackRepo{err: dbDown},
				wantErr:  dbDown,
			},
		}

		for _, tc := range errorCases {
			t.Run(tc.name, func(t *testing.T) {
				pcRepo := &fakeGrantPlayerCardRepo{}
				svc := NewGrantInteractor(tc.packRepo, pcRepo)
				_, err := svc.GrantPack(context.Background(), "player-1", "any")

				require.ErrorIs(t, err, tc.wantErr)
				assert.False(t, pcRepo.called)
			})
		}
	})
}
