//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/repository"
)

func TestProcessedEventRepoInsert(t *testing.T) {
	t.Run("[処理済みイベントリポジトリ] 処理済みイベント記録", func(t *testing.T) {
		t.Run("指定したevent_idが未処理のとき、記録に成功したことを示すtrueが返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgProcessedEventRepository(sharedPg.Pool)

			inserted, err := repo.Insert(context.Background(), "44444444-4444-4444-4444-444444444444", "card_pack_purchased")

			require.NoError(t, err)
			require.True(t, inserted)
		})

		t.Run("指定したevent_idが既に処理済みのとき、記録が行われなかったことを示すfalseが返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgProcessedEventRepository(sharedPg.Pool)
			eventID := "44444444-4444-4444-4444-444444444444"
			_, err := repo.Insert(context.Background(), eventID, "card_pack_purchased")
			require.NoError(t, err)

			inserted, err := repo.Insert(context.Background(), eventID, "card_pack_purchased")

			require.NoError(t, err)
			require.False(t, inserted)
		})

		t.Run("既に処理済みのevent_idに対して、異なるevent_typeを指定して記録しようとしても、DBに保存されているevent_typeは最初に登録した値のまま変わらない", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgProcessedEventRepository(sharedPg.Pool)
			eventID := "44444444-4444-4444-4444-444444444444"
			_, err := repo.Insert(context.Background(), eventID, "card_pack_purchased")
			require.NoError(t, err)

			_, err = repo.Insert(context.Background(), eventID, "player_onboarded")
			require.NoError(t, err)

			require.Equal(t, "card_pack_purchased", fetchProcessedEventType(t, eventID))
		})

		t.Run("event_idがUUID形式でない文字列のとき、エラーを返す", func(t *testing.T) {
			sharedPg.Truncate(t)
			repo := repository.NewPgProcessedEventRepository(sharedPg.Pool)

			_, err := repo.Insert(context.Background(), "not-a-uuid", "card_pack_purchased")

			require.Error(t, err)
		})
	})
}
