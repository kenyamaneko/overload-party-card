package pubsub

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// fakeProcessedEventRepo は port.ProcessedEventRepo のテスト用スタブ。
// event_id ごとの挿入済み状態を追跡し、同じ event_id への 2 回目以降の Insert が
// false を返す実 DB の UNIQUE 制約相当の挙動を再現する。insertErr を設定すると
// event_id によらず常にエラーを返す。
type fakeProcessedEventRepo struct {
	insertErr error
	inserted  map[string]bool
}

var _ port.ProcessedEventRepo = (*fakeProcessedEventRepo)(nil)

// newFakeProcessedEventRepo は空の fakeProcessedEventRepo を生成する。
func newFakeProcessedEventRepo() *fakeProcessedEventRepo {
	return &fakeProcessedEventRepo{inserted: make(map[string]bool)}
}

func (r *fakeProcessedEventRepo) Insert(_ context.Context, eventID, _ string) (bool, error) {
	if r.insertErr != nil {
		return false, r.insertErr
	}
	if r.inserted[eventID] {
		return false, nil
	}
	r.inserted[eventID] = true
	return true, nil
}

// mustMarshal は json.Marshal の testing 版。event_type 不一致など、意図的に
// 壊れた値を持つ typed event のペイロードをテストケースに含めるためのヘルパ。
func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
