package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gamedesign "github.com/kenyamaneko/overload-party-common/packages/game-design-constants"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// errGrantTestDBDown は「card repo エラーは上位に伝播する」ケースで
// 注入＆期待値として使う sentinel エラー。
// errors.Is で同一性を確認するためにテーブル行と注入先で共有する。
var errGrantTestDBDown = errors.New("db down")

// fakeGrantCardRepo は FindCardIDsByFactions の呼び出し引数と返却値/エラーを
// 制御するテスト用スタブです。
// 呼び出しで渡された factions を記録し「どのファクションを対象にカード ID を
// 引いたか」の検証に使います。
type fakeGrantCardRepo struct {
	gotFactions []string
	retIDs      []string
	retErr      error
}

func (f *fakeGrantCardRepo) FindAll(_ context.Context) ([]*domain.Card, error) {
	return nil, nil
}

func (f *fakeGrantCardRepo) FindCardIDsByFactions(_ context.Context, factions []string) ([]string, error) {
	f.gotFactions = append([]string(nil), factions...)
	return f.retIDs, f.retErr
}

// fakeGrantPlayerCardRepo は AddCards の呼び出し引数と返却値/エラーを制御する
// テスト用スタブです。
// 付与対象 (playerID, cardIDs, countPerCard) を記録します。
type fakeGrantPlayerCardRepo struct {
	gotPlayerID     string
	gotCardIDs      []string
	gotCountPerCard int
	retAdded        int
	retErr          error
	called          bool
}

func (f *fakeGrantPlayerCardRepo) GetPlayerCards(_ context.Context, _ string) ([]*domain.PlayerCard, error) {
	return nil, nil
}

func (f *fakeGrantPlayerCardRepo) AddCards(_ context.Context, playerID string, cardIDs []string, countPerCard int) (int, error) {
	f.called = true
	f.gotPlayerID = playerID
	f.gotCardIDs = append([]string(nil), cardIDs...)
	f.gotCountPerCard = countPerCard
	return f.retAdded, f.retErr
}

// grantCase は Grant 系テストのテーブル行。
// cardRepoErr を nil/non-nil で切り分けることで「正常系 / repo 伝播エラー」
// 「バリデーションエラー」の 3 種のケースをテーブル行だけで表現する。
type grantCase struct {
	name         string
	faction      string
	repoCardIDs  []string
	cardRepoErr  error
	addedReturn  int
	wantFactions []string // 空ならバリデーションで repo 未呼び出し
	wantCardIDs  []string
	wantCopies   int
	wantErrIs    error // nil なら成功ケース
}

// TestGrantInitialPack は「初期配布は選択ファクション + Neutral」
// という配布仕様を検証する。
// faction pack との責務分離（Neutral 同梱の有無）が仕様上の差分なので
// FindCardIDsByFactions に渡された factions を仕様として assert する。
func TestGrantInitialPack(t *testing.T) {
	tests := []grantCase{
		{
			name:         "SHE 選択時は SHE と Neutral の card_id を引いて配布する",
			faction:      gamedesign.FactionSHE,
			repoCardIDs:  []string{"SH-0001", "NE-0001"},
			addedReturn:  6,
			wantFactions: []string{gamedesign.FactionSHE, gamedesign.FactionNeutral},
			wantCardIDs:  []string{"SH-0001", "NE-0001"},
			wantCopies:   6,
		},
		{
			name:         "Tenki 選択時は Tenki と Neutral の card_id を引いて配布する",
			faction:      gamedesign.FactionTenki,
			repoCardIDs:  []string{"TE-0001", "NE-0001"},
			addedReturn:  6,
			wantFactions: []string{gamedesign.FactionTenki, gamedesign.FactionNeutral},
			wantCardIDs:  []string{"TE-0001", "NE-0001"},
			wantCopies:   6,
		},
		{
			name:      "選択不可ファクションは ErrInvalidArgument を返し repo を呼ばない",
			faction:   gamedesign.FactionNeutral,
			wantErrIs: port.ErrInvalidArgument,
		},
		{
			name:        "card repo エラーは上位に伝播する",
			faction:     gamedesign.FactionSHE,
			cardRepoErr: errGrantTestDBDown,
			wantErrIs:   errGrantTestDBDown,
		},
		{
			name:        "active カード 0 件なら ErrCardMasterEmpty を返し AddCards を呼ばない",
			faction:     gamedesign.FactionSHE,
			repoCardIDs: nil,
			wantErrIs:   port.ErrCardMasterEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runGrantCase(t, tt, "player-1", grantInitialPackInvoker)
		})
	}
}

