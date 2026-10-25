package pubsub_test

import (
	"context"
	"encoding/json"
	"testing"

	apishop "github.com/kenyamaneko/overload-party-shop/packages/api-shop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/handler/pubsub"
	"github.com/kenyamaneko/overload-party-card/internal/handler/pubsub/pubsubtest"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"
)

func newFakeGrantStoreWithPurchasedPack() *pubsubtest.FakeGrantStore {
	return pubsubtest.NewFakeGrantStore(domain.CardPack{
		PackID:   "TST-0100",
		IsActive: true,
		Cards: []domain.CardPackCard{
			{CardID: "TST-0001", Copies: 1},
			{CardID: "TST-0002", Copies: 2},
		},
	})
}

func newFakeGrantStoreWithPurchasedPackAndOwnedCard() *pubsubtest.FakeGrantStore {
	store := pubsubtest.NewFakeGrantStore(domain.CardPack{
		PackID:   "TST-0100",
		IsActive: true,
		Cards:    []domain.CardPackCard{{CardID: "TST-0001", Copies: 1}},
	})
	return store.SeedOwnedCard(pubsubtest.PlayerID, "TST-0009", 2)
}

func newCardPackPurchasedSubscriber(store *pubsubtest.FakeGrantStore) *pubsub.CardPackPurchasedSubscriber {
	return pubsub.NewCardPackPurchasedSubscriber(usecase.NewGrantInteractor(store))
}

func marshalCardPackPurchasedEvent(t *testing.T, eventType, eventID string) []byte {
	t.Helper()
	payload, err := json.Marshal(apishop.CardPackPurchasedEvent{
		EventType:  eventType,
		EventID:    eventID,
		PlayerID:   pubsubtest.PlayerID,
		CardPackID: "TST-0100",
	})
	require.NoError(t, err)
	return payload
}

