package pubsub

import (
	"context"
	"testing"
	"time"

	"github.com/kenyamaneko/overload-party-scenario/packages/api-scenario/apiscenariofake"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// apiscenariofakeStream は apiscenariofake.Subscriber を port.MessageStream として
// 露出するテスト用 adapter。apishopfakeStream と同じ設計 (eager subscribe +
// handled channel) を scenario 側 fake 向けに再掲している。
type apiscenariofakeStream struct {
	ch      <-chan []byte
	topic   string
	handled chan error
}

func newApiscenariofakeStream(sub *apiscenariofake.Subscriber, topic string) *apiscenariofakeStream {
	return &apiscenariofakeStream{
		ch:      sub.Messages(topic),
		topic:   topic,
		handled: make(chan error, 16),
	}
}

func (s *apiscenariofakeStream) Consume(ctx context.Context, handler port.MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case data, ok := <-s.ch:
			if !ok {
				return nil
			}
			s.handled <- handler(ctx, data)
		}
	}
}

func (s *apiscenariofakeStream) ExpectHandled(t *testing.T, timeout time.Duration) error {
	t.Helper()
	select {
	case err := <-s.handled:
		return err
	case <-time.After(timeout):
		t.Fatalf("apiscenariofakeStream[%s]: timeout waiting for handler completion (%s)", s.topic, timeout)
		return nil
	}
}
