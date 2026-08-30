package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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

// testPlayerID is the player_id the default fake verifier reports for authenticated requests.
const testPlayerID = "TST-PLAYER-0001"

// fakeCardRepo is a port.CardRepo test double whose behavior each case sets via FindAllFn.
type fakeCardRepo struct {
	FindAllFn func(ctx context.Context) ([]*domain.Card, error)
}

func (f *fakeCardRepo) FindAll(ctx context.Context) ([]*domain.Card, error) {
	if f.FindAllFn == nil {
		panic("fakeCardRepo.FindAll called without FindAllFn")
	}
	return f.FindAllFn(ctx)
}

var _ port.CardRepo = (*fakeCardRepo)(nil)

// fakePlayerCardRepo is a port.PlayerCardRepo test double whose behavior each case sets via GetPlayerCardsFn.
type fakePlayerCardRepo struct {
	GetPlayerCardsFn func(ctx context.Context, playerID string) ([]*domain.PlayerCard, error)
}

func (f *fakePlayerCardRepo) GetPlayerCards(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
	if f.GetPlayerCardsFn == nil {
		panic("fakePlayerCardRepo.GetPlayerCards called without GetPlayerCardsFn")
	}
	return f.GetPlayerCardsFn(ctx, playerID)
}

func (f *fakePlayerCardRepo) AddCards(ctx context.Context, playerID string, cards []domain.CardPackCard) (int, error) {
	panic("fakePlayerCardRepo.AddCards is outside the handler/router spec scope")
}

var _ port.PlayerCardRepo = (*fakePlayerCardRepo)(nil)

// fakeDeckRepo is a port.DeckRepo test double whose behavior each case sets per method.
type fakeDeckRepo struct {
	CreateFn         func(ctx context.Context, deck domain.Deck, entries []domain.DeckCardEntry) (int64, error)
	FindByPlayerIDFn func(ctx context.Context, playerID string) ([]*domain.Deck, error)
	FindByIDFn       func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error)
	GetDeckCardsFn   func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error)
	UpdateFn         func(ctx context.Context, deck domain.Deck, entries []domain.DeckCardEntry) error
	DeleteFn         func(ctx context.Context, playerID string, deckID int64) error
}

func (f *fakeDeckRepo) Create(ctx context.Context, deck domain.Deck, entries []domain.DeckCardEntry) (int64, error) {
	if f.CreateFn == nil {
		panic("fakeDeckRepo.Create called without CreateFn")
	}
	return f.CreateFn(ctx, deck, entries)
}

func (f *fakeDeckRepo) FindByPlayerID(ctx context.Context, playerID string) ([]*domain.Deck, error) {
	if f.FindByPlayerIDFn == nil {
		panic("fakeDeckRepo.FindByPlayerID called without FindByPlayerIDFn")
	}
	return f.FindByPlayerIDFn(ctx, playerID)
}

func (f *fakeDeckRepo) FindByID(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
	if f.FindByIDFn == nil {
		panic("fakeDeckRepo.FindByID called without FindByIDFn")
	}
	return f.FindByIDFn(ctx, playerID, deckID)
}

func (f *fakeDeckRepo) GetDeckCards(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
	if f.GetDeckCardsFn == nil {
		panic("fakeDeckRepo.GetDeckCards called without GetDeckCardsFn")
	}
	return f.GetDeckCardsFn(ctx, playerID, deckID)
}

func (f *fakeDeckRepo) Update(ctx context.Context, deck domain.Deck, entries []domain.DeckCardEntry) error {
	if f.UpdateFn == nil {
		panic("fakeDeckRepo.Update called without UpdateFn")
	}
	return f.UpdateFn(ctx, deck, entries)
}

func (f *fakeDeckRepo) Delete(ctx context.Context, playerID string, deckID int64) error {
	if f.DeleteFn == nil {
		panic("fakeDeckRepo.Delete called without DeleteFn")
	}
	return f.DeleteFn(ctx, playerID, deckID)
}

var _ port.DeckRepo = (*fakeDeckRepo)(nil)

// fakeFactionClient is a port.FactionClient test double whose behavior each case sets via ListPlayerFactionsFn.
type fakeFactionClient struct {
	ListPlayerFactionsFn func(ctx context.Context, playerID string) ([]string, error)
}

func (f *fakeFactionClient) ListPlayerFactions(ctx context.Context, playerID string) ([]string, error) {
	if f.ListPlayerFactionsFn == nil {
		panic("fakeFactionClient.ListPlayerFactions called without ListPlayerFactionsFn")
	}
	return f.ListPlayerFactionsFn(ctx, playerID)
}

var _ port.FactionClient = (*fakeFactionClient)(nil)

// testRouterConfig holds the fakes injected into router.New for one test case.
type testRouterConfig struct {
	cardRepo                 port.CardRepo
	playerCardRepo           port.PlayerCardRepo
	deckRepo                 port.DeckRepo
	factionClient            port.FactionClient
	cardCache                *cache.CardCache
	productCache             *cache.ProductCache
	initiativeCache          *cache.InitiativeCache
	verifier                 internalauth.Verifier
	playerOnboardedHandler   port.MessageHandler
	cardPackPurchasedHandler port.MessageHandler
}

type testRouterOption func(*testRouterConfig)

func withCardRepo(r port.CardRepo) testRouterOption {
	return func(c *testRouterConfig) { c.cardRepo = r }
}

func withPlayerCardRepo(r port.PlayerCardRepo) testRouterOption {
	return func(c *testRouterConfig) { c.playerCardRepo = r }
}

func withDeckRepo(r port.DeckRepo) testRouterOption {
	return func(c *testRouterConfig) { c.deckRepo = r }
}

