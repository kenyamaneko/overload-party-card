package router_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/handler/rest"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/router"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"

	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// stubCardRepo is a port.CardRepo test double; FindAll uses findAllFn if set, else panics.
type stubCardRepo struct {
	findAllFn func(ctx context.Context) ([]*domain.Card, error)
}

func (s *stubCardRepo) FindAll(ctx context.Context) ([]*domain.Card, error) {
	if s.findAllFn == nil {
		panic("stubCardRepo.FindAll called unexpectedly")
	}
	return s.findAllFn(ctx)
}

// stubPlayerCardRepo is a port.PlayerCardRepo test double that panics on use; router-level
// tests never exercise a code path that reaches it.
type stubPlayerCardRepo struct{}

func (s *stubPlayerCardRepo) GetPlayerCards(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
	panic("stubPlayerCardRepo.GetPlayerCards called unexpectedly")
}

func (s *stubPlayerCardRepo) AddCards(ctx context.Context, playerID string, cards []domain.CardPackCard) (int, error) {
	panic("stubPlayerCardRepo.AddCards called unexpectedly")
}

// stubDeckRepo is a port.DeckRepo test double that panics on use.
type stubDeckRepo struct{}

func (s *stubDeckRepo) Create(ctx context.Context, deck domain.Deck, entries []domain.DeckCardEntry) (int64, error) {
	panic("stubDeckRepo.Create called unexpectedly")
}

func (s *stubDeckRepo) FindByPlayerID(ctx context.Context, playerID string) ([]*domain.Deck, error) {
	panic("stubDeckRepo.FindByPlayerID called unexpectedly")
}

func (s *stubDeckRepo) FindByID(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
	panic("stubDeckRepo.FindByID called unexpectedly")
}

func (s *stubDeckRepo) GetDeckCards(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
	panic("stubDeckRepo.GetDeckCards called unexpectedly")
}

func (s *stubDeckRepo) Update(ctx context.Context, deck domain.Deck, entries []domain.DeckCardEntry) error {
	panic("stubDeckRepo.Update called unexpectedly")
}

func (s *stubDeckRepo) Delete(ctx context.Context, playerID string, deckID int64) error {
	panic("stubDeckRepo.Delete called unexpectedly")
}

// stubFactionClient is a port.FactionClient test double that panics on use.
type stubFactionClient struct{}

func (s *stubFactionClient) ListPlayerFactions(ctx context.Context, playerID string) ([]string, error) {
	panic("stubFactionClient.ListPlayerFactions called unexpectedly")
}

// newTestEngine builds router.New wired to the given cardRepo / initiativeCache / pubsub
// handlers / verifier, defaulting each to a value the router-level tests never need to exercise
// (nil defaults to a panic-on-use stub, or an empty cache). The other four handlers (deck,
// player-card, product, and their underlying ports) always get panic-on-use stubs: router-level
// tests only assert whether a request reaches *some* handler and whether auth gates it, not any
// per-endpoint response body, so those paths are never exercised here.
func newTestEngine(t *testing.T, cardRepo port.CardRepo, initiativeCache *cache.InitiativeCache, playerOnboarded, cardPackPurchased port.MessageHandler, verifier internalauth.Verifier) *gin.Engine {
	t.Helper()
	if cardRepo == nil {
		cardRepo = &stubCardRepo{}
	}
	if initiativeCache == nil {
		initiativeCache = cache.NewInitiativeCache()
	}
	if playerOnboarded == nil {
		playerOnboarded = func(ctx context.Context, data []byte) error {
			panic("player-onboarded handler called unexpectedly")
		}
	}
	if cardPackPurchased == nil {
		cardPackPurchased = func(ctx context.Context, data []byte) error {
			panic("card-pack-purchased handler called unexpectedly")
		}
	}

	playerCardRepo := &stubPlayerCardRepo{}
	deckRepo := &stubDeckRepo{}
	factionClient := &stubFactionClient{}
	cardCache := cache.NewCardCache()
	productCache := cache.NewProductCache()

	cardInteractor := usecase.NewCardInteractor(cardRepo, playerCardRepo)
	deckInteractor := usecase.NewDeckInteractor(deckRepo, playerCardRepo, cardCache, productCache, initiativeCache, factionClient)
	playerCardInteractor := usecase.NewPlayerCardInteractor(playerCardRepo, cardCache)

	cardH := rest.NewCardHandler(cardInteractor)
	deckH := rest.NewDeckHandler(deckInteractor)
	playerCardH := rest.NewPlayerCardHandler(playerCardInteractor)
	productH := rest.NewProductHandler(productCache, initiativeCache)
	initiativeH := rest.NewInitiativeHandler(initiativeCache)
	pubsubPushH := rest.NewPubSubPushHandler(playerOnboarded, cardPackPurchased)

	return router.New(cardH, deckH, playerCardH, productH, initiativeH, pubsubPushH, verifier)
}

