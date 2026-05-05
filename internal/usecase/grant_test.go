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

// fakeCardPackRepo は CardPackRepo のテスト用スタブです。
type fakeCardPackRepo struct {
	pack       *domain.CardPack
	err        error
	gotPackIDs []string
}

func (f *fakeCardPackRepo) GetPack(_ context.Context, packID string) (*domain.CardPack, error) {
	f.gotPackIDs = append(f.gotPackIDs, packID)
	if f.err != nil {
		return nil, f.err
	}
	return f.pack, nil
}

// fakeGrantCardRepo は CardRepo の最小スタブです。FindCardIDsByFactions の引数記録 +
// 返却値制御で「pack の selection.factions が repo に正しく伝播するか」を検証します。
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

// fakeGrantPlayerCardRepo は PlayerCardRepo の最小スタブ。AddCards の引数を記録します。
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

// errPackTestRepoDown は repo 失敗ケースの注入 / 期待値共有用 sentinel。
var errPackTestRepoDown = errors.New("db down")

// grantCase は GrantPack のテーブル駆動テストケース。
//
// pack 取得 → selection 解決 → AddCards 配布 という 3 段の各分岐 (success / pack not found
// / inactive / invalid selection / 配布対象 0 件) を、ケースの差し替えだけで網羅する。
// テストコード内に if 文を書かないため、成功/失敗の分岐は wantErrIs の有無で表現する。
type grantCase struct {
	name string
	// pack repo 入力
	pack       *domain.CardPack
	packRepoEr error
	// card repo 入力 (selection が by_factions の場合に評価)
	cardIDs    []string
	cardRepoEr error
	// addCards 戻り値
	addedReturn int
	// 期待値
	wantPackIDQueried []string
	wantFactionsQuery []string
	wantAddCalled     bool
	wantCardIDsAdded  []string
	wantCountPerCard  int
	wantCopies        int
	wantErrIs         error
}

const (
	testPlayerID  = "player-1"
	testPackID    = "initial_she"
	testCopies    = 3
	testFaction   = "SHE"
	testNeutral   = "Neutral"
	testCardIDSH1 = "SH-0001"
	testCardIDNE1 = "NE-0001"
	testCardIDLM1 = "LM-0001"
)

func packByFactions(factions []string, copies int, active bool) *domain.CardPack {
	return &domain.CardPack{
		PackID:        testPackID,
		Selection:     domain.SelectionByFactions{Factions: factions},
		CopiesPerCard: copies,
		IsActive:      active,
	}
}

func packByCardIDs(cardIDs []string, copies int, active bool) *domain.CardPack {
	return &domain.CardPack{
		PackID:        "limited_test",
		Selection:     domain.SelectionByCardIDs{CardIDs: cardIDs},
		CopiesPerCard: copies,
		IsActive:      active,
	}
}