func withFactionClient(fc port.FactionClient) testRouterOption {
	return func(c *testRouterConfig) { c.factionClient = fc }
}

func withCardCache(cc *cache.CardCache) testRouterOption {
	return func(c *testRouterConfig) { c.cardCache = cc }
}

func withProductCache(pc *cache.ProductCache) testRouterOption {
	return func(c *testRouterConfig) { c.productCache = pc }
}

func withInitiativeCache(ic *cache.InitiativeCache) testRouterOption {
	return func(c *testRouterConfig) { c.initiativeCache = ic }
}

func withPlayerOnboardedHandler(h port.MessageHandler) testRouterOption {
	return func(c *testRouterConfig) { c.playerOnboardedHandler = h }
}

func withCardPackPurchasedHandler(h port.MessageHandler) testRouterOption {
	return func(c *testRouterConfig) { c.cardPackPurchasedHandler = h }
}

// newTestRouter builds a router.New engine wired to fake port implementations. Unset fakes
// default to panicking on use so an unconfigured case fails loudly instead of returning a
// silent zero value; options override the fakes each case actually needs.
func newTestRouter(t *testing.T, opts ...testRouterOption) *gin.Engine {
	t.Helper()

	cfg := &testRouterConfig{
		cardRepo:        &fakeCardRepo{},
		playerCardRepo:  &fakePlayerCardRepo{},
		deckRepo:        &fakeDeckRepo{},
		factionClient:   &fakeFactionClient{},
		cardCache:       cache.NewCardCache(),
		productCache:    cache.NewProductCache(),
		initiativeCache: cache.NewInitiativeCache(),
		verifier: &internalauth.MockVerifier{
			VerifyFn: func(string) (string, error) { return testPlayerID, nil },
		},
		playerOnboardedHandler: func(ctx context.Context, data []byte) error {
			panic("playerOnboardedHandler called without withPlayerOnboardedHandler")
		},
		cardPackPurchasedHandler: func(ctx context.Context, data []byte) error {
			panic("cardPackPurchasedHandler called without withCardPackPurchasedHandler")
		},
	}
	for _, opt := range opts {
		opt(cfg)
	}

	cardInteractor := usecase.NewCardInteractor(cfg.cardRepo, cfg.playerCardRepo)
	deckInteractor := usecase.NewDeckInteractor(cfg.deckRepo, cfg.playerCardRepo, cfg.cardCache, cfg.productCache, cfg.initiativeCache, cfg.factionClient)
	playerCardInteractor := usecase.NewPlayerCardInteractor(cfg.playerCardRepo, cfg.cardCache)

	cardH := rest.NewCardHandler(cardInteractor)
	deckH := rest.NewDeckHandler(deckInteractor)
	playerCardH := rest.NewPlayerCardHandler(playerCardInteractor)
	productH := rest.NewProductHandler(cfg.productCache, cfg.initiativeCache)
	initiativeH := rest.NewInitiativeHandler(cfg.initiativeCache)
	pubsubPushH := rest.NewPubSubPushHandler(cfg.playerOnboardedHandler, cfg.cardPackPurchasedHandler)

	return router.New(cardH, deckH, playerCardH, productH, initiativeH, pubsubPushH, cfg.verifier)
}

// newJSONRequest builds an httptest request. body may be nil (no body), a string/[]byte (sent
// verbatim, for malformed-payload cases) or any other value (JSON-encoded).
func newJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	var reader *bytes.Reader
	switch v := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case []byte:
		reader = bytes.NewReader(v)
	case string:
		reader = bytes.NewReader([]byte(v))
	default:
		encoded, err := json.Marshal(v)
		require.NoError(t, err)
		reader = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// doAuthedRequest issues a request through engine with a valid X-Internal-Auth header present.
func doAuthedRequest(t *testing.T, engine *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	req := newJSONRequest(t, method, path, body)
	req.Header.Set(internalauth.HeaderName, "any-token")
	rr := httptest.NewRecorder()
	engine.ServeHTTP(rr, req)
	return rr
}

// doRequest issues a request through engine without any X-Internal-Auth header, for the
// unauthenticated /internal/v1 endpoints.
func doRequest(t *testing.T, engine *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	req := newJSONRequest(t, method, path, body)
	rr := httptest.NewRecorder()
	engine.ServeHTTP(rr, req)
	return rr
}

// decodeJSON unmarshals the response body into out.
func decodeJSON(t *testing.T, rr *httptest.ResponseRecorder, out any) {
	t.Helper()
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), out))
}

// errorBody is the shape of every error response body ({"error": "..."}).
type errorBody struct {
	Error string `json:"error"`
}

// decodeErrorMessage decodes the response body's error field.
func decodeErrorMessage(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	var body errorBody
	decodeJSON(t, rr, &body)
	return body.Error
}

// newTestCardCache builds a cache.CardCache preloaded with the given cards via InjectForTest.
func newTestCardCache(cards ...*domain.Card) *cache.CardCache {
	c := cache.NewCardCache()
	for _, card := range cards {
		c.InjectForTest(card.CardID, card)
	}
	return c
}

// newTestCard builds a domain.Card fixture with the given identity/classification fields and
// otherwise-realistic dummy content.
func newTestCard(cardID, faction, cardType, restriction string) *domain.Card {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return &domain.Card{
		CardID:        cardID,
		CardName:      "テストカード " + cardID,
		ResourceLabel: "TP",
		Faction:       faction,
		CardType:      cardType,
		Resizable:     false,
		Elastic:       false,
		Stats:         json.RawMessage(`{"availability":1,"maintenance_cost":1,"sla_penalty":1,"throughput":1}`),
		Restriction:   restriction,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
