package rest_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

func TestCardHandlerListAllRaw(t *testing.T) {
	t.Run("[handler] カードマスター全件配信", func(t *testing.T) {
		t.Run("カード定義が取得できるとき、200になり各要素がカード定義の内容と一致する", func(t *testing.T) {
			card := newTestCard("TST-0001", "SHE", "Compute", "unlimited")
			engine := newTestRouter(t, withCardRepo(&fakeCardRepo{
				FindAllFn: func(ctx context.Context) ([]*domain.Card, error) {
					return []*domain.Card{card}, nil
				},
			}))

			rr := doRequest(t, engine, http.MethodGet, "/internal/v1/cards", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			var body []apicard.CardDefinition
			decodeJSON(t, rr, &body)
			require.Len(t, body, 1)
			got := body[0]
			assert.Equal(t, card.CardID, got.CardID)
			assert.Equal(t, card.CardName, got.CardName)
			assert.Equal(t, card.ResourceLabel, got.ResourceLabel)
			assert.Equal(t, card.Faction, got.Faction)
			assert.Equal(t, card.CardType, got.CardType)
			assert.Equal(t, card.Resizable, got.Resizable)
			assert.Equal(t, card.Elastic, got.Elastic)
			assert.JSONEq(t, string(card.Stats), string(got.Stats))
			assert.Equal(t, card.Restriction, got.Restriction)
			assert.Equal(t, card.IsActive, got.IsActive)
			assert.True(t, card.CreatedAt.Equal(got.CreatedAt))
			assert.True(t, card.UpdatedAt.Equal(got.UpdatedAt))
		})

		t.Run("カード定義が1件も無いとき、200になりレスポンスボディは空配列になる", func(t *testing.T) {
			engine := newTestRouter(t, withCardRepo(&fakeCardRepo{
				FindAllFn: func(ctx context.Context) ([]*domain.Card, error) { return []*domain.Card{}, nil },
			}))

			rr := doRequest(t, engine, http.MethodGet, "/internal/v1/cards", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			assert.JSONEq(t, "[]", rr.Body.String())
		})

		t.Run("port.CardRepo.FindAllがエラーを返すとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			engine := newTestRouter(t, withCardRepo(&fakeCardRepo{
				FindAllFn: func(ctx context.Context) ([]*domain.Card, error) { return nil, errors.New("find all failed") },
			}))

			rr := doRequest(t, engine, http.MethodGet, "/internal/v1/cards", nil)

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})
	})
}

func TestCardHandlerListForPlayer(t *testing.T) {
	t.Run("[handler] 所持状態付きカード一覧取得", func(t *testing.T) {
		t.Run("カードがプレイヤーの所持カード一覧に1件以上含まれるとき、そのカードのis_ownedはtrueになる", func(t *testing.T) {
			card := newTestCard("TST-0001", "SHE", "Compute", "unlimited")
			engine := newTestRouter(t,
				withCardRepo(&fakeCardRepo{FindAllFn: func(ctx context.Context) ([]*domain.Card, error) {
					return []*domain.Card{card}, nil
				}}),
				withPlayerCardRepo(&fakePlayerCardRepo{GetPlayerCardsFn: func(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
					return []*domain.PlayerCard{{PlayerID: playerID, CardID: "TST-0001", ArtNo: 1, Count: 1}}, nil
				}}),
			)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/cards/with-ownership", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			var body []apicard.CardWithOwnership
			decodeJSON(t, rr, &body)
			got := findCardWithOwnership(t, body, "TST-0001")
			assert.True(t, got.IsOwned)
		})

		t.Run("カードがプレイヤーの所持カード一覧に含まれないとき、そのカードのis_ownedはfalseになる", func(t *testing.T) {
			card := newTestCard("TST-0002", "SHE", "Compute", "unlimited")
			engine := newTestRouter(t,
				withCardRepo(&fakeCardRepo{FindAllFn: func(ctx context.Context) ([]*domain.Card, error) {
					return []*domain.Card{card}, nil
				}}),
				withPlayerCardRepo(&fakePlayerCardRepo{GetPlayerCardsFn: func(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
					return []*domain.PlayerCard{}, nil
				}}),
			)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/cards/with-ownership", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			var body []apicard.CardWithOwnership
			decodeJSON(t, rr, &body)
			got := findCardWithOwnership(t, body, "TST-0002")
			assert.False(t, got.IsOwned)
		})

		errorCases := []struct {
			name           string
			cardRepo       *fakeCardRepo
			playerCardRepo *fakePlayerCardRepo
		}{
			{
				name: "port.CardRepo.FindAllがエラーを返すとき、500になりボディのerrorフィールドはinternal server errorになる",
				cardRepo: &fakeCardRepo{FindAllFn: func(ctx context.Context) ([]*domain.Card, error) {
					return nil, errors.New("find all failed")
				}},
				playerCardRepo: &fakePlayerCardRepo{},
			},
			{
				name: "port.CardRepo.FindAllが成功しport.PlayerCardRepo.GetPlayerCardsがエラーを返すとき、500になりボディのerrorフィールドはinternal server errorになる",
				cardRepo: &fakeCardRepo{FindAllFn: func(ctx context.Context) ([]*domain.Card, error) {
					return []*domain.Card{newTestCard("TST-0001", "SHE", "Compute", "unlimited")}, nil
				}},
				playerCardRepo: &fakePlayerCardRepo{GetPlayerCardsFn: func(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
					return nil, errors.New("get player cards failed")
				}},
			},
		}
		for _, tc := range errorCases {
			t.Run(tc.name, func(t *testing.T) {
				engine := newTestRouter(t, withCardRepo(tc.cardRepo), withPlayerCardRepo(tc.playerCardRepo))

				rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/cards/with-ownership", nil)

				require.Equal(t, http.StatusInternalServerError, rr.Code)
				assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
			})
		}
	})
}

// findCardWithOwnership locates the response element for cardID, failing the test if absent.
func findCardWithOwnership(t *testing.T, cards []apicard.CardWithOwnership, cardID string) apicard.CardWithOwnership {
	t.Helper()
	for _, c := range cards {
		if c.CardID == cardID {
			return c
		}
	}
	t.Fatalf("card %s not found in response", cardID)
	return apicard.CardWithOwnership{}
}
