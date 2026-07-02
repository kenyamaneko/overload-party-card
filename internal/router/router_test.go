package router

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/handler/rest"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"

	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// fakeRouterVerifier は router 単体テスト用の internalauth.Verifier 最小 fake。
type fakeRouterVerifier struct {
	playerID string
	err      error
}

func (f fakeRouterVerifier) Verify(string) (string, error) {
	return f.playerID, f.err
}

// stubCardRepo は固定のカードマスター 1 件を返す port.CardRepo スタブ。
type stubCardRepo struct{}

// FindAll は TST ダミーカード 1 件を返す。
func (stubCardRepo) FindAll(context.Context) ([]*domain.Card, error) {
	return []*domain.Card{{CardID: "TST-0001"}}, nil
}

// stubPlayerCardRepo は所持カードなしを返す port.PlayerCardRepo スタブ。
type stubPlayerCardRepo struct{}

// GetPlayerCards は所持カードなしを返す。
func (stubPlayerCardRepo) GetPlayerCards(context.Context, string) ([]*domain.PlayerCard, error) {
	return []*domain.PlayerCard{}, nil
}

// AddCards は加算 0 件で成功する。
func (stubPlayerCardRepo) AddCards(context.Context, string, []domain.CardPackCard) (int, error) {
	return 0, nil
}

// stubDeckRepo はデッキなしを返す port.DeckRepo スタブ。
type stubDeckRepo struct{}

// Create は固定の deck_id で成功する。
func (stubDeckRepo) Create(context.Context, domain.Deck, []domain.DeckCardEntry) (int64, error) {
	return 1, nil
}

// FindByPlayerID はデッキなしを返す。
func (stubDeckRepo) FindByPlayerID(context.Context, string) ([]*domain.Deck, error) {
	return []*domain.Deck{}, nil
}

// FindByID は空のデッキを返す。
func (stubDeckRepo) FindByID(context.Context, string, int64) (*domain.Deck, error) {
	return &domain.Deck{}, nil
}

// GetDeckCards はデッキカードなしを返す。
func (stubDeckRepo) GetDeckCards(context.Context, string, int64) ([]domain.DeckCard, error) {
	return []domain.DeckCard{}, nil
}

// Update は常に成功する。
func (stubDeckRepo) Update(context.Context, domain.Deck, []domain.DeckCardEntry) error {
	return nil
}

// Delete は常に成功する。
func (stubDeckRepo) Delete(context.Context, string, int64) error {
	return nil
}

// stubFactionClient は固定の所持ファクションを返す port.FactionClient スタブ。
type stubFactionClient struct{}

// ListPlayerFactions は TST ダミーファクション 1 件を返す。
func (stubFactionClient) ListPlayerFactions(context.Context, string) ([]string, error) {
	return []string{"TST-FACTION"}, nil
}

// stubInitiativeJSON は InitiativeCache へロードする施策フィクスチャ (空だとロードがエラーになる)。
const stubInitiativeJSON = `[{"initiative_id":"IN-TST-0001","product_id":"PR-TST-0001",` +
	`"kind":"TST-KIND","name":"TST initiative","insight_cost":1,"effect_text":"",` +
	`"effect":null,"is_active":true}]`

// newTestRouter はスタブ repository で組んだ実 interactor / handler で router を構築する。
func newTestRouter(t *testing.T, verifier internalauth.Verifier) *gin.Engine {
	t.Helper()

	cardCache := cache.NewCardCache()
	productCache := cache.NewProductCache()
	initiativeCache := cache.NewInitiativeCache()
	require.NoError(t, initiativeCache.LoadFromBytes([]byte(stubInitiativeJSON)))

	cardI := usecase.NewCardInteractor(stubCardRepo{}, stubPlayerCardRepo{})
	deckI := usecase.NewDeckInteractor(
		stubDeckRepo{}, stubPlayerCardRepo{}, cardCache, productCache, initiativeCache, stubFactionClient{},
	)
	playerCardI := usecase.NewPlayerCardInteractor(stubPlayerCardRepo{}, cardCache)

	return New(
		rest.NewCardHandler(cardI),
		rest.NewDeckHandler(deckI),
		rest.NewPlayerCardHandler(playerCardI),
		rest.NewProductHandler(productCache, initiativeCache),
		rest.NewInitiativeHandler(initiativeCache),
		verifier,
	)
}

// /health は auth middleware を通らず常に 200 を返す。
func TestNew_HealthEndpoint(t *testing.T) {
	r := newTestRouter(t, fakeRouterVerifier{})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestNew_InternalRoutesAreAuthFree は /internal/v1 配下が auth header なしで
// handler の成功応答まで到達することを確かめる。
func TestNew_InternalRoutesAreAuthFree(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{name: "/internal/v1/cards は auth-free でカードマスターを返し 200", path: "/internal/v1/cards"},
		{name: "/internal/v1/initiatives は auth-free で施策マスターを返し 200", path: "/internal/v1/initiatives"},
	}

	r := newTestRouter(t, fakeRouterVerifier{})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// /api/v1/cards 配下は auth middleware を通る。
// header 欠落時は 401 を返し、handler に到達しないことを確認する。
func TestNew_ApiRouteRequiresInternalAuth(t *testing.T) {
	r := newTestRouter(t, fakeRouterVerifier{playerID: "TST-PLAYER-1"})

	cases := []struct {
		name string
		path string
	}{
		{name: "/api/v1/cards/decks は auth header 欠落で 401", path: "/api/v1/cards/decks"},
		{name: "/api/v1/cards/cards は auth header 欠落で 401", path: "/api/v1/cards/cards"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// verifier が error を返すと 401 を返し handler に到達しない。
func TestNew_ApiRouteRejectsVerifierError(t *testing.T) {
	r := newTestRouter(t, fakeRouterVerifier{err: errors.New("invalid token")})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cards/decks", nil)
	req.Header.Set(internalauth.HeaderName, "any.token")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestNew_ApiRouteWithValidTokenReachesHandler は verifier を通過したリクエストが
// handler の成功応答まで到達することを確かめる。
func TestNew_ApiRouteWithValidTokenReachesHandler(t *testing.T) {
	r := newTestRouter(t, fakeRouterVerifier{playerID: "TST-PLAYER-1"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cards/cards", nil)
	req.Header.Set(internalauth.HeaderName, "any.token")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
