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

// fakeGrantCardRepo は CardRepo の最小スタブ。
type fakeGrantCardRepo struct {
	gotFactions []string
	retIDs      []string
	retErr      error
}

func (f *fakeGrantCardRepo) FindAll(_ context.Context) ([]*domain.Card, error) { return nil, nil }
func (f *fakeGrantCardRepo) FindCardIDsByFactions(_ context.Context, factions []string) ([]string, error) {
	f.gotFactions = append([]string(nil), factions...)
	return f.retIDs, f.retErr
}

// fakeGrantPlayerCardRepo は PlayerCardRepo の最小スタブ。AddCards の引数を記録する。
type fakeGrantPlayerCardRepo struct {
	called          bool
	gotPlayerID     string
	gotCardIDs      []string
	gotCountPerCard int
	retAdded        int
}

func (f *fakeGrantPlayerCardRepo) GetPlayerCards(_ context.Context, _ string) ([]*domain.PlayerCard, error) {
	return nil, nil
}
func (f *fakeGrantPlayerCardRepo) AddCards(_ context.Context, playerID string, cardIDs []string, countPerCard int) (int, error) {
	f.called = true
	f.gotPlayerID = playerID
	f.gotCardIDs = append([]string(nil), cardIDs...)
	f.gotCountPerCard = countPerCard
	return f.retAdded, nil
}

// TestGrantPack_ByFactions は selection.by_factions の pack を配布したとき、
// pack に書かれた factions で解決された card_id 群と pack の copies_per_card が
// AddCards に渡ることを固定する。
func TestGrantPack_ByFactions(t *testing.T) {
	packRepo := &fakeCardPackRepo{pack: &domain.CardPack{
		Selection:     domain.SelectionByFactions{Factions: []string{"SHE", "Neutral"}},
		CopiesPerCard: 3,
		IsActive:      true,
	}}
	cardRepo := &fakeGrantCardRepo{retIDs: []string{"SH-0001", "NE-0001"}}
	pcRepo := &fakeGrantPlayerCardRepo{retAdded: 6}

	svc := NewGrantInteractor(packRepo, cardRepo, pcRepo)
	got, err := svc.GrantPack(context.Background(), "player-1", "any")

	require.NoError(t, err)
	assert.Equal(t, 6, got)
	assert.Equal(t, []string{"SHE", "Neutral"}, cardRepo.gotFactions)
	assert.Equal(t, "player-1", pcRepo.gotPlayerID)
	assert.Equal(t, []string{"SH-0001", "NE-0001"}, pcRepo.gotCardIDs)
	assert.Equal(t, 3, pcRepo.gotCountPerCard)
}

// TestGrantPack_ByCardIDs は selection.by_card_ids の pack を配布したとき、
// pack の card_ids がそのまま AddCards に渡り、CardRepo は呼ばれないことを固定する。
func TestGrantPack_ByCardIDs(t *testing.T) {
	packRepo := &fakeCardPackRepo{pack: &domain.CardPack{
		Selection:     domain.SelectionByCardIDs{CardIDs: []string{"LM-0001"}},
		CopiesPerCard: 1,
		IsActive:      true,
	}}
	cardRepo := &fakeGrantCardRepo{}
	pcRepo := &fakeGrantPlayerCardRepo{retAdded: 1}

	svc := NewGrantInteractor(packRepo, cardRepo, pcRepo)
	got, err := svc.GrantPack(context.Background(), "player-1", "any")

	require.NoError(t, err)
	assert.Equal(t, 1, got)
	assert.Nil(t, cardRepo.gotFactions, "by_card_ids 経路では cardRepo を引かない")
	assert.Equal(t, []string{"LM-0001"}, pcRepo.gotCardIDs)
	assert.Equal(t, 1, pcRepo.gotCountPerCard)
}

// TestGrantPack_InactivePack は is_active=false の pack に対して
// port.ErrPackInactive を返し、AddCards を呼ばないことを固定する。
func TestGrantPack_InactivePack(t *testing.T) {
	packRepo := &fakeCardPackRepo{pack: &domain.CardPack{
		Selection:     domain.SelectionByFactions{Factions: []string{"SHE"}},
		CopiesPerCard: 3,
		IsActive:      false,
	}}
	pcRepo := &fakeGrantPlayerCardRepo{}

	svc := NewGrantInteractor(packRepo, &fakeGrantCardRepo{}, pcRepo)
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

	svc := NewGrantInteractor(packRepo, &fakeGrantCardRepo{}, pcRepo)
	_, err := svc.GrantPack(context.Background(), "player-1", "any")

	require.Error(t, err)
	assert.ErrorIs(t, err, port.ErrNotFound)
	assert.False(t, pcRepo.called)
}

// TestGrantPack_CardRepoError は selection.by_factions 解決時の cardRepo エラーが
// 呼び出し側に伝播し、AddCards が呼ばれないことを固定する。
func TestGrantPack_CardRepoError(t *testing.T) {
	dbDown := errors.New("db down")
	packRepo := &fakeCardPackRepo{pack: &domain.CardPack{
		Selection:     domain.SelectionByFactions{Factions: []string{"SHE"}},
		CopiesPerCard: 3,
		IsActive:      true,
	}}
	cardRepo := &fakeGrantCardRepo{retErr: dbDown}
	pcRepo := &fakeGrantPlayerCardRepo{}

	svc := NewGrantInteractor(packRepo, cardRepo, pcRepo)
	_, err := svc.GrantPack(context.Background(), "player-1", "any")

	require.Error(t, err)
	assert.ErrorIs(t, err, dbDown)
	assert.False(t, pcRepo.called)
}

// TestGrantPack_EmptyResolvedSet は selection 解決の結果が 0 件の場合、
// port.ErrCardMasterEmpty を返し AddCards を呼ばないことを固定する。
func TestGrantPack_EmptyResolvedSet(t *testing.T) {
	packRepo := &fakeCardPackRepo{pack: &domain.CardPack{
		Selection:     domain.SelectionByFactions{Factions: []string{"SHE"}},
		CopiesPerCard: 3,
		IsActive:      true,
	}}
	cardRepo := &fakeGrantCardRepo{retIDs: nil}
	pcRepo := &fakeGrantPlayerCardRepo{}

	svc := NewGrantInteractor(packRepo, cardRepo, pcRepo)
	_, err := svc.GrantPack(context.Background(), "player-1", "any")

	require.Error(t, err)
	assert.ErrorIs(t, err, port.ErrCardMasterEmpty)
	assert.False(t, pcRepo.called)
}
