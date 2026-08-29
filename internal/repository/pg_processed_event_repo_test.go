//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

func TestInsert(t *testing.T) {
	t.Run("[処理済みイベントリポジトリ]processed_eventsへの冪等挿入", func(t *testing.T) {
		// ON CONFLICT DO NOTHING RETURNING で pgx.ErrNoRows を潰し、新規は true・重複は false を返す。
		tests := []struct {
			name         string
			preInsert    []string // 先に insert しておく event_id 群
			eventID      string
			eventType    string
			wantInserted bool
		}{
			{
				name:         "新規event_idのとき、inserted=trueになる",
				preInsert:    nil,
				eventID:      "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				eventType:    "card_pack_purchased",
				wantInserted: true,
			},
			{
				name:         "既存event_idのとき、inserted=falseになる (重複適用抑止)",
				preInsert:    []string{"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"},
				eventID:      "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
				eventType:    "card_pack_purchased",
				wantInserted: false,
			},
			{
				name:         "他のevent_idが存在しても新規のとき、inserted=trueになる",
				preInsert:    []string{"cccccccc-cccc-cccc-cccc-cccccccccccc"},
				eventID:      "dddddddd-dddd-dddd-dddd-dddddddddddd",
				eventType:    "player_onboarded",
				wantInserted: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				sharedPg.Truncate(t)
				repo := repository.NewPgProcessedEventRepository(sharedPg.Pool)
				ctx := context.Background()

				for _, id := range tt.preInsert {
					_, err := repo.Insert(ctx, id, "card_pack_purchased")
					require.NoError(t, err)
				}

				got, err := repo.Insert(ctx, tt.eventID, tt.eventType)
				require.NoError(t, err)
				assert.Equal(t, tt.wantInserted, got)
			})
		}
	})
}
