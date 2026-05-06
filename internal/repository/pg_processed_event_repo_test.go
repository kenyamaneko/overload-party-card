//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

// TestInsert は冪等性ガードの仕様を検証する。
// 新規 event_id は (true, nil) を返し、同一 event_id の再挿入は (false, nil) を返す。
// ON CONFLICT DO NOTHING RETURNING で pgx.ErrNoRows が潰されていることも間接的に確認する。
func TestInsert(t *testing.T) {
	tests := []struct {
		name         string
		preInsert    []string // 先に insert しておく event_id 群
		eventID      string
		eventType    string
		wantInserted bool
	}{
		{
			name:         "新規 event_id は inserted=true",
			preInsert:    nil,
			eventID:      "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			eventType:    "card_pack_purchased",
			wantInserted: true,
		},
		{
			name:         "既存 event_id は inserted=false (重複適用抑止)",
			preInsert:    []string{"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"},
			eventID:      "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
			eventType:    "card_pack_purchased",
			wantInserted: false,
		},
		{
			name:         "他の event_id が存在しても新規は inserted=true",
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
}