// pubsubEnvelopeBody builds a valid Cloud Pub/Sub push envelope JSON body carrying payload,
// base64-encoded as the wire format requires.
func pubsubEnvelopeBody(t *testing.T, payload string) string {
	t.Helper()
	envelope := struct {
		Message struct {
			Data string `json:"data"`
		} `json:"message"`
	}{}
	envelope.Message.Data = base64.StdEncoding.EncodeToString([]byte(payload))
	encoded, err := json.Marshal(envelope)
	require.NoError(t, err)
	return string(encoded)
}

func TestRouterHealth(t *testing.T) {
	t.Run("[ルーティング] ヘルスチェック", func(t *testing.T) {
		t.Run("GET /healthは常に200になり、レスポンスボディは{\"status\":\"ok\"}になる", func(t *testing.T) {
			engine := newTestEngine(t, nil, nil, nil, nil, &internalauth.MockVerifier{})

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rr := httptest.NewRecorder()
			engine.ServeHTTP(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)
			assert.JSONEq(t, `{"status":"ok"}`, rr.Body.String())
		})
	})
}

func TestRouterInternalV1NoAuth(t *testing.T) {
	t.Run("[ルーティング] internal/v1配下の認証除外", func(t *testing.T) {
		t.Run("GET /internal/v1/cardsは、X-Internal-Authヘッダが無くてもハンドラが呼び出され200になる", func(t *testing.T) {
			engine := newTestEngine(t, &stubCardRepo{findAllFn: func(ctx context.Context) ([]*domain.Card, error) {
				return nil, nil
			}}, nil, nil, nil, &internalauth.MockVerifier{})

			req := httptest.NewRequest(http.MethodGet, "/internal/v1/cards", nil)
			rr := httptest.NewRecorder()
			engine.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		})

		t.Run("GET /internal/v1/initiativesは、X-Internal-Authヘッダが無くてもハンドラが呼び出され200になる", func(t *testing.T) {
			engine := newTestEngine(t, nil, cache.NewInitiativeCache(), nil, nil, &internalauth.MockVerifier{})

			req := httptest.NewRequest(http.MethodGet, "/internal/v1/initiatives", nil)
			rr := httptest.NewRecorder()
			engine.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		})

		t.Run("POST /internal/v1/pubsub/player-onboardedは、X-Internal-Authヘッダが無くてもハンドラが呼び出され200になる", func(t *testing.T) {
			called := false
			engine := newTestEngine(t, nil, nil, func(ctx context.Context, data []byte) error {
				called = true
				return nil
			}, nil, &internalauth.MockVerifier{})

			req := httptest.NewRequest(http.MethodPost, "/internal/v1/pubsub/player-onboarded", strings.NewReader(pubsubEnvelopeBody(t, "x")))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			engine.ServeHTTP(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)
			assert.True(t, called)
		})

		t.Run("POST /internal/v1/pubsub/card-pack-purchasedは、X-Internal-Authヘッダが無くてもハンドラが呼び出され200になる", func(t *testing.T) {
			called := false
			engine := newTestEngine(t, nil, nil, nil, func(ctx context.Context, data []byte) error {
				called = true
				return nil
			}, &internalauth.MockVerifier{})

			req := httptest.NewRequest(http.MethodPost, "/internal/v1/pubsub/card-pack-purchased", strings.NewReader(pubsubEnvelopeBody(t, "y")))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			engine.ServeHTTP(rr, req)

			require.Equal(t, http.StatusOK, rr.Code)
			assert.True(t, called)
		})
	})
}

func TestRouterAPIV1Auth(t *testing.T) {
	t.Run("[ルーティング] api/v1/cards配下の認証", func(t *testing.T) {
		cases := []struct {
			name      string
			verifier  internalauth.Verifier
			setHeader bool
		}{
			{
				name:      "X-Internal-Authヘッダが無いとき、401になる",
				verifier:  &internalauth.MockVerifier{},
				setHeader: false,
			},
			{
				name: "ヘッダはあるがトークンの検証に失敗するとき、401になる",
				verifier: &internalauth.MockVerifier{VerifyFn: func(string) (string, error) {
					return "", errors.New("invalid token")
				}},
				setHeader: true,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				engine := newTestEngine(t, nil, nil, nil, nil, tc.verifier)

				req := httptest.NewRequest(http.MethodGet, "/api/v1/cards/cards", nil)
				if tc.setHeader {
					req.Header.Set(internalauth.HeaderName, "any-token")
				}
				rr := httptest.NewRecorder()
				engine.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusUnauthorized, rr.Code)
			})
		}
	})
}