// TestGrantPack は「pack マスターから対象を解決して配布する」という配布仕様を、
// pack の selection / is_active / repo 失敗 / 配布対象 0 件 の各観点で検証する。
//
// テストの仕様意図:
//   - selection.by_factions: pack に書かれた factions が cardRepo に渡り、得た card_id を AddCards へ
//   - selection.by_card_ids: pack の card_ids がそのまま AddCards へ
//   - inactive pack: AddCards は呼ばれず ErrPackInactive
//   - pack not found: AddCards は呼ばれず ErrNotFound 伝播
//   - card repo 失敗: AddCards は呼ばれずエラー伝播
//   - 配布対象 0 件: AddCards は呼ばれず ErrCardMasterEmpty
func TestGrantPack(t *testing.T) {
	tests := []grantCase{
		{
			name:              "by_factions: pack の factions で card_id を引き、copies_per_card 枚配布する",
			pack:              packByFactions([]string{testFaction, testNeutral}, testCopies, true),
			cardIDs:           []string{testCardIDSH1, testCardIDNE1},
			addedReturn:       6,
			wantPackIDQueried: []string{testPackID},
			wantFactionsQuery: []string{testFaction, testNeutral},
			wantAddCalled:     true,
			wantCardIDsAdded:  []string{testCardIDSH1, testCardIDNE1},
			wantCountPerCard:  testCopies,
			wantCopies:        6,
		},
		{
			name:              "by_card_ids: pack の card_ids をそのまま使い、cardRepo は呼ばれない",
			pack:              packByCardIDs([]string{testCardIDLM1}, 1, true),
			addedReturn:       1,
			wantPackIDQueried: []string{testPackID},
			wantFactionsQuery: nil,
			wantAddCalled:     true,
			wantCardIDsAdded:  []string{testCardIDLM1},
			wantCountPerCard:  1,
			wantCopies:        1,
		},
		{
			name:              "inactive pack は ErrPackInactive を返し AddCards を呼ばない",
			pack:              packByFactions([]string{testFaction}, testCopies, false),
			wantPackIDQueried: []string{testPackID},
			wantErrIs:         port.ErrPackInactive,
		},
		{
			name:              "pack 不在 (port.ErrNotFound) は伝播する",
			packRepoEr:        port.ErrNotFound,
			wantPackIDQueried: []string{testPackID},
			wantErrIs:         port.ErrNotFound,
		},
		{
			name:              "card repo エラーは伝播する (selection.by_factions 経路)",
			pack:              packByFactions([]string{testFaction}, testCopies, true),
			cardRepoEr:        errPackTestRepoDown,
			wantPackIDQueried: []string{testPackID},
			wantFactionsQuery: []string{testFaction},
			wantErrIs:         errPackTestRepoDown,
		},
		{
			name:              "active カード 0 件は ErrCardMasterEmpty を返し AddCards を呼ばない",
			pack:              packByFactions([]string{testFaction}, testCopies, true),
			cardIDs:           nil,
			wantPackIDQueried: []string{testPackID},
			wantFactionsQuery: []string{testFaction},
			wantErrIs:         port.ErrCardMasterEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packRepo := &fakeCardPackRepo{pack: tt.pack, err: tt.packRepoEr}
			cardRepo := &fakeGrantCardRepo{retIDs: tt.cardIDs, retErr: tt.cardRepoEr}
			pcRepo := &fakeGrantPlayerCardRepo{retAdded: tt.addedReturn}

			svc := NewGrantInteractor(packRepo, cardRepo, pcRepo)
			got, err := svc.GrantPack(context.Background(), testPlayerID, testPackID)

			assert.Equal(t, tt.wantPackIDQueried, packRepo.gotPackIDs, "GetPack 呼び出し履歴")
			assert.Equal(t, tt.wantFactionsQuery, cardRepo.gotFactions, "FindCardIDsByFactions に渡る factions")

			verifiers := map[bool]func(){
				true: func() {
					require.NoError(t, err)
					assert.Equal(t, tt.wantCopies, got)
					assert.True(t, pcRepo.called, "成功時は AddCards が呼ばれる")
					assert.Equal(t, testPlayerID, pcRepo.gotPlayerID)
					assert.Equal(t, tt.wantCardIDsAdded, pcRepo.gotCardIDs)
					assert.Equal(t, tt.wantCountPerCard, pcRepo.gotCountPerCard)
				},
				false: func() {
					require.Error(t, err)
					assert.ErrorIs(t, err, tt.wantErrIs)
					assert.Equal(t, 0, got)
					assert.Equal(t, tt.wantAddCalled, pcRepo.called, "失敗時は AddCards を呼ばない")
				},
			}
			verifiers[tt.wantErrIs == nil]()
		})
	}
}

// 注意: domain.Selection は private marker (isSelection) で外部実装をブロックするため、
// 「未知 Selection 実装混入時に ErrInvalidPackSelection を返す」防御 (resolveSelection
// の default case) は **compile time で到達不能**。型システムが invariant を担保しており
// 同パッケージ内に偽 Selection を作ることもできない。defensive default case は将来
// domain パッケージ内で sum を増やしたとき grant.go の switch を更新し忘れる事故への
// 保険として残す。
