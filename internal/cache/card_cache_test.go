package cache

import (
	"context"
	"errors"
	"testing"

	gencache "github.com/kenyamaneko/overload-party-card/data/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	gamedesign "github.com/kenyamaneko/overload-party-common/packages/game-design-constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubCardRepo は port.CardRepo のテスト用スタブ。
type stubCardRepo struct {
	cards []*domain.Card
	err   error
}

func (r *stubCardRepo) FindAll(_ context.Context) ([]*domain.Card, error) {
	return r.cards, r.err
}

func isResourceType(cardType string) bool {
	return cardType == gamedesign.CardTypeCompute || cardType == gamedesign.CardTypeDataResource
}

func loadTestCache(t *testing.T) *CardCache {
	t.Helper()
	cc := NewCardCache()
	err := cc.LoadFromBytes(gencache.CardsJSON)
	require.NoError(t, err, "LoadFromBytes failed")
	return cc
}

func TestLoadFromBytes(t *testing.T) {
	t.Run("[カードキャッシュ]生成カードキャッシュのロード", func(t *testing.T) {
		t.Run("0件JSONのとき、マスター欠落としてエラーになる", func(t *testing.T) {
			cc := NewCardCache()
			err := cc.LoadFromBytes([]byte(`[]`))
			assert.Error(t, err)
		})

		t.Run("JSONとして不正なバイト列のとき、読み込みがエラーになる", func(t *testing.T) {
			cc := NewCardCache()
			err := cc.LoadFromBytes([]byte(`{`))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "parse card data")
		})

		t.Run("全カードでresource種別ならresource_labelがあり、support種別なら無い", func(t *testing.T) {
			// リソース種別か否かと label 有無を 1 枚ごとに等式で突き合わせ、if 分岐なしで網羅する。
			cc := loadTestCache(t)
			for cardID, card := range cc.All() {
				isResource := isResourceType(card.CardType)
				hasLabel := card.ResourceLabel != ""
				assert.Equalf(t, isResource, hasLabel,
					"card %s (type=%s, label=%q): resource types must have resource_label, support types must not",
					cardID, card.CardType, card.ResourceLabel)
			}
		})
	})
}

func TestLoad(t *testing.T) {
	t.Run("[カードキャッシュ]DBからのカードキャッシュ読み込み", func(t *testing.T) {
		t.Run("DBに定義があるとき、読み込んだ定義が検索で引けるようになる", func(t *testing.T) {
			repo := &stubCardRepo{cards: []*domain.Card{
				{CardID: "TST-0001", CardName: "Test Card"},
			}}
			cc := NewCardCache()

			err := cc.Load(context.Background(), repo)

			require.NoError(t, err)
			got := cc.Get("TST-0001")
			require.NotNil(t, got)
			assert.Equal(t, "Test Card", got.CardName)
		})

		t.Run("DBが0件のとき、マスター欠落としてエラーになる", func(t *testing.T) {
			repo := &stubCardRepo{cards: nil}
			cc := NewCardCache()

			err := cc.Load(context.Background(), repo)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "0 cards loaded")
		})

		t.Run("DB読み込みが失敗したとき、そのエラーが伝播する", func(t *testing.T) {
			dbErr := errors.New("db down")
			repo := &stubCardRepo{err: dbErr}
			cc := NewCardCache()

			err := cc.Load(context.Background(), repo)

			require.Error(t, err)
			assert.ErrorIs(t, err, dbErr)
		})
	})
}
