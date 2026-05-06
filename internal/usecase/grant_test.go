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

// TestGrantPack_PassesPackCardsToAddCards は pack マスターから取得したカード集合
// (card_id と copies の組) がそのまま AddCards に渡ることを固定する。
// pack 内でカードごとに枚数が異なるケースで、写し漏れを検出する。
func TestGrantPack_PassesPackCardsToAddCards(t *testing.T) {
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
}

// TestGrantPack_InactivePack は is_active=false の pack に対して
// port.ErrPackInactive を返し、AddCards を呼ばないことを固定する。
func TestGrantPack_InactivePack(t *testing.T) {
	packRepo := &fakeCardPackRepo{pack: &domain.CardPack{
		IsActive: false,
		Cards:    []domain.CardPackCard{{CardID: "SH-0001", Copies: 3}},
	}}
	pcRepo := &fakeGrantPlayerCardRepo{}

	svc := NewGrantInteractor(packRepo, pcRepo)
	_, err := svc.GrantPack(context.Background(), "player-1", "any")

	require.Error(t, err)
	assert.ErrorIs(t, err, port.ErrPackInactive)
	assert.False(t, pcRepo.called)
}

// TestGrantPack_PackNotFound は pack repo の ErrNotFound が呼び出し側に伝播し、
// AddCards が呼ばれないことを固定する。
func TestGrantPack_PackNotFound(t *testing.T) {
	packRepo := &fakeCardPackRepo{err: port.ErrNotFound}
	pcRepo := &fakeGrantPlayerCardRepo{}

	svc := NewGrantInteractor(packRepo, pcRepo)
	_, err := svc.GrantPack(context.Background(), "player-1", "any")

	require.Error(t, err)
	assert.ErrorIs(t, err, port.ErrNotFound)
	assert.False(t, pcRepo.called)
}

// TestGrantPack_EmptyPack は内包カードが 0 件の pack に対して
// port.ErrEmptyPack を返し、AddCards を呼ばないことを固定する。
func TestGrantPack_EmptyPack(t *testing.T) {
	packRepo := &fakeCardPackRepo{pack: &domain.CardPack{
		IsActive: true,
		Cards:    nil,
	}}
	pcRepo := &fakeGrantPlayerCardRepo{}

	svc := NewGrantInteractor(packRepo, pcRepo)
	_, err := svc.GrantPack(context.Background(), "player-1", "any")

	require.Error(t, err)
	assert.ErrorIs(t, err, port.ErrEmptyPack)
	assert.False(t, pcRepo.called)
}

// TestGrantPack_PackRepoError は pack repo の任意エラーが伝播し、
// AddCards が呼ばれないことを固定する。
func TestGrantPack_PackRepoError(t *testing.T) {
	dbDown := errors.New("db down")
	packRepo := &fakeCardPackRepo{err: dbDown}
	pcRepo := &fakeGrantPlayerCardRepo{}

	svc := NewGrantInteractor(packRepo, pcRepo)
	_, err := svc.GrantPack(context.Background(), "player-1", "any")

	require.Error(t, err)
	assert.ErrorIs(t, err, dbDown)
	assert.False(t, pcRepo.called)
}