func TestCardPackPurchasedSubscriber(t *testing.T) {
	t.Run("[カードパック購入通知購読]購入パックの配布", func(t *testing.T) {
		ctx := context.Background()

		t.Run("正常系", func(t *testing.T) {
			t.Run("購入されたカードパックがTST-0100で、TST-0100がTST-0001を1枚とTST-0002を2枚含み、プレイヤーの所持カードが無いとき", func(t *testing.T) {
				t.Run("未処理のカードパック購入イベントを受け取ったとき", func(t *testing.T) {
					t.Run("処理は成功で終わる", func(t *testing.T) {
						subscriber := newCardPackPurchasedSubscriber(newFakeGrantStoreWithPurchasedPack())

						err := subscriber.GrantPurchasedCards(ctx, marshalCardPackPurchasedEvent(t, apishop.EventTypeCardPackPurchased, pubsubtest.EventID))

						assert.NoError(t, err)
					})

					t.Run("TST-0001の所持枚数が1枚、TST-0002の所持枚数が2枚になる", func(t *testing.T) {
						store := newFakeGrantStoreWithPurchasedPack()
						subscriber := newCardPackPurchasedSubscriber(store)
						require.NoError(t, subscriber.GrantPurchasedCards(ctx, marshalCardPackPurchasedEvent(t, apishop.EventTypeCardPackPurchased, pubsubtest.EventID)))

						counts := pubsubtest.ReadOwnedCardCounts(t, store, pubsubtest.PlayerID)

						assert.Equal(t, 1, counts["TST-0001"])
						assert.Equal(t, 2, counts["TST-0002"])
					})
				})

				t.Run("同じイベントを2回続けて受け取ったとき", func(t *testing.T) {
					t.Run("2回目の処理は成功で終わる", func(t *testing.T) {
						subscriber := newCardPackPurchasedSubscriber(newFakeGrantStoreWithPurchasedPack())
						payload := marshalCardPackPurchasedEvent(t, apishop.EventTypeCardPackPurchased, pubsubtest.EventID)
						require.NoError(t, subscriber.GrantPurchasedCards(ctx, payload))

						err := subscriber.GrantPurchasedCards(ctx, payload)

						assert.NoError(t, err)
					})

					t.Run("TST-0001の所持枚数が1枚、TST-0002の所持枚数が2枚になる", func(t *testing.T) {
						store := newFakeGrantStoreWithPurchasedPack()
						subscriber := newCardPackPurchasedSubscriber(store)
						payload := marshalCardPackPurchasedEvent(t, apishop.EventTypeCardPackPurchased, pubsubtest.EventID)
						require.NoError(t, subscriber.GrantPurchasedCards(ctx, payload))
						require.NoError(t, subscriber.GrantPurchasedCards(ctx, payload))

						counts := pubsubtest.ReadOwnedCardCounts(t, store, pubsubtest.PlayerID)

						assert.Equal(t, 1, counts["TST-0001"])
						assert.Equal(t, 2, counts["TST-0002"])
					})
				})

				t.Run("イベントIDが異なる2つのイベントを同じプレイヤーと同じカードパックについて受け取ったとき、TST-0001の所持枚数が2枚、TST-0002の所持枚数が4枚になる", func(t *testing.T) {
					store := newFakeGrantStoreWithPurchasedPack()
					subscriber := newCardPackPurchasedSubscriber(store)
					require.NoError(t, subscriber.GrantPurchasedCards(ctx, marshalCardPackPurchasedEvent(t, apishop.EventTypeCardPackPurchased, pubsubtest.EventID)))
					require.NoError(t, subscriber.GrantPurchasedCards(ctx, marshalCardPackPurchasedEvent(t, apishop.EventTypeCardPackPurchased, pubsubtest.OtherEventID)))

					counts := pubsubtest.ReadOwnedCardCounts(t, store, pubsubtest.PlayerID)

					assert.Equal(t, 2, counts["TST-0001"])
					assert.Equal(t, 4, counts["TST-0002"])
				})
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("内容が文字列not-jsonのとき、内容が不正であることを示すエラーになる", func(t *testing.T) {
				subscriber := newCardPackPurchasedSubscriber(newFakeGrantStoreWithPurchasedPack())

				err := subscriber.GrantPurchasedCards(ctx, []byte(pubsubtest.MalformedPayload))

				assert.ErrorIs(t, err, pubsub.ErrMalformedPayload)
			})

			t.Run("イベント種別がTST_UNKNOWNのとき、想定外のイベント種別であることを示すエラーになる", func(t *testing.T) {
				subscriber := newCardPackPurchasedSubscriber(newFakeGrantStoreWithPurchasedPack())

				err := subscriber.GrantPurchasedCards(ctx, marshalCardPackPurchasedEvent(t, pubsubtest.UnknownEventType, pubsubtest.EventID))

				assert.ErrorIs(t, err, pubsub.ErrUnexpectedEventType)
			})

			t.Run("TST-0100のカードパック購入イベントを受け取り、処理済みイベントの保管先が保存に失敗したとき、処理済みイベントの保存に失敗したことを示すエラーになる", func(t *testing.T) {
				subscriber := newCardPackPurchasedSubscriber(newFakeGrantStoreWithPurchasedPack().FailProcessedEventInsert())

				err := subscriber.GrantPurchasedCards(ctx, marshalCardPackPurchasedEvent(t, apishop.EventTypeCardPackPurchased, pubsubtest.EventID))

				assert.ErrorIs(t, err, pubsubtest.ErrProcessedEventStoreFailure)
			})

			t.Run("購入されたカードパックがTST-0100で、TST-0100がTST-0001を1枚含み、プレイヤーがTST-0009を2枚所持しているとき、イベントを受け取ってエラーになると", func(t *testing.T) {
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
						payload: marshalCardPackPurchasedEvent(t, pubsubtest.UnknownEventType, pubsubtest.EventID),
					},
					{
						name:      "TST-0100のカードパック購入イベントを受け取り、処理済みイベントの保管先が保存に失敗したとき、所持カードがTST-0009の2枚だけのままになる",
						payload:   marshalCardPackPurchasedEvent(t, apishop.EventTypeCardPackPurchased, pubsubtest.EventID),
						insertErr: pubsubtest.ErrProcessedEventStoreFailure,
					},
				}
				for _, tt := range tests {
					t.Run(tt.name, func(t *testing.T) {
						store := newFakeGrantStoreWithPurchasedPackAndOwnedCard()
						store.InsertErr = tt.insertErr
						subscriber := newCardPackPurchasedSubscriber(store)
						require.Error(t, subscriber.GrantPurchasedCards(ctx, tt.payload))

						counts := pubsubtest.ReadOwnedCardCounts(t, store, pubsubtest.PlayerID)

						assert.Equal(t, map[string]int{"TST-0009": 2}, counts)
					})
				}
			})
		})
	})
}
