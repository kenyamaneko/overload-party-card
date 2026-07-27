package rest

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiscenario "github.com/kenyamaneko/overload-party-scenario/packages/api-scenario"
	apishop "github.com/kenyamaneko/overload-party-shop/packages/api-shop"

	pubsubadapter "github.com/kenyamaneko/overload-party-card/internal/adapter/pubsub"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"
)

const (
	fxPushPlayerID = "TST-PLAYER-1"
	fxPushCardID   = "TST-0001"
)

// fakeCardPackRepo は port.CardPackRepo のテスト用スタブ。pack_id をキーに
// 固定のパック定義を返す。
type fakeCardPackRepo struct {
	packs map[string]*domain.CardPack
}

func newFakeCardPackRepo(packs map[string]*domain.CardPack) *fakeCardPackRepo {
	return &fakeCardPackRepo{packs: packs}
}

func (r *fakeCardPackRepo) GetPack(_ context.Context, packID string) (*domain.CardPack, error) {
	pack, ok := r.packs[packID]
	if !ok {
		return nil, port.ErrNotFound
	}
	return pack, nil
}

// fakeProcessedEventRepo は port.ProcessedEventRepo のテスト用スタブ。
// event_id ごとの挿入済み状態を追跡し、実 DB の UNIQUE 制約相当の dedup を再現する。
type fakeProcessedEventRepo struct {
	inserted map[string]bool
}

func newFakeProcessedEventRepo() *fakeProcessedEventRepo {
	return &fakeProcessedEventRepo{inserted: make(map[string]bool)}
}

func (r *fakeProcessedEventRepo) Insert(_ context.Context, eventID, _ string) (bool, error) {
	if r.inserted[eventID] {
		return false, nil
	}
	r.inserted[eventID] = true
	return true, nil
}

// newPubSubPushHandlerFixture は実 subscriber (pubsubadapter) と実 GrantInteractor を
// fake repo に載せた PubSubPushHandler を組み立てる。既存の subscriber ロジックを
// 通した「受け口 (push handler) を叩くと実際に付与される」結合検証に使う。
func newPubSubPushHandlerFixture(packRepo *fakeCardPackRepo) (*PubSubPushHandler, *fakePlayerCardRepo) {
	pcRepo := newFakePlayerCardRepo()
	grantInteractor := usecase.NewGrantInteractor(packRepo, pcRepo)
	eventRepo := newFakeProcessedEventRepo()

	onboardedSub := pubsubadapter.NewPlayerOnboardedSubscriber(grantInteractor, eventRepo)
	cardPackPurchasedSub := pubsubadapter.NewCardPackPurchasedSubscriber(grantInteractor, eventRepo)

	return NewPubSubPushHandler(onboardedSub.Handle, cardPackPurchasedSub.Handle), pcRepo
}

// servePush は Cloud Pub/Sub push subscription が送るリクエストを模した body を
// gin handler へ実際の HTTP ディスパッチ経路 (Engine.ServeHTTP) で通す。
// gin.Context を直接呼ぶだけでは応答ヘッダのフラッシュ (200 応答時など) が
// Engine の終端処理に依存するため確定しない。
func servePush(t *testing.T, handlerFunc gin.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := gin.New()
	r.POST("/", handlerFunc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
	return w
}

// pushEnvelopeBody は payload を base64 埋め込みした push envelope の JSON 文字列を返す。
func pushEnvelopeBody(t *testing.T, payload []byte) string {
	t.Helper()
	env := struct {
		Message struct {
			Data string `json:"data"`
		} `json:"message"`
	}{}
	env.Message.Data = base64.StdEncoding.EncodeToString(payload)
	b, err := json.Marshal(env)
	require.NoError(t, err)
	return string(b)
}

func TestPubSubPushHandler_HandleCardPackPurchased(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pack := &domain.CardPack{
		PackID:   "faction_set_tuners",
		IsActive: true,
		Cards:    []domain.CardPackCard{{CardID: fxPushCardID, Copies: 3}},
	}

	t.Run("card-pack-purchased の push 受け口", func(t *testing.T) {
		t.Run("push 本文を受け取ると、カードが付与される", func(t *testing.T) {
			h, pcRepo := newPubSubPushHandlerFixture(newFakeCardPackRepo(map[string]*domain.CardPack{
				"faction_set_tuners": pack,
			}))
			payload, err := json.Marshal(apishop.CardPackPurchasedEvent{
				EventType:  apishop.EventTypeCardPackPurchased,
				EventID:    "evt-1",
				PlayerID:   fxPushPlayerID,
				CardPackID: "faction_set_tuners",
			})
			require.NoError(t, err)

			w := servePush(t, h.HandleCardPackPurchased, pushEnvelopeBody(t, payload))

			assert.Equal(t, http.StatusOK, w.Code)
			cards, getErr := pcRepo.GetPlayerCards(context.Background(), fxPushPlayerID)
			require.NoError(t, getErr)
			require.Len(t, cards, 1)
			assert.Equal(t, fxPushCardID, cards[0].CardID)
			assert.Equal(t, 3, cards[0].Count)
		})

		t.Run("base64 で復号できない data のとき、400 になり応答本文の error が base64 不正を示す文言と完全に一致する", func(t *testing.T) {
			h, _ := newPubSubPushHandlerFixture(newFakeCardPackRepo(nil))
			body := `{"message":{"data":"not-valid-base64!!!"}}`

			w := servePush(t, h.HandleCardPackPurchased, body)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, "pubsub push: malformed base64 data", decodeBody(t, w)["error"])
		})

		t.Run("既知でない event_type のとき、400 とは異なる 500 になり、応答本文の error は原因を伏せた固定文言でカードも付与されない", func(t *testing.T) {
			h, pcRepo := newPubSubPushHandlerFixture(newFakeCardPackRepo(map[string]*domain.CardPack{
				"faction_set_tuners": pack,
			}))
			payload, err := json.Marshal(apishop.CardPackPurchasedEvent{
				EventType:  "unknown",
				EventID:    "evt-2",
				PlayerID:   fxPushPlayerID,
				CardPackID: "faction_set_tuners",
			})
			require.NoError(t, err)

			w := servePush(t, h.HandleCardPackPurchased, pushEnvelopeBody(t, payload))

			assert.Equal(t, http.StatusInternalServerError, w.Code)
			assert.Equal(t, "internal server error", decodeBody(t, w)["error"])
			cards, getErr := pcRepo.GetPlayerCards(context.Background(), fxPushPlayerID)
			require.NoError(t, getErr)
			assert.Empty(t, cards)
		})

		t.Run("同じイベントを二度投げても、2 回目はカードが二重に付与されない", func(t *testing.T) {
			h, pcRepo := newPubSubPushHandlerFixture(newFakeCardPackRepo(map[string]*domain.CardPack{
				"faction_set_tuners": pack,
			}))
			payload, err := json.Marshal(apishop.CardPackPurchasedEvent{
				EventType:  apishop.EventTypeCardPackPurchased,
				EventID:    "evt-repeat",
				PlayerID:   fxPushPlayerID,
				CardPackID: "faction_set_tuners",
			})
			require.NoError(t, err)
			body := pushEnvelopeBody(t, payload)

			w1 := servePush(t, h.HandleCardPackPurchased, body)
			w2 := servePush(t, h.HandleCardPackPurchased, body)

			assert.Equal(t, http.StatusOK, w1.Code)
			assert.Equal(t, http.StatusOK, w2.Code)
			cards, getErr := pcRepo.GetPlayerCards(context.Background(), fxPushPlayerID)
			require.NoError(t, getErr)
			require.Len(t, cards, 1, "GrantPack は 1 回だけ実行される")
			assert.Equal(t, 3, cards[0].Count)
		})
	})
}

