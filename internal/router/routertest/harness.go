package routertest

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/adapter/clock"
	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/handler/pubsub"
	"github.com/kenyamaneko/overload-party-card/internal/handler/rest"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/router"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"
)

type fakeProductRepository struct {
	products []*domain.Product
}

func (r fakeProductRepository) FindAll(_ context.Context) ([]*domain.Product, error) {
	return r.products, nil
}

type fakeInitiativeRepository struct {
	initiatives []*domain.Initiative
}

func (r fakeInitiativeRepository) FindAll(_ context.Context) ([]*domain.Initiative, error) {
	return r.initiatives, nil
}

// FakeFactionClient はアカウントサービスへの陣営の所持の照会の偽物です。
type FakeFactionClient struct {
	Factions []string
	Err      error
}

// ListPlayerFactions は設定した陣営の一覧とエラーを返します。
func (c *FakeFactionClient) ListPlayerFactions(_ context.Context, _ string) ([]string, error) {
	return c.Factions, c.Err
}

func verifyTestToken(token string) (string, error) {
	switch token {
	case TokenA:
		return PlayerA, nil
	default:
		return "", errInvalidToken
	}
}

// Dependencies は組み立てるルーターが依存するポートの実装です。
type Dependencies struct {
	Cards             port.CardRepository
	PlayerCards       port.PlayerCardRepository
	Decks             port.DeckRepository
	TransactionRunner port.GrantTransactionRunner
	Factions          port.FactionClient
	Products          port.ProductRepository
	Initiatives       port.InitiativeRepository
}

// NewRouter は依存を読み込んでルーターを組み立てます。
func NewRouter(t *testing.T, deps Dependencies) http.Handler {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	cardCache := cache.NewCardCache()
	require.NoError(t, cardCache.Load(ctx, deps.Cards))
	productCache := cache.NewProductCache()
	require.NoError(t, productCache.Load(ctx, deps.Products))
	initiativeCache := cache.NewInitiativeCache()
	require.NoError(t, initiativeCache.Load(ctx, deps.Initiatives))

	grantInteractor := usecase.NewGrantInteractor(deps.TransactionRunner)
	return router.New(
		rest.NewCardHandler(usecase.NewCardInteractor(deps.Cards, deps.PlayerCards)),
		rest.NewDeckHandler(usecase.NewDeckInteractor(deps.Decks, deps.PlayerCards, cardCache, productCache, initiativeCache, deps.Factions, clock.NewSystemClock())),
		rest.NewPlayerCardHandler(usecase.NewPlayerCardInteractor(deps.PlayerCards, cardCache)),
		rest.NewProductHandler(usecase.NewProductInteractor(productCache, initiativeCache)),
		rest.NewPubSubPushHandler(pubsub.NewPlayerOnboardedSubscriber(grantInteractor), pubsub.NewCardPackPurchasedSubscriber(grantInteractor)),
		&internalauth.MockVerifier{VerifyFn: verifyTestToken},
	)
}

// Harness は組み立てたルーターと、そのケース専用の偽物とログの捕捉先です。
type Harness struct {
	DB       *FakeDatabase
	Factions *FakeFactionClient
	Handler  http.Handler
	Logs     *bytes.Buffer
}

// NewHarness は既定のマスターデータでハーネスを組み立てます。
func NewHarness(t *testing.T) *Harness {
	return NewHarnessWithMasterData(t, NewMasterData())
}

// NewHarnessWithMasterData は指定のマスターデータでハーネスを組み立てます。
func NewHarnessWithMasterData(t *testing.T, masterData MasterData) *Harness {
	logs := &bytes.Buffer{}
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	db := NewFakeDatabase()
	db.Cards = masterData.Cards
	db.CardPacks = masterData.CardPacks
	factions := &FakeFactionClient{Factions: []string{"SHE"}}
	return &Harness{
		DB:       db,
		Factions: factions,
		Handler: NewRouter(t, Dependencies{
			Cards:             db,
			PlayerCards:       db,
			Decks:             db,
			TransactionRunner: db,
			Factions:          factions,
			Products:          fakeProductRepository{products: masterData.Products},
			Initiatives:       fakeInitiativeRepository{initiatives: masterData.Initiatives},
		}),
		Logs: logs,
	}
}

// NewHarnessOwningValidDeckCards はプレイヤー A が 30 枚のデッキの構成カードをすべて所持するハーネスを組み立てます。
func NewHarnessOwningValidDeckCards(t *testing.T) *Harness {
	h := NewHarness(t)
	h.DB.SetOwnedComposition(PlayerA, ValidDeckComposition)
	return h
}

