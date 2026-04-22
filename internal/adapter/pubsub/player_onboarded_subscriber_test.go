package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiscenario "github.com/kenyamaneko/overload-party-scenario/packages/api-scenario"
)

// fakeInitialGranter は initialPackGranter のテスト用スタブです。
type fakeInitialGranter struct {
	err          error
	calls        int
	lastPlayerID string
	lastFaction  string
}

func (f *fakeInitialGranter) GrantInitialPack(_ context.Context, playerID, faction string) (int, error) {
	f.calls++
	f.lastPlayerID = playerID
	f.lastFaction = faction
	if f.err != nil {
		return 0, f.err
	}
	return 6, nil
}

func TestPlayerOnboardedSubscriber_Process(t *testing.T) {
	validEvent := apiscenario.PlayerOnboardedEvent{
		EventType:        apiscenario.EventTypePlayerOnboarded,
		EventID:          "evt-1",
		PlayerID:         "player-1",
		DisplayName:      "Alice",
		InitialFactionID: "SHE",
	}
	validPayload, err := json.Marshal(validEvent)
	require.NoError(t, err)

	unknownEvent := struct {
		EventType string `json:"event_type"`
		EventID   string `json:"event_id"`
	}{EventType: "unknown", EventID: "evt-2"}
	unknownPayload, err := json.Marshal(unknownEvent)
	require.NoError(t, err)

	tests := []struct {
		name             string
		payload          []byte
		repoInsertResult bool
		repoInsertErr    error
		granterErr       error
		wantAck          bool
		wantGrantCalls   int
	}{
		{
			name:             "新規イベントは初期パック (faction + Neutral) を配布して Ack する",
			payload:          validPayload,
			repoInsertResult: true,
			wantAck:          true,
			wantGrantCalls:   1,
		},
		{
			name:             "重複イベント (processed_events 既存) は配布せず Ack する",
			payload:          validPayload,
			repoInsertResult: false,
			wantAck:          true,
			wantGrantCalls:   0,
		},
		{
			name:          "processed_events insert 失敗は Nack する",
			payload:       validPayload,
			repoInsertErr: errors.New("db down"),
			wantAck:       false,
		},
		{
			name:             "GrantInitialPack 失敗は Nack する",
			payload:          validPayload,
			repoInsertResult: true,
			granterErr:       errors.New("grant failed"),
			wantAck:          false,
			wantGrantCalls:   1,
		},
		{
			name:    "malformed JSON は Nack する",
			payload: []byte("not-json"),
			wantAck: false,
		},
		{
			name:    "未知の event_type は Nack する (publisher バグを DLQ で検出)",
			payload: unknownPayload,
			wantAck: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			granter := &fakeInitialGranter{err: tt.granterErr}
			repo := &fakeProcessedEventRepo{
				insertResult: tt.repoInsertResult,
				insertErr:    tt.repoInsertErr,
			}
			sub := &PlayerOnboardedSubscriber{
				grantService: granter,
				eventRepo:    repo,
			}

			gotAck := sub.process(context.Background(), tt.payload)

			assert.Equal(t, tt.wantAck, gotAck)
			assert.Equal(t, tt.wantGrantCalls, granter.calls)
		})
	}
}

func TestPlayerOnboardedSubscriber_Process_GrantsInitialFaction(t *testing.T) {
	// ADR-022: player-onboarded subscriber は initial_faction_id のカードと
	// Neutral カードを配る (GrantInitialPack)。initial_faction_id が
	// GrantInitialPack へそのまま渡ることを確認する。
	ev := apiscenario.PlayerOnboardedEvent{
		EventType:        apiscenario.EventTypePlayerOnboarded,
		EventID:          "evt-1",
		PlayerID:         "player-1",
		DisplayName:      "Bob",
		InitialFactionID: "Tuners",
	}
	payload, err := json.Marshal(ev)
	require.NoError(t, err)

	granter := &fakeInitialGranter{}
	repo := &fakeProcessedEventRepo{insertResult: true}
	sub := &PlayerOnboardedSubscriber{
		grantService: granter,
		eventRepo:    repo,
	}

	require.True(t, sub.process(context.Background(), payload))

	assert.Equal(t, 1, granter.calls)
	assert.Equal(t, "player-1", granter.lastPlayerID)
	assert.Equal(t, "Tuners", granter.lastFaction)
}