// TestGrantFactionPack は「faction 購入配布は選択ファクションのみ
// (Neutral 無し)」という配布仕様を検証する。
// initial pack との唯一の責務差分が「Neutral を含めないこと」なので
// FindCardIDsByFactions に渡された factions にその仕様を固定する。
func TestGrantFactionPack(t *testing.T) {
	tests := []grantCase{
		{
			name:         "SHE 購入時は SHE のみの card_id を引いて配布する (Neutral 無し)",
			faction:      gamedesign.FactionSHE,
			repoCardIDs:  []string{"SH-0001", "SH-0002"},
			addedReturn:  6,
			wantFactions: []string{gamedesign.FactionSHE},
			wantCardIDs:  []string{"SH-0001", "SH-0002"},
			wantCopies:   6,
		},
		{
			name:         "Sugar 購入時は Sugar のみの card_id を引いて配布する (Neutral 無し)",
			faction:      gamedesign.FactionSugar,
			repoCardIDs:  []string{"SU-0001"},
			addedReturn:  3,
			wantFactions: []string{gamedesign.FactionSugar},
			wantCardIDs:  []string{"SU-0001"},
			wantCopies:   3,
		},
		{
			name:      "選択不可ファクションは ErrInvalidArgument を返し repo を呼ばない",
			faction:   gamedesign.FactionNeutral,
			wantErrIs: port.ErrInvalidArgument,
		},
		{
			name:        "card repo エラーは上位に伝播する",
			faction:     gamedesign.FactionSHE,
			cardRepoErr: errGrantTestDBDown,
			wantErrIs:   errGrantTestDBDown,
		},
		{
			name:        "active カード 0 件なら ErrCardMasterEmpty を返し AddCards を呼ばない",
			faction:     gamedesign.FactionSHE,
			repoCardIDs: nil,
			wantErrIs:   port.ErrCardMasterEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runGrantCase(t, tt, "player-2", grantFactionPackInvoker)
		})
	}
}

// grantInvoker は initial / faction それぞれの GrantInteractor メソッドを
// 同じテストランナーから呼び出すための薄いアダプタ。
type grantInvoker func(svc *GrantInteractor, ctx context.Context, playerID, faction string) (int, error)

func grantInitialPackInvoker(svc *GrantInteractor, ctx context.Context, playerID, faction string) (int, error) {
	return svc.GrantInitialPack(ctx, playerID, faction)
}

func grantFactionPackInvoker(svc *GrantInteractor, ctx context.Context, playerID, faction string) (int, error) {
	return svc.GrantFactionPack(ctx, playerID, faction)
}

// runGrantCase は grantCase を実行し、成功/失敗の両パターンを検証する。
// 成功ケース (wantErrIs==nil) と失敗ケース (wantErrIs!=nil) は
// map + switch による分岐で表現し、ケース内の if 分岐を避ける。
func runGrantCase(t *testing.T, tt grantCase, playerID string, invoke grantInvoker) {
	t.Helper()
	cardRepo := &fakeGrantCardRepo{retIDs: tt.repoCardIDs, retErr: tt.cardRepoErr}
	pcRepo := &fakeGrantPlayerCardRepo{retAdded: tt.addedReturn}
	svc := NewGrantInteractor(cardRepo, pcRepo)

	got, err := invoke(svc, context.Background(), playerID, tt.faction)

	verifiers := map[bool]func(){
		true: func() {
			require.NoError(t, err)
			assert.Equal(t, tt.wantCopies, got)
			assert.Equal(t, tt.wantFactions, cardRepo.gotFactions)
			assert.True(t, pcRepo.called, "AddCards must be invoked on success")
			assert.Equal(t, playerID, pcRepo.gotPlayerID)
			assert.Equal(t, tt.wantCardIDs, pcRepo.gotCardIDs)
			assert.Equal(t, copiesPerGrant, pcRepo.gotCountPerCard)
		},
		false: func() {
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErrIs)
			assert.Equal(t, 0, got)
			assert.False(t, pcRepo.called, "AddCards must NOT be invoked on error")
		},
	}
	verifiers[tt.wantErrIs == nil]()
}
