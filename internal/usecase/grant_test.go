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

// fakeGrantPlayerCardRepo は PlayerCardRepo の最小スタブ。AddCards の引数を記録する。
type fakeGrantPlayerCardRepo struct {
	called      bool
	gotPlayerID string
	gotCards    []domain.CardPackCard
	retAdded    int
}

func (f *fakeGrantPlayerCardRepo) GetPlayerCards(_ context.Context, _ string) ([]*domain.PlayerCard, error) {
	return nil, nil
}
func (f *fakeGrantPlayerCardRepo) AddCards(_ context.Context, playerID string, cards []domain.CardPackCard) (int, error) {
	f.called = true
	f.gotPlayerID = playerID
	f.gotCards = append([]domain.CardPackCard(nil), cards...)
	return f.retAdded, nil
}

func TestGrantPack(t *testing.T) {
	t.Run("パック付与", func(t *testing.T) {
		t.Run("有効な pack のとき、pack の全カードを AddCards に渡し付与枚数を返す", func(t *testing.T) {
			// pack 内でカードごとに枚数が異なるケースで、写し漏れを検出する。
			packRepo := &fakeCardPackRepo{pack: &domain.CardPack{
				IsActive: true,
				Cards: []domain.CardPackCard{
					{CardID: "SH-0001", Copies: 3},
					{CardID: "SH-0002", Copies: 1},
				},
			}}
			pcRepo := &fakeGrantPlayerCardRepo{retAdded: 4}

			svc := NewGrantInteractor(packRepo, pcRepo)
			got, err := svc.GrantPack(context.Background(), "player-1", "any")

			require.NoError(t, err)
			assert.Equal(t, 4, got)
			assert.Equal(t, "player-1", pcRepo.gotPlayerID)
			assert.Equal(t, []domain.CardPackCard{
				{CardID: "SH-0001", Copies: 3},
				{CardID: "SH-0002", Copies: 1},
			}, pcRepo.gotCards)
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
