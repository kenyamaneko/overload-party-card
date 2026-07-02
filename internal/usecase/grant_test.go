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

// TestGrantPack_PersistsPackCardsAndReturnsCopyTotal は、pack マスターのカードが
// プレイヤーへ配布されて所持枚数が pack 定義どおりに永続化されること・戻り値が
// 配布コピー総数であることを検証します。カードごとに枚数が異なるケースで写し漏れを
// 検出します。
func TestGrantPack_PersistsPackCardsAndReturnsCopyTotal(t *testing.T) {
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
}

// TestGrantPack_AddsToExistingOwnedCards は、既に所持しているカードへ pack を
// 配布したとき所持枚数が加算されること・戻り値は今回配布したコピー総数のみである
// ことを検証します。
func TestGrantPack_AddsToExistingOwnedCards(t *testing.T) {
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
}

// TestGrantPack_InactivePack は is_active=false の pack に対して
// port.ErrPackInactive を返し、カードを配布しないことを検証します。
func TestGrantPack_InactivePack(t *testing.T) {
	packRepo := &fakeCardPackRepo{pack: &domain.CardPack{
		IsActive: false,
		Cards:    []domain.CardPackCard{{CardID: "TST-0001", Copies: 3}},
	}}
	pcRepo := newInMemoryPlayerCardRepo()

	svc := NewGrantInteractor(packRepo, pcRepo)
	_, err := svc.GrantPack(context.Background(), "player-1", "any")

	require.Error(t, err)
	assert.ErrorIs(t, err, port.ErrPackInactive)
	assertNoCardsGranted(t, pcRepo, "player-1")
}

// TestGrantPack_PackNotFound は pack repo の ErrNotFound が呼び出し側に伝播し、
// カードを配布しないことを検証します。
func TestGrantPack_PackNotFound(t *testing.T) {
	packRepo := &fakeCardPackRepo{err: port.ErrNotFound}
	pcRepo := newInMemoryPlayerCardRepo()

	svc := NewGrantInteractor(packRepo, pcRepo)
	_, err := svc.GrantPack(context.Background(), "player-1", "any")

	require.Error(t, err)
	assert.ErrorIs(t, err, port.ErrNotFound)
	assertNoCardsGranted(t, pcRepo, "player-1")
}

// TestGrantPack_EmptyPack は内包カードが 0 件の pack に対して
// port.ErrEmptyPack を返し、カードを配布しないことを検証します。
func TestGrantPack_EmptyPack(t *testing.T) {
	packRepo := &fakeCardPackRepo{pack: &domain.CardPack{
		IsActive: true,
		Cards:    nil,
	}}
	pcRepo := newInMemoryPlayerCardRepo()

	svc := NewGrantInteractor(packRepo, pcRepo)
	_, err := svc.GrantPack(context.Background(), "player-1", "any")

	require.Error(t, err)
	assert.ErrorIs(t, err, port.ErrEmptyPack)
	assertNoCardsGranted(t, pcRepo, "player-1")
}

// TestGrantPack_PackRepoError は pack repo の任意エラーが伝播し、カードを
// 配布しないことを検証します。
func TestGrantPack_PackRepoError(t *testing.T) {
	dbDown := errors.New("db down")
	packRepo := &fakeCardPackRepo{err: dbDown}
	pcRepo := newInMemoryPlayerCardRepo()

	svc := NewGrantInteractor(packRepo, pcRepo)
	_, err := svc.GrantPack(context.Background(), "player-1", "any")

	require.Error(t, err)
	assert.ErrorIs(t, err, dbDown)
	assertNoCardsGranted(t, pcRepo, "player-1")
}
