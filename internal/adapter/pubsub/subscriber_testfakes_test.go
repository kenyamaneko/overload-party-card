package pubsub

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// fakeProcessedEventRepo は port.ProcessedEventRepo のテスト用スタブ。
// Insert は insertResult / insertErr の固定値で振る舞いを制御し、subscriber の
// fast-path dedup 分岐を確定的にテストできる。
type fakeProcessedEventRepo struct {
	insertResult bool
	insertErr    error
}

var _ port.ProcessedEventRepo = (*fakeProcessedEventRepo)(nil)

func (r *fakeProcessedEventRepo) Insert(_ context.Context, _, _ string) (bool, error) {
	if r.insertErr != nil {
		return false, r.insertErr
	}
	return r.insertResult, nil
}

// mustMarshal は json.Marshal の testing 版。event_type 欠落など typed helper
// (apishopfake.PublishXxx / apiscenariofake.PublishXxx) で表現できない壊れた
// ペイロードをテストケースに含めるためのヘルパ。
func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
