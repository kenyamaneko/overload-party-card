package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubInitiativeRepo は port.InitiativeRepo のテスト用スタブ。
type stubInitiativeRepo struct {
	initiatives []*domain.Initiative
	err         error
}

func (r *stubInitiativeRepo) FindAll(_ context.Context) ([]*domain.Initiative, error) {
	return r.initiatives, r.err
}

// testInitiativesFixtureJSON は ID 検索の期待値を固定するための制御フィクスチャ。
// 生成データの並びに依存しないよう、既知の initiative_id を持つ複数件を用意する。
const testInitiativesFixtureJSON = `[
  {"initiative_id":"IN-TST-0001","product_id":"PD-TST-0001","kind":"routine","name":"テスト施策1","insight_cost":100,"effect_text":"","effect":{"ops":[]},"is_active":true},
  {"initiative_id":"IN-TST-0002","product_id":"PD-TST-0002","kind":"special","name":"テスト施策2","insight_cost":200,"effect_text":"","effect":{"ops":[]},"is_active":true}
]`

func TestInitiativeFindByID(t *testing.T) {
	t.Run("施策のID検索", func(t *testing.T) {
		ic := NewInitiativeCache()
		require.NoError(t, ic.LoadFromBytes([]byte(testInitiativesFixtureJSON)))

		t.Run("既知のIDのとき、該当する施策が返る", func(t *testing.T) {
			got := ic.FindByID("IN-TST-0001")
			require.NotNil(t, got)
			assert.Equal(t, "IN-TST-0001", got.InitiativeID)
			assert.Equal(t, "routine", got.Kind)
			assert.Equal(t, "テスト施策1", got.Name)
		})

		t.Run("未知のIDのとき、nilを返す", func(t *testing.T) {
			assert.Nil(t, ic.FindByID("IN-NOPE"))
		})
	})
}

func TestInitiativeLoadFromBytes(t *testing.T) {
	t.Run("施策キャッシュのロード", func(t *testing.T) {
		t.Run("0件JSONのとき、マスター欠落としてエラーになる", func(t *testing.T) {
			ic := NewInitiativeCache()
			err := ic.LoadFromBytes([]byte(`[]`))
			assert.Error(t, err)
		})

		t.Run("JSONとして不正なバイト列のとき、読み込みがエラーになる", func(t *testing.T) {
			ic := NewInitiativeCache()
			err := ic.LoadFromBytes([]byte(`{`))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "parse initiative data")
		})
	})
}

func TestInitiativeLoad(t *testing.T) {
	t.Run("DBからの施策キャッシュ読み込み", func(t *testing.T) {
		t.Run("DBに定義があるとき、読み込んだ定義が検索で引けるようになる", func(t *testing.T) {
			repo := &stubInitiativeRepo{initiatives: []*domain.Initiative{
				{InitiativeID: "TST-0001", ProductID: "PD-TST", Kind: "routine", Name: "テスト施策"},
			}}
			ic := NewInitiativeCache()

			err := ic.Load(context.Background(), repo)

			require.NoError(t, err)
			got := ic.FindByID("TST-0001")
			require.NotNil(t, got)
			assert.Equal(t, "テスト施策", got.Name)
		})

		t.Run("DBが0件のとき、マスター欠落としてエラーになる", func(t *testing.T) {
			repo := &stubInitiativeRepo{initiatives: nil}
			ic := NewInitiativeCache()

			err := ic.Load(context.Background(), repo)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "0 initiatives loaded")
		})

		t.Run("DB読み込みが失敗したとき、そのエラーが伝播する", func(t *testing.T) {
			dbErr := errors.New("db down")
			repo := &stubInitiativeRepo{err: dbErr}
			ic := NewInitiativeCache()

			err := ic.Load(context.Background(), repo)

			require.Error(t, err)
			assert.ErrorIs(t, err, dbErr)
		})
	})
}
