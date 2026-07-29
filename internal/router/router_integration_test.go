//go:build integration

package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/handler/rest"
	"github.com/kenyamaneko/overload-party-card/internal/repository"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"

	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
)

const (
	routerTestPlayerWithCards = "11111111-1111-1111-1111-111111111111"
	routerTestPlayerNoCards   = "22222222-2222-2222-2222-222222222222"
)

// productWithInitiativesResponse は ProductHandler.ListAll のレスポンス 1 件分を
// unmarshal するための型。rest.productWithInitiatives は非公開のため router 側で
// 同一の JSON 形状を再定義する。
type productWithInitiativesResponse struct {
	domain.Product
	Initiatives []domain.Initiative `json:"initiatives"`
}

// newIntegrationRouter は実 interactor・実リポジトリ・実キャッシュで結線した router を返す。
// デッキ関連 handler の結線には fakeFactionClient を渡し、account への外部呼び出しを避ける。
func newIntegrationRouter(t *testing.T, playerID string) *gin.Engine {
	t.Helper()
	ctx := context.Background()

	cardRepo := repository.NewPgCardRepository(sharedPg.Pool)
	playerCardRepo := repository.NewPgPlayerCardRepository(sharedPg.Pool)
	productRepo := repository.NewPgProductRepository(sharedPg.Pool)
	initiativeRepo := repository.NewPgInitiativeRepository(sharedPg.Pool)
	deckRepo := repository.NewPgDeckRepository(sharedPg.Pool)

	cardCache := cache.NewCardCache()
	require.NoError(t, cardCache.Load(ctx, cardRepo))

	productCache := cache.NewProductCache()
	require.NoError(t, productCache.Load(ctx, productRepo))

	initiativeCache := cache.NewInitiativeCache()
	require.NoError(t, initiativeCache.Load(ctx, initiativeRepo))

	cardInteractor := usecase.NewCardInteractor(cardRepo, playerCardRepo)
	deckInteractor := usecase.NewDeckInteractor(deckRepo, playerCardRepo, cardCache, productCache, initiativeCache, fakeFactionClient{})
	playerCardInteractor := usecase.NewPlayerCardInteractor(playerCardRepo, cardCache)

	cardH := rest.NewCardHandler(cardInteractor)
	deckH := rest.NewDeckHandler(deckInteractor)
	playerCardH := rest.NewPlayerCardHandler(playerCardInteractor)
	productH := rest.NewProductHandler(productCache, initiativeCache)
	initiativeH := rest.NewInitiativeHandler(initiativeCache)
	pubsubPushH := rest.NewPubSubPushHandler(stubMessageHandler, stubMessageHandler)

	return New(cardH, deckH, playerCardH, productH, initiativeH, pubsubPushH, fakeRouterVerifier{playerID: playerID})
}

