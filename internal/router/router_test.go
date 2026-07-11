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

func TestNew(t *testing.T) {
	t.Run("ルーターの認証配線", func(t *testing.T) {
		t.Run("/health は auth middleware を通らず 200 を返す", func(t *testing.T) {
			// VerifyFn 未設定: /health が verifier に到達しないことの検出を兼ねる
			r := newTestRouter(t, &internalauth.MockVerifier{})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("/internal/v1 配下は auth header なしで handler の成功応答まで到達する", func(t *testing.T) {
			// VerifyFn 未設定: auth-free ルートが verifier に到達しないことの検出を兼ねる
			r := newTestRouter(t, &internalauth.MockVerifier{})

			cases := []struct {
				name string
				path string
			}{
				{name: "/internal/v1/cards は 200 でカードマスターを返す", path: "/internal/v1/cards"},
				{name: "/internal/v1/initiatives は 200 で施策マスターを返す", path: "/internal/v1/initiatives"},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					w := httptest.NewRecorder()
					r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
					assert.Equal(t, http.StatusOK, w.Code)
				})
			}
		})

		t.Run("/api/v1/cards 配下は auth header 欠落で 401 になる", func(t *testing.T) {
			// VerifyFn 未設定: header 欠落時は middleware が verifier に到達しないことの検出を兼ねる
			r := newTestRouter(t, &internalauth.MockVerifier{})

			cases := []struct {
				name string
				path string
			}{
				{name: "/api/v1/cards/decks は 401 になる", path: "/api/v1/cards/decks"},
				{name: "/api/v1/cards/cards は 401 になる", path: "/api/v1/cards/cards"},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					w := httptest.NewRecorder()
					r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
					assert.Equal(t, http.StatusUnauthorized, w.Code)
				})
			}
		})

		t.Run("verifier がエラーを返すとき、401 になる", func(t *testing.T) {
			r := newTestRouter(t, &internalauth.MockVerifier{
				VerifyFn: func(string) (string, error) { return "", errors.New("invalid token") },
			})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/cards/decks", nil)
			req.Header.Set(internalauth.HeaderName, "any.token")
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("verifier を通過したとき、handler の成功応答まで到達する", func(t *testing.T) {
			r := newTestRouter(t, &internalauth.MockVerifier{
				VerifyFn: func(string) (string, error) { return "TST-PLAYER-1", nil },
			})
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/cards/cards", nil)
			req.Header.Set(internalauth.HeaderName, "any.token")
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	})
}
