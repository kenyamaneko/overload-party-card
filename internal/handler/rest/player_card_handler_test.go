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

func TestPlayerCardHandlerGetPlayerCards(t *testing.T) {
	t.Run("[所持カードAPI] プレイヤー所持カード一覧取得", func(t *testing.T) {
		t.Run("所持カードそれぞれのcard_idがカード定義キャッシュに存在するとき、200になり所持数量とカード定義が組み合わさって反映される", func(t *testing.T) {
			cardCache := newTestCardCache(newTestCard("TST-0001", "SHE", "Compute", "unlimited"))
			engine := newTestRouter(t,
				withCardCache(cardCache),
				withPlayerCardRepo(&fakePlayerCardRepo{GetPlayerCardsFn: func(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
					return []*domain.PlayerCard{{PlayerID: playerID, CardID: "TST-0001", ArtNo: 2, Count: 3}}, nil
				}}),
			)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/cards", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			var body []apicard.PlayerCardWithDef
			decodeJSON(t, rr, &body)
			require.Len(t, body, 1)
			got := body[0]
			assert.Equal(t, "TST-0001", got.CardID)
			assert.Equal(t, int64(2), got.ArtNo)
			assert.Equal(t, 3, got.Count)
			assert.Equal(t, "テストカード TST-0001", got.CardName)
			assert.Equal(t, "TP", got.ResourceLabel)
			assert.Equal(t, "SHE", got.Faction)
			assert.Equal(t, "Compute", got.CardType)
			assert.Equal(t, "unlimited", got.Restriction)
		})

		t.Run("所持カードが1件も無いとき、200になりレスポンスボディは空配列になる", func(t *testing.T) {
			engine := newTestRouter(t,
				withPlayerCardRepo(&fakePlayerCardRepo{GetPlayerCardsFn: func(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
					return []*domain.PlayerCard{}, nil
				}}),
			)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/cards", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			assert.JSONEq(t, "[]", rr.Body.String())
		})

		t.Run("プレイヤーの所持カード取得に失敗するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			engine := newTestRouter(t,
				withPlayerCardRepo(&fakePlayerCardRepo{GetPlayerCardsFn: func(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
					return nil, errors.New("get player cards failed")
				}}),
			)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/cards", nil)

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("所持カードのcard_idがカード定義キャッシュに存在しないとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			engine := newTestRouter(t,
				withCardCache(newTestCardCache()),
				withPlayerCardRepo(&fakePlayerCardRepo{GetPlayerCardsFn: func(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
					return []*domain.PlayerCard{{PlayerID: playerID, CardID: "TST-9999", ArtNo: 1, Count: 1}}, nil
				}}),
			)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/cards", nil)

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})
	})
}
