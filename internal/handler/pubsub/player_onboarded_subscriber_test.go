package pubsub_test

import (
	"context"
	"encoding/json"
	"testing"

	apiscenario "github.com/kenyamaneko/overload-party-scenario/packages/api-scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/handler/pubsub"
	"github.com/kenyamaneko/overload-party-card/internal/handler/pubsub/pubsubtest"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"
)

func newFakeGrantStoreWithOnboardedPacks() *pubsubtest.FakeGrantStore {
	return pubsubtest.NewFakeGrantStore(
		domain.CardPack{
			PackID:   "basic",
			IsActive: true,
			Cards: []domain.CardPackCard{
				{CardID: "TST-0001", Copies: 1},
				{CardID: "TST-0002", Copies: 2},
			},
		},
		domain.CardPack{
			PackID:   "faction_set_she",
			IsActive: true,
			Cards:    []domain.CardPackCard{{CardID: "TST-0003", Copies: 3}},
		},
	)
}

func newFakeGrantStoreWithOnboardedPacksAndOwnedCard() *pubsubtest.FakeGrantStore {
	store := pubsubtest.NewFakeGrantStore(
		domain.CardPack{PackID: "basic", IsActive: true, Cards: []domain.CardPackCard{{CardID: "TST-0001", Copies: 1}}},
		domain.CardPack{PackID: "faction_set_she", IsActive: true, Cards: []domain.CardPackCard{{CardID: "TST-0003", Copies: 3}}},
	)
	return store.SeedOwnedCard(pubsubtest.PlayerID, "TST-0009", 2)
}

func newPlayerOnboardedSubscriber(store *pubsubtest.FakeGrantStore) *pubsub.PlayerOnboardedSubscriber {
	return pubsub.NewPlayerOnboardedSubscriber(usecase.NewGrantInteractor(store))
}

func marshalPlayerOnboardedEvent(t *testing.T, eventType, eventID string) []byte {
	t.Helper()
	payload, err := json.Marshal(apiscenario.PlayerOnboardedEvent{
		EventType:        eventType,
		EventID:          eventID,
		PlayerID:         pubsubtest.PlayerID,
		InitialFactionID: "SHE",
	})
	require.NoError(t, err)
	return payload
}