// NewHarnessWithStoredDeck はプレイヤー A の保存済みデッキ ID7 (TST-0001 のアート 2 の 3 枚) を持つハーネスを組み立てます。
func NewHarnessWithStoredDeck(t *testing.T) *Harness {
	h := NewHarness(t)
	h.StoreDeckWithCard(7, CardCount{CardID: "TST-0001", ArtNo: 2, Count: 3})
	return h
}

// StoreValidDeck はプレイヤー A の 30 枚のデッキを保存し、構成カードをすべて所持させます。
func (h *Harness) StoreValidDeck(deckID int64) {
	h.DB.StoreDeck(NewStoredDeck(deckID), BuildDeckCards(PlayerA, deckID, ValidDeckComposition))
	h.DB.SetOwnedComposition(PlayerA, ValidDeckComposition)
}

// StoreDeckWithCard はプレイヤー A の 1 種類のカードだけのデッキを保存します。
func (h *Harness) StoreDeckWithCard(deckID int64, entry CardCount) {
	h.DB.StoreDeck(NewStoredDeck(deckID), BuildDeckCards(PlayerA, deckID, []CardCount{entry}))
}

// ServeHTTPRequest は要求をルーターに渡して応答を記録します。
func ServeHTTPRequest(handler http.Handler, request *http.Request) *httptest.ResponseRecorder {
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

// SendRequestWithToken は X-Internal-Auth ヘッダにトークンを付けた要求を送ります。
func (h *Harness) SendRequestWithToken(method, path, token, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set(internalauth.HeaderName, token)
	return ServeHTTPRequest(h.Handler, request)
}

// SendRequestWithoutToken は X-Internal-Auth ヘッダを付けない要求を送ります。
func (h *Harness) SendRequestWithoutToken(method, path, body string) *httptest.ResponseRecorder {
	return ServeHTTPRequest(h.Handler, httptest.NewRequest(method, path, strings.NewReader(body)))
}

// SendGetRequest はプレイヤー A のトークンで GET 要求を送ります。
func (h *Harness) SendGetRequest(path string) *httptest.ResponseRecorder {
	return h.SendRequestWithToken(http.MethodGet, path, TokenA, "")
}

// BuildDeckPath はデッキ ID を含むデッキの経路を組み立てます。
func BuildDeckPath(deckID int64) string {
	return DecksPath + "/" + strconv.FormatInt(deckID, 10)
}

// GetDecks はプレイヤー A のデッキ一覧を取得します。
func (h *Harness) GetDecks() *httptest.ResponseRecorder {
	return h.SendGetRequest(DecksPath)
}

// GetDeck はプレイヤー A のデッキ詳細を取得します。
func (h *Harness) GetDeck(deckID int64) *httptest.ResponseRecorder {
	return h.SendGetRequest(BuildDeckPath(deckID))
}

// GetPlayerCards はプレイヤー A の所持カード一覧を取得します。
func (h *Harness) GetPlayerCards() *httptest.ResponseRecorder {
	return h.SendGetRequest(PlayerCardsPath)
}

// CreateDeck はプレイヤー A としてデッキを作成します。
func (h *Harness) CreateDeck(t *testing.T, body DeckRequestBody) *httptest.ResponseRecorder {
	return h.SendRequestWithToken(http.MethodPost, DecksPath, TokenA, EncodeJSON(t, body))
}

// UpdateDeck はプレイヤー A としてデッキを更新します。
func (h *Harness) UpdateDeck(t *testing.T, deckID int64, body DeckRequestBody) *httptest.ResponseRecorder {
	return h.SendRequestWithToken(http.MethodPut, BuildDeckPath(deckID), TokenA, EncodeJSON(t, body))
}

// DeleteDeck はプレイヤー A としてデッキの経路に削除要求を送ります。
func (h *Harness) DeleteDeck(path string) *httptest.ResponseRecorder {
	return h.SendRequestWithToken(http.MethodDelete, path, TokenA, "")
}

// ValidateDeckForBattle はプレイヤー A としてデッキの経路にバトル使用検証の要求を送ります。
func (h *Harness) ValidateDeckForBattle(path string) *httptest.ResponseRecorder {
	return h.SendRequestWithToken(http.MethodPost, path+"/validate-for-battle", TokenA, "")
}

// SendPlayerOnboarded はプレイヤー登録完了イベントの受信へ本文を送ります。
func (h *Harness) SendPlayerOnboarded(body string) *httptest.ResponseRecorder {
	return h.SendRequestWithoutToken(http.MethodPost, PlayerOnboardedPath, body)
}

// SendCardPackPurchased はカードパック購入イベントの受信へ本文を送ります。
func (h *Harness) SendCardPackPurchased(body string) *httptest.ResponseRecorder {
	return h.SendRequestWithoutToken(http.MethodPost, CardPackPurchasedPath, body)
}