// doAuthedGet は internalauth ヘッダを付与して GET を実行する。
func doAuthedGet(r *gin.Engine, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set(internalauth.HeaderName, "any.token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// seedFillerProductAndInitiative は router 構築 (cache.Load) が要求する
// ProductCache / InitiativeCache の非ゼロ件ロードを満たすための、テストの検証対象
// ではないプロダクト・施策を投入する。
func seedFillerProductAndInitiative(t *testing.T) {
	t.Helper()
	seedProduct(t, productSeed{ProductID: "PD-TST0", Faction: "SHE", ProductName: "Filler Product"})
	seedInitiative(t, initiativeSeed{InitiativeID: "IN-TST-R0", ProductID: "PD-TST0", Kind: "routine", Name: "Filler Routine", InsightCost: 0, Effect: `{"ops":[]}`})
}

func TestNewMasterAndOwnershipRoutesReachRealHandler(t *testing.T) {
	t.Run("プロダクト一覧配信", func(t *testing.T) {
		t.Run("プロダクト2件のうち片方だけに施策2件が紐付くとき、施策を持つプロダクトには自身の施策だけが合成されて返り、施策の無いプロダクトは施策が空で返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCard(t, cardSeed{CardID: "TST-0000", CardName: "Filler", Faction: "SHE", CardType: "Resource", Restriction: "unlimited", IsActive: true})
			seedProduct(t, productSeed{ProductID: "PD-TST1", Faction: "SHE", ProductName: "P1"})
			seedProduct(t, productSeed{ProductID: "PD-TST2", Faction: "Tenki", ProductName: "P2"})
			seedInitiative(t, initiativeSeed{InitiativeID: "IN-TST-R1", ProductID: "PD-TST1", Kind: "routine", Name: "R1", InsightCost: 100, Effect: `{"ops":[]}`})
			seedInitiative(t, initiativeSeed{InitiativeID: "IN-TST-S1", ProductID: "PD-TST1", Kind: "special", Name: "S1", InsightCost: 200, Effect: `{"ops":[]}`})

			r := newIntegrationRouter(t, routerTestPlayerWithCards)
			w := doAuthedGet(r, "/api/v1/cards/products")

			require.Equal(t, http.StatusOK, w.Code)
			var got []productWithInitiativesResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
			require.Len(t, got, 2)

			initiativeIDsByProduct := make(map[string][]string, len(got))
			for _, p := range got {
				ids := make([]string, 0, len(p.Initiatives))
				for _, i := range p.Initiatives {
					ids = append(ids, i.InitiativeID)
				}
				initiativeIDsByProduct[p.ProductID] = ids
			}
			assert.Equal(t, map[string][]string{
				"PD-TST1": {"IN-TST-R1", "IN-TST-S1"},
				"PD-TST2": {},
			}, initiativeIDsByProduct)
		})
	})

	t.Run("施策一覧の内部配信", func(t *testing.T) {
		t.Run("施策を投入した状態で取得すると、認証なしで200と全施策が返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCard(t, cardSeed{CardID: "TST-0000", CardName: "Filler", Faction: "SHE", CardType: "Resource", Restriction: "unlimited", IsActive: true})
			seedProduct(t, productSeed{ProductID: "PD-TST1", Faction: "SHE", ProductName: "P1"})
			seedInitiative(t, initiativeSeed{InitiativeID: "IN-TST-R1", ProductID: "PD-TST1", Kind: "routine", Name: "R1", InsightCost: 100, Effect: `{"ops":[]}`})
			seedInitiative(t, initiativeSeed{InitiativeID: "IN-TST-S1", ProductID: "PD-TST1", Kind: "special", Name: "S1", InsightCost: 200, Effect: `{"ops":[]}`})

			r := newIntegrationRouter(t, routerTestPlayerWithCards)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/internal/v1/initiatives", nil))

			require.Equal(t, http.StatusOK, w.Code)
			var got []domain.Initiative
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))

			var gotIDs []string
			for _, i := range got {
				gotIDs = append(gotIDs, i.InitiativeID)
			}
			assert.ElementsMatch(t, []string{"IN-TST-R1", "IN-TST-S1"}, gotIDs)
		})
	})

	t.Run("所持状態付きカード一覧", func(t *testing.T) {
		t.Run("3種のカードのうち1種を所持するプレイヤーが取得すると、全カードが返り所持カードだけ所持済み印になる", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedFillerProductAndInitiative(t)
			seedCard(t, cardSeed{CardID: "TST-0001", CardName: "C1", Faction: "SHE", CardType: "Resource", Restriction: "unlimited", IsActive: true})
			seedCard(t, cardSeed{CardID: "TST-0002", CardName: "C2", Faction: "SHE", CardType: "Resource", Restriction: "unlimited", IsActive: true})
			seedCard(t, cardSeed{CardID: "TST-0003", CardName: "C3", Faction: "SHE", CardType: "Resource", Restriction: "unlimited", IsActive: true})
			seedPlayerCard(t, playerCardSeed{PlayerID: routerTestPlayerWithCards, CardID: "TST-0001", ArtNo: 0, Count: 1})

			r := newIntegrationRouter(t, routerTestPlayerWithCards)
			w := doAuthedGet(r, "/api/v1/cards/cards/with-ownership")

			require.Equal(t, http.StatusOK, w.Code)
			var got []apicard.CardWithOwnership
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
			require.Len(t, got, 3)

			isOwnedByID := make(map[string]bool, len(got))
			for _, c := range got {
				isOwnedByID[c.CardID] = c.IsOwned
			}
			assert.Equal(t, map[string]bool{
				"TST-0001": true,
				"TST-0002": false,
				"TST-0003": false,
			}, isOwnedByID)
		})

		t.Run("1枚も所持しないプレイヤーが取得すると、全カードが未所持で返る", func(t *testing.T) {
			sharedPg.Truncate(t)
			seedCard(t, cardSeed{CardID: "TST-0001", CardName: "C1", Faction: "SHE", CardType: "Resource", Restriction: "unlimited", IsActive: true})
			seedCard(t, cardSeed{CardID: "TST-0002", CardName: "C2", Faction: "SHE", CardType: "Resource", Restriction: "unlimited", IsActive: true})
			seedFillerProductAndInitiative(t)

			r := newIntegrationRouter(t, routerTestPlayerNoCards)
			w := doAuthedGet(r, "/api/v1/cards/cards/with-ownership")

			require.Equal(t, http.StatusOK, w.Code)
			var got []apicard.CardWithOwnership
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
			require.Len(t, got, 2)
			for _, c := range got {
				assert.False(t, c.IsOwned)
			}
		})
	})
}
