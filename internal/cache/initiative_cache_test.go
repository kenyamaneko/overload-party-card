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

type fakeInitiativeRepo struct {
	initiatives []*domain.Initiative
	err         error
}

func (f *fakeInitiativeRepo) FindAll(ctx context.Context) ([]*domain.Initiative, error) {
	return f.initiatives, f.err
}

func TestInitiativeCacheFindByID(t *testing.T) {
	t.Run("[cache] 施策定義の取得", func(t *testing.T) {
		t.Run("複数件の施策定義を読み込んだあと、読み込み済みのinitiative_id TST-0001を指定して取得すると、そのinitiative_idに対応する施策定義が返る", func(t *testing.T) {
			c := cache.NewInitiativeCache()
			repo := &fakeInitiativeRepo{initiatives: []*domain.Initiative{
				{InitiativeID: "TST-0001"},
				{InitiativeID: "TST-0002"},
			}}
			require.NoError(t, c.Load(context.Background(), repo))

			got := c.FindByID("TST-0001")

			require.NotNil(t, got)
			assert.Equal(t, "TST-0001", got.InitiativeID)
		})

		t.Run("複数件の施策定義を読み込んだあと、読み込んでいないinitiative_id TST-9999を指定して取得すると、見つからない", func(t *testing.T) {
			c := cache.NewInitiativeCache()
			repo := &fakeInitiativeRepo{initiatives: []*domain.Initiative{
				{InitiativeID: "TST-0001"},
				{InitiativeID: "TST-0002"},
			}}
			require.NoError(t, c.Load(context.Background(), repo))

			got := c.FindByID("TST-9999")

			assert.Nil(t, got)
		})
	})
}

func TestInitiativeCacheLoad(t *testing.T) {
	t.Run("[cache] 施策定義の読み込み", func(t *testing.T) {
		cases := []struct {
			name           string
			repo           *fakeInitiativeRepo
			wantErrContain string
		}{
			{
				name:           "施策定義の取得元が0件を返したとき、読み込みはエラーになる",
				repo:           &fakeInitiativeRepo{initiatives: nil},
				wantErrContain: "0 initiatives loaded",
			},
			{
				name:           "施策定義の取得元がエラーを返したとき、読み込みはエラーになる",
				repo:           &fakeInitiativeRepo{err: errors.New("fetch failed")},
				wantErrContain: "load initiative cache",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				c := cache.NewInitiativeCache()

				err := c.Load(context.Background(), tc.repo)

				assert.ErrorContains(t, err, tc.wantErrContain)
			})
		}
	})
}

func TestInitiativeCacheAll(t *testing.T) {
	t.Run("[cache] 施策定義の全件取得", func(t *testing.T) {
		t.Run("initiative_idがTST-0002、TST-0001の順で読み込まれても、全件取得するとinitiative_idの昇順で返る", func(t *testing.T) {
			c := cache.NewInitiativeCache()
			repo := &fakeInitiativeRepo{initiatives: []*domain.Initiative{
				{InitiativeID: "TST-0002"},
				{InitiativeID: "TST-0001"},
			}}
			require.NoError(t, c.Load(context.Background(), repo))

			got := c.All()

			require.Len(t, got, 2)
			assert.Equal(t, "TST-0001", got[0].InitiativeID)
			assert.Equal(t, "TST-0002", got[1].InitiativeID)
		})
	})
}

func TestInitiativeCacheCount(t *testing.T) {
	t.Run("[cache] 施策定義の件数", func(t *testing.T) {
		t.Run("読み込み前の件数は0になる", func(t *testing.T) {
			c := cache.NewInitiativeCache()

			assert.Equal(t, 0, c.Count())
		})

		t.Run("施策定義を読み込んだあと、件数は読み込んだ件数になる", func(t *testing.T) {
			c := cache.NewInitiativeCache()
			repo := &fakeInitiativeRepo{initiatives: []*domain.Initiative{
				{InitiativeID: "TST-0001"},
				{InitiativeID: "TST-0002"},
				{InitiativeID: "TST-0003"},
			}}
			require.NoError(t, c.Load(context.Background(), repo))

			assert.Equal(t, 3, c.Count())
		})
	})
}

func TestInitiativeCacheLoadFromBytes(t *testing.T) {
	t.Run("[cache] JSONからの施策定義読み込み", func(t *testing.T) {
		t.Run("施策定義1件以上を含むJSONデータから読み込むと、その施策定義が取得できるようになる", func(t *testing.T) {
			c := cache.NewInitiativeCache()

			err := c.LoadFromBytes([]byte(`[{"initiative_id":"TST-0001"}]`))
			require.NoError(t, err)

			got := c.FindByID("TST-0001")
			require.NotNil(t, got)
			assert.Equal(t, "TST-0001", got.InitiativeID)
		})

		cases := []struct {
			name           string
			data           []byte
			wantErrContain string
		}{
			{"要素0件のJSON配列から読み込むと、エラーになる", []byte(`[]`), "0 initiatives loaded"},
			{"JSONとして解釈できないデータから読み込むと、エラーになる", []byte(`not json`), "parse initiative data"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				c := cache.NewInitiativeCache()

				err := c.LoadFromBytes(tc.data)

				assert.ErrorContains(t, err, tc.wantErrContain)
			})
		}
	})
}
