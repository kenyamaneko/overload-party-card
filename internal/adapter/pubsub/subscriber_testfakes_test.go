package pubsub_test

import "context"

type grantCall struct {
	playerID string
	packID   string
}

type fakePackGranter struct {
	calls     []grantCall
	errFor    map[string]error
	copiesFor map[string]int
}

func (f *fakePackGranter) GrantPack(ctx context.Context, playerID, packID string) (int, error) {
	f.calls = append(f.calls, grantCall{playerID: playerID, packID: packID})
	if err, ok := f.errFor[packID]; ok {
		return 0, err
	}
	return f.copiesFor[packID], nil
}

type fakeProcessedEventRepo struct {
	inserted bool
	err      error
}

func (f *fakeProcessedEventRepo) Insert(ctx context.Context, eventID, eventType string) (bool, error) {
	return f.inserted, f.err
}
