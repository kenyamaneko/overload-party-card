package cache_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

type fakeCardRepo struct {
	cards []*domain.Card
	err   error
}

func (f *fakeCardRepo) FindAll(ctx context.Context) ([]*domain.Card, error) {
	return f.cards, f.err
}

func TestCardCacheGet(t *testing.T) {
	t.Run("[cache] カード定義の取得", func(t *testing.T) {
		t.Run("一度もカード定義を読み込んでいない状態でcard_id TST-0001を指定して取得すると、見つからない", func(t *testing.T) {
			c := cache.NewCardCache()

			got := c.Get("TST-0001")

			assert.Nil(t, got)
		})

		t.Run("複数件のカード定義を読み込んだあと、読み込み済みのcard_id TST-0001を指定して取得すると、そのcard_idに対応するカード定義が返る", func(t *testing.T) {
			c := cache.NewCardCache()
			repo := &fakeCardRepo{cards: []*domain.Card{
				{CardID: "TST-0001"},
				{CardID: "TST-0002"},
			}}
			require.NoError(t, c.Load(context.Background(), repo))

			got := c.Get("TST-0001")

			require.NotNil(t, got)
			assert.Equal(t, "TST-0001", got.CardID)
		})

		t.Run("複数件のカード定義を読み込んだあと、読み込んでいないcard_id TST-9999を指定して取得すると、見つからない", func(t *testing.T) {
			c := cache.NewCardCache()
			repo := &fakeCardRepo{cards: []*domain.Card{
				{CardID: "TST-0001"},
				{CardID: "TST-0002"},
			}}
			require.NoError(t, c.Load(context.Background(), repo))

			got := c.Get("TST-9999")

			assert.Nil(t, got)
		})
	})
}

func TestCardCacheLoad(t *testing.T) {
	t.Run("[cache] カード定義の読み込み", func(t *testing.T) {
		cases := []struct {
			name           string
			repo           *fakeCardRepo
			wantErrContain string
		}{
			{
				name:           "カード定義の取得元が0件を返したとき、読み込みはエラーになる",
				repo:           &fakeCardRepo{cards: nil},
				wantErrContain: "0 cards loaded",
			},
			{
				name:           "カード定義の取得元がエラーを返したとき、読み込みはエラーになる",
				repo:           &fakeCardRepo{err: errors.New("fetch failed")},
				wantErrContain: "load card cache",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				c := cache.NewCardCache()

				err := c.Load(context.Background(), tc.repo)

				assert.ErrorContains(t, err, tc.wantErrContain)
			})
		}

		t.Run("読み込みを複数回行ったとき、最新の読み込みで得られたカード定義だけが取得でき、前回の読み込みにのみ存在したcard_idは取得できない", func(t *testing.T) {
			c := cache.NewCardCache()
			require.NoError(t, c.Load(context.Background(), &fakeCardRepo{cards: []*domain.Card{{CardID: "TST-0001"}}}))
			require.NoError(t, c.Load(context.Background(), &fakeCardRepo{cards: []*domain.Card{{CardID: "TST-0002"}}}))

			assert.NotNil(t, c.Get("TST-0002"))
			assert.Nil(t, c.Get("TST-0001"))
		})
	})
}

func TestCardCacheCount(t *testing.T) {
	t.Run("[cache] カード定義の件数", func(t *testing.T) {
		t.Run("読み込み前の件数は0になる", func(t *testing.T) {
			c := cache.NewCardCache()

			assert.Equal(t, 0, c.Count())
		})

		t.Run("カード定義を読み込んだあと、件数は読み込んだ件数になる", func(t *testing.T) {
			c := cache.NewCardCache()
			repo := &fakeCardRepo{cards: []*domain.Card{
				{CardID: "TST-0001"},
				{CardID: "TST-0002"},
				{CardID: "TST-0003"},
			}}
			require.NoError(t, c.Load(context.Background(), repo))

			assert.Equal(t, 3, c.Count())
		})
	})
}

func TestCardCacheAll(t *testing.T) {
	t.Run("[cache] カード定義の全件取得", func(t *testing.T) {
		t.Run("カード定義を読み込んだあと、全件取得すると読み込んだ全件を含む", func(t *testing.T) {
			c := cache.NewCardCache()
			repo := &fakeCardRepo{cards: []*domain.Card{
				{CardID: "TST-0001"},
				{CardID: "TST-0002"},
			}}
			require.NoError(t, c.Load(context.Background(), repo))

			got := c.All()

			assert.Len(t, got, 2)
			assert.Contains(t, got, "TST-0001")
			assert.Contains(t, got, "TST-0002")
		})
	})
}

func TestCardCacheLoadFromBytes(t *testing.T) {
	t.Run("[cache] JSONからのカード定義読み込み", func(t *testing.T) {
		t.Run("カード定義1件以上を含むJSONデータから読み込むと、そのカード定義が取得できるようになる", func(t *testing.T) {
			c := cache.NewCardCache()

			err := c.LoadFromBytes([]byte(`[{"card_id":"TST-0001"}]`))
			require.NoError(t, err)

			got := c.Get("TST-0001")
			require.NotNil(t, got)
			assert.Equal(t, "TST-0001", got.CardID)
		})

		cases := []struct {
			name           string
			data           []byte
			wantErrContain string
		}{
			{"要素0件のJSON配列から読み込むと、エラーになる", []byte(`[]`), "0 cards loaded"},
			{"JSONとして解釈できないデータから読み込むと、エラーになる", []byte(`not json`), "parse card data"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				c := cache.NewCardCache()

				err := c.LoadFromBytes(tc.data)

				assert.ErrorContains(t, err, tc.wantErrContain)
			})
		}
	})
}