func TestPlayerOnboardedSubscriber(t *testing.T) {
	t.Run("[プレイヤー登録完了通知購読]初期カードの配布", func(t *testing.T) {
		ctx := context.Background()

		t.Run("正常系", func(t *testing.T) {
			t.Run("イベントの初期陣営がSHEで、プレイヤーの所持カードが無く、基本パックがTST-0001を1枚とTST-0002を2枚、faction_set_sheがTST-0003を3枚含むとき", func(t *testing.T) {
				t.Run("未処理のプレイヤー登録完了イベントを受け取ったとき", func(t *testing.T) {
					t.Run("処理は成功で終わる", func(t *testing.T) {
						subscriber := newPlayerOnboardedSubscriber(newFakeGrantStoreWithOnboardedPacks())

						err := subscriber.GrantInitialCards(ctx, marshalPlayerOnboardedEvent(t, apiscenario.EventTypePlayerOnboarded, pubsubtest.EventID))

						assert.NoError(t, err)
					})

					t.Run("基本パックのTST-0001の所持枚数が1枚、TST-0002の所持枚数が2枚になる", func(t *testing.T) {
						store := newFakeGrantStoreWithOnboardedPacks()
						subscriber := newPlayerOnboardedSubscriber(store)
						require.NoError(t, subscriber.GrantInitialCards(ctx, marshalPlayerOnboardedEvent(t, apiscenario.EventTypePlayerOnboarded, pubsubtest.EventID)))

						counts := pubsubtest.ReadOwnedCardCounts(t, store, pubsubtest.PlayerID)

						assert.Equal(t, 1, counts["TST-0001"])
						assert.Equal(t, 2, counts["TST-0002"])
					})

					t.Run("faction_set_sheのTST-0003の所持枚数が3枚になる", func(t *testing.T) {
						store := newFakeGrantStoreWithOnboardedPacks()
						subscriber := newPlayerOnboardedSubscriber(store)
						require.NoError(t, subscriber.GrantInitialCards(ctx, marshalPlayerOnboardedEvent(t, apiscenario.EventTypePlayerOnboarded, pubsubtest.EventID)))

						counts := pubsubtest.ReadOwnedCardCounts(t, store, pubsubtest.PlayerID)

						assert.Equal(t, 3, counts["TST-0003"])
					})
				})

				t.Run("同じイベントを2回続けて受け取ったとき", func(t *testing.T) {
					t.Run("2回目の処理は成功で終わる", func(t *testing.T) {
						subscriber := newPlayerOnboardedSubscriber(newFakeGrantStoreWithOnboardedPacks())
						payload := marshalPlayerOnboardedEvent(t, apiscenario.EventTypePlayerOnboarded, pubsubtest.EventID)
						require.NoError(t, subscriber.GrantInitialCards(ctx, payload))

						err := subscriber.GrantInitialCards(ctx, payload)

						assert.NoError(t, err)
					})

					t.Run("基本パックのTST-0001の所持枚数が1枚、TST-0002の所持枚数が2枚になる", func(t *testing.T) {
						store := newFakeGrantStoreWithOnboardedPacks()
						subscriber := newPlayerOnboardedSubscriber(store)
						payload := marshalPlayerOnboardedEvent(t, apiscenario.EventTypePlayerOnboarded, pubsubtest.EventID)
						require.NoError(t, subscriber.GrantInitialCards(ctx, payload))
						require.NoError(t, subscriber.GrantInitialCards(ctx, payload))

						counts := pubsubtest.ReadOwnedCardCounts(t, store, pubsubtest.PlayerID)

						assert.Equal(t, 1, counts["TST-0001"])
						assert.Equal(t, 2, counts["TST-0002"])
					})

					t.Run("faction_set_sheのTST-0003の所持枚数が3枚になる", func(t *testing.T) {
						store := newFakeGrantStoreWithOnboardedPacks()
						subscriber := newPlayerOnboardedSubscriber(store)
						payload := marshalPlayerOnboardedEvent(t, apiscenario.EventTypePlayerOnboarded, pubsubtest.EventID)
						require.NoError(t, subscriber.GrantInitialCards(ctx, payload))
						require.NoError(t, subscriber.GrantInitialCards(ctx, payload))

						counts := pubsubtest.ReadOwnedCardCounts(t, store, pubsubtest.PlayerID)

						assert.Equal(t, 3, counts["TST-0003"])
					})
				})

				t.Run("イベントIDが異なる2つのイベントを同じプレイヤーについて受け取ったとき", func(t *testing.T) {
					t.Run("基本パックのTST-0001の所持枚数が2枚、TST-0002の所持枚数が4枚になる", func(t *testing.T) {
						store := newFakeGrantStoreWithOnboardedPacks()
						subscriber := newPlayerOnboardedSubscriber(store)
						require.NoError(t, subscriber.GrantInitialCards(ctx, marshalPlayerOnboardedEvent(t, apiscenario.EventTypePlayerOnboarded, pubsubtest.EventID)))
						require.NoError(t, subscriber.GrantInitialCards(ctx, marshalPlayerOnboardedEvent(t, apiscenario.EventTypePlayerOnboarded, pubsubtest.OtherEventID)))

						counts := pubsubtest.ReadOwnedCardCounts(t, store, pubsubtest.PlayerID)

						assert.Equal(t, 2, counts["TST-0001"])
						assert.Equal(t, 4, counts["TST-0002"])
					})

					t.Run("faction_set_sheのTST-0003の所持枚数が6枚になる", func(t *testing.T) {
						store := newFakeGrantStoreWithOnboardedPacks()
						subscriber := newPlayerOnboardedSubscriber(store)
						require.NoError(t, subscriber.GrantInitialCards(ctx, marshalPlayerOnboardedEvent(t, apiscenario.EventTypePlayerOnboarded, pubsubtest.EventID)))
						require.NoError(t, subscriber.GrantInitialCards(ctx, marshalPlayerOnboardedEvent(t, apiscenario.EventTypePlayerOnboarded, pubsubtest.OtherEventID)))

						counts := pubsubtest.ReadOwnedCardCounts(t, store, pubsubtest.PlayerID)

						assert.Equal(t, 6, counts["TST-0003"])
					})
				})
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("内容が文字列not-jsonのとき、内容が不正であることを示すエラーになる", func(t *testing.T) {
				subscriber := newPlayerOnboardedSubscriber(newFakeGrantStoreWithOnboardedPacks())

				err := subscriber.GrantInitialCards(ctx, []byte(pubsubtest.MalformedPayload))

				assert.ErrorIs(t, err, pubsub.ErrMalformedPayload)
			})

			t.Run("イベント種別がTST_UNKNOWNのとき、想定外のイベント種別であることを示すエラーになる", func(t *testing.T) {
				subscriber := newPlayerOnboardedSubscriber(newFakeGrantStoreWithOnboardedPacks())

				err := subscriber.GrantInitialCards(ctx, marshalPlayerOnboardedEvent(t, pubsubtest.UnknownEventType, pubsubtest.EventID))

				assert.ErrorIs(t, err, pubsub.ErrUnexpectedEventType)
			})

			t.Run("初期陣営がSHEのイベントを受け取り、処理済みイベントの保管先が保存に失敗したとき、処理済みイベントの保存に失敗したことを示すエラーになる", func(t *testing.T) {
				subscriber := newPlayerOnboardedSubscriber(newFakeGrantStoreWithOnboardedPacks().FailProcessedEventInsert())

				err := subscriber.GrantInitialCards(ctx, marshalPlayerOnboardedEvent(t, apiscenario.EventTypePlayerOnboarded, pubsubtest.EventID))

				assert.ErrorIs(t, err, pubsubtest.ErrProcessedEventStoreFailure)
			})

			t.Run("基本パックがTST-0001を1枚、faction_set_sheがTST-0003を3枚含み、プレイヤーがTST-0009を2枚所持しているとき、イベントを受け取ってエラーになると", func(t *testing.T) {
				tests := []struct {
					name      string
					payload   []byte
					insertErr error
				}{
					{
						name:    "内容が文字列not-jsonのとき、所持カードがTST-0009の2枚だけのままになる",
						payload: []byte(pubsubtest.MalformedPayload),
					},
					{
						name:    "イベント種別がTST_UNKNOWNのとき、所持カードがTST-0009の2枚だけのままになる",
						payload: marshalPlayerOnboardedEvent(t, pubsubtest.UnknownEventType, pubsubtest.EventID),
					},
					{
						name:      "初期陣営がSHEのイベントを受け取り、処理済みイベントの保管先が保存に失敗したとき、所持カードがTST-0009の2枚だけのままになる",
						payload:   marshalPlayerOnboardedEvent(t, apiscenario.EventTypePlayerOnboarded, pubsubtest.EventID),
						insertErr: pubsubtest.ErrProcessedEventStoreFailure,
					},
				}
				for _, tt := range tests {
					t.Run(tt.name, func(t *testing.T) {
						store := newFakeGrantStoreWithOnboardedPacksAndOwnedCard()
						store.InsertErr = tt.insertErr
						subscriber := newPlayerOnboardedSubscriber(store)
						require.Error(t, subscriber.GrantInitialCards(ctx, tt.payload))

						counts := pubsubtest.ReadOwnedCardCounts(t, store, pubsubtest.PlayerID)

						assert.Equal(t, map[string]int{"TST-0009": 2}, counts)
					})
				}
			})
		})
	})
}