func TestPubSubPushHandler_HandlePlayerOnboarded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	packs := map[string]*domain.CardPack{
		"basic":              {PackID: "basic", IsActive: true, Cards: []domain.CardPackCard{{CardID: "TST-0002", Copies: 2}}},
		"faction_set_tuners": {PackID: "faction_set_tuners", IsActive: true, Cards: []domain.CardPackCard{{CardID: fxPushCardID, Copies: 3}}},
	}

	t.Run("player-onboarded の push 受け口", func(t *testing.T) {
		t.Run("push 本文を受け取ると、basic と faction のカードが付与される", func(t *testing.T) {
			h, pcRepo := newPubSubPushHandlerFixture(newFakeCardPackRepo(packs))
			payload, err := json.Marshal(apiscenario.PlayerOnboardedEvent{
				EventType:        apiscenario.EventTypePlayerOnboarded,
				EventID:          "evt-1",
				PlayerID:         fxPushPlayerID,
				InitialFactionID: "Tuners",
			})
			require.NoError(t, err)

			w := servePush(t, h.HandlePlayerOnboarded, pushEnvelopeBody(t, payload))

			assert.Equal(t, http.StatusOK, w.Code)
			cards, getErr := pcRepo.GetPlayerCards(context.Background(), fxPushPlayerID)
			require.NoError(t, getErr)
			assert.ElementsMatch(t, []*domain.PlayerCard{
				{PlayerID: fxPushPlayerID, CardID: "TST-0002", Count: 2},
				{PlayerID: fxPushPlayerID, CardID: fxPushCardID, Count: 3},
			}, cards)
		})

		t.Run("JSON として復号できない body のとき、400 になり応答本文の error が envelope 不正を示す文言と完全に一致する", func(t *testing.T) {
			h, _ := newPubSubPushHandlerFixture(newFakeCardPackRepo(nil))

			w := servePush(t, h.HandlePlayerOnboarded, `not-json-at-all`)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, "pubsub push: malformed envelope", decodeBody(t, w)["error"])
		})

		t.Run("message フィールドが無いとき、400 になり応答本文の error が envelope 不正を示す文言と完全に一致する", func(t *testing.T) {
			h, _ := newPubSubPushHandlerFixture(newFakeCardPackRepo(nil))

			w := servePush(t, h.HandlePlayerOnboarded, `{}`)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, "pubsub push: malformed envelope", decodeBody(t, w)["error"])
		})

		t.Run("message.data が空文字のとき、400 になり応答本文の error が envelope 不正を示す文言と完全に一致する", func(t *testing.T) {
			h, _ := newPubSubPushHandlerFixture(newFakeCardPackRepo(nil))

			w := servePush(t, h.HandlePlayerOnboarded, `{"message":{"data":""}}`)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, "pubsub push: malformed envelope", decodeBody(t, w)["error"])
		})
	})
}
