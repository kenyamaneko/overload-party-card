package pubsub

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apishop "github.com/kenyamaneko/overload-party-shop/packages/api-shop"
)

// fakeFactionGranter は factionPackGranter のテスト用スタブです。
type fakeFactionGranter struct {
	err          error
	calls        int
	lastPlayerID string
	lastFaction  string
}

func (f *fakeFactionGranter) GrantFactionPack(_ context.Context, playerID, faction string) (int, error) {
	f.calls++
	f.lastPlayerID = playerID
	f.lastFaction = faction
	if f.err != nil {
		return 0, f.err
	}
	return 3, nil
}

// fakeProcessedEventRepo は port.ProcessedEventRepo のテスト用スタブです。
type fakeProcessedEventRepo struct {
	insertResult bool
	insertErr    error
	calls        int
}

func (r *fakeProcessedEventRepo) Insert(_ context.Context, _, _ string) (bool, error) {
	r.calls++
	if r.insertErr != nil {
		return false, r.insertErr
	}
	return r.insertResult, nil
}

func TestFactionPurchasedSubscriber_Process(t *testing.T) {
	validEvent := apishop.FactionPurchasedEvent{
		EventType: apishop.EventTypeFactionPurchased,
		EventID:   "evt-1",
		PlayerID:  "player-1",
		Faction:   "SHE",
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
			name:             "新規イベントは faction のみ配布して Ack する",
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
			name:             "GrantFactionPack 失敗は Nack する",
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
			granter := &fakeFactionGranter{err: tt.granterErr}
			repo := &fakeProcessedEventRepo{
				insertResult: tt.repoInsertResult,
				insertErr:    tt.repoInsertErr,
			}
			sub := &FactionPurchasedSubscriber{
				grantService: granter,
				eventRepo:    repo,
			}

			gotAck := sub.process(context.Background(), tt.payload)

			assert.Equal(t, tt.wantAck, gotAck)
			assert.Equal(t, tt.wantGrantCalls, granter.calls)
		})
	}
}

func TestFactionPurchasedSubscriber_Process_GrantsFactionOnly(t *testing.T) {
	// ADR-022: faction-purchased subscriber は Neutral を配らず faction のみ配る。
	// GrantFactionPack が faction 引数のみで呼ばれること (GrantInitialPack ではないこと)
	// を確認する。
	ev := apishop.FactionPurchasedEvent{
		EventType: apishop.EventTypeFactionPurchased,
		EventID:   "evt-1",
		PlayerID:  "player-1",
		Faction:   "Tenki",
	}
	payload, err := json.Marshal(ev)
	require.NoError(t, err)

	granter := &fakeFactionGranter{}
	repo := &fakeProcessedEventRepo{insertResult: true}
	sub := &FactionPurchasedSubscriber{
		grantService: granter,
		eventRepo:    repo,
	}

	require.True(t, sub.process(context.Background(), payload))

	assert.Equal(t, 1, granter.calls)
	assert.Equal(t, "player-1", granter.lastPlayerID)
	assert.Equal(t, "Tenki", granter.lastFaction)
}
