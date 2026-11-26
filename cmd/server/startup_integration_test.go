//go:build integration

package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net"
	"net/http"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
	apiscenario "github.com/kenyamaneko/overload-party-scenario/packages/api-scenario"
	apishop "github.com/kenyamaneko/overload-party-shop/packages/api-shop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/cmd/server/servertest"
	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/config/configtest"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

const (
	testPlayerID     = "11111111-1111-4111-8111-111111111111"
	testEventID      = "22222222-2222-4222-8222-222222222222"
	testTokenTTL     = 5 * time.Minute
	testRSAKeyBits   = 2048
	testPurchasePack = "TST-0100"
)

func generateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, testRSAKeyBits)
	require.NoError(t, err)
	return key
}

func encodePublicKeyPEM(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func signInternalToken(t *testing.T, key *rsa.PrivateKey, playerID string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		Subject:   playerID,
		Issuer:    internalauth.ExpectedIssuer,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(testTokenTTL)),
	})
	token.Header["kid"] = string(internalauth.DefaultKeyID)
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

type masterDataActivity struct {
	isCardActive       bool
	isProductActive    bool
	isInitiativeActive bool
}

var allMasterDataActive = masterDataActivity{isCardActive: true, isProductActive: true, isInitiativeActive: true}

func execSQL(t *testing.T, statement string, args ...any) {
	t.Helper()
	_, err := sharedPg.Pool.Exec(context.Background(), statement, args...)
	require.NoError(t, err)
}

func insertMasterData(t *testing.T, activity masterDataActivity) {
	t.Helper()
	for _, cardID := range []string{"TST-0001", "TST-0002", "TST-0003", "TST-0004"} {
		execSQL(t, `INSERT INTO card.card_definitions
			(card_id, card_name, resource_label, faction, card_type, resizable, elastic, stats, restriction, is_active)
			VALUES ($1, $2, 'TST-label', 'Neutral', 'Strategy', false, false, '{}'::jsonb, 'unlimited', $3)`,
			cardID, "TST-name-"+cardID, activity.isCardActive)
	}
	for _, product := range [][]string{{"TST-0001", "SHE"}, {"TST-0002", "Tenki"}} {
		execSQL(t, `INSERT INTO card.products (product_id, faction, product_name, is_active) VALUES ($1, $2, $3, $4)`,
			product[0], product[1], "TST-product-"+product[0], activity.isProductActive)
	}
	for _, initiative := range [][]string{
		{"TST-0001", "TST-0001", "routine"},
		{"TST-0002", "TST-0001", "special"},
		{"TST-0003", "TST-0002", "routine"},
	} {
		execSQL(t, `INSERT INTO card.initiatives
			(initiative_id, product_id, kind, name, insight_cost, effect_text, effect, is_active)
			VALUES ($1, $2, $3, $4, 0, 'TST-effect', '{}'::jsonb, $5)`,
			initiative[0], initiative[1], initiative[2], "TST-initiative-"+initiative[0], activity.isInitiativeActive)
	}
	insertCardPack(t, "basic", map[string]int{"TST-0001": 1, "TST-0002": 2})
	insertCardPack(t, "faction_set_she", map[string]int{"TST-0003": 3})
	insertCardPack(t, testPurchasePack, map[string]int{"TST-0004": 2, "TST-0003": 1})
}

func insertCardPack(t *testing.T, packID string, copiesByCardID map[string]int) {
	t.Helper()
	execSQL(t, `INSERT INTO card.card_pack (pack_id, description, is_active) VALUES ($1, 'TST-pack', true)`, packID)
	for cardID, copies := range copiesByCardID {
		execSQL(t, `INSERT INTO card.card_pack_cards (pack_id, card_id, copies) VALUES ($1, $2, $3)`, packID, cardID, copies)
	}
}

func prepareMasterData(t *testing.T, activity masterDataActivity) {
	t.Helper()
	sharedPg.Truncate(t)
	insertMasterData(t, activity)
}

func pickFreePort(t *testing.T) string {
	t.Helper()
	_, port, err := net.SplitHostPort(servertest.PickFreeAddress(t))
	require.NoError(t, err)
	return port
}

func readConnectionString(t *testing.T) string {
	t.Helper()
	connectionString, err := sharedPg.Container.ConnectionString(context.Background(), "sslmode=disable")
	require.NoError(t, err)
	return connectionString
}

func setStartupEnvironment(t *testing.T, port string, internalAuthKey *rsa.PrivateKey, overrides map[string]string) {
	t.Helper()
	environment := map[string]string{
		"PORT":                     port,
		"DATABASE_CONN":            readConnectionString(t),
		"INTERNAL_AUTH_PUBLIC_KEY": encodePublicKeyPEM(t, internalAuthKey),
	}
	for key, value := range overrides {
		environment[key] = value
	}
	configtest.SetEnvironment(t, environment)
}

type startedServer struct {
	client          *apicard.ClientWithResponses
	internalAuthKey *rsa.PrivateKey
	stop            func() error
}

func newCardClient(t *testing.T, port string) *apicard.ClientWithResponses {
	t.Helper()
	client, err := apicard.NewClientWithResponses("http://127.0.0.1:"+port, apicard.WithHTTPClient(servertest.NewHTTPClient()))
	require.NoError(t, err)
	return client
}

func startServer(t *testing.T) *startedServer {
	t.Helper()
	port := pickFreePort(t)
	internalAuthKey := generateRSAKey(t)
	setStartupEnvironment(t, port, internalAuthKey, nil)
	exited := make(chan error, 1)
	go func() { exited <- run() }()
	stop := sync.OnceValue(func() error {
		require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))
		return servertest.AwaitExit(t, exited)
	})
	client := newCardClient(t, port)
	require.Eventually(t, func() bool {
		response, err := client.GetHealthWithResponse(context.Background())
		return err == nil && response.StatusCode() == http.StatusOK
	}, servertest.WaitTimeout, servertest.PollInterval)
	t.Cleanup(func() { require.NoError(t, stop()) })
	return &startedServer{client: client, internalAuthKey: internalAuthKey, stop: stop}
}

func runUntilFailure(t *testing.T, overrides map[string]string) error {
	t.Helper()
	setStartupEnvironment(t, pickFreePort(t), generateRSAKey(t), overrides)
	exited := make(chan error, 1)
	go func() { exited <- run() }()
	return servertest.AwaitExit(t, exited)
}

func addInternalAuthHeader(token string) apicard.RequestEditorFn {
	return func(_ context.Context, request *http.Request) error {
		request.Header.Set(internalauth.HeaderName, token)
		return nil
	}
}

func (s *startedServer) requestProducts(t *testing.T, token string) *apicard.ListProductsForPlayerResponse {
	t.Helper()
	response, err := s.client.ListProductsForPlayerWithResponse(context.Background(), addInternalAuthHeader(token))
	require.NoError(t, err)
	return response
}

func (s *startedServer) requestOwnedCardCounts(t *testing.T, playerID string) map[string]int {
	t.Helper()
	token := signInternalToken(t, s.internalAuthKey, playerID)
	response, err := s.client.ListPlayerCardsWithResponse(context.Background(), addInternalAuthHeader(token))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode())
	counts := map[string]int{}
	for _, card := range *response.JSON200 {
		counts[card.CardID] += card.Count
	}
	return counts
}

func encodePushEnvelope(t *testing.T, event any) apicard.PubSubPushEnvelope {
	t.Helper()
	data, err := json.Marshal(event)
	require.NoError(t, err)
	var envelope apicard.PubSubPushEnvelope
	envelope.Message.Data = data
	return envelope
}

func (s *startedServer) sendPlayerOnboarded(t *testing.T) int {
	t.Helper()
	envelope := encodePushEnvelope(t, apiscenario.PlayerOnboardedEvent{
		EventType:        apiscenario.EventTypePlayerOnboarded,
		EventID:          testEventID,
		PlayerID:         testPlayerID,
		InitialFactionID: "SHE",
	})
	response, err := s.client.PubsubPlayerOnboardedWithResponse(context.Background(), envelope)
	require.NoError(t, err)
	return response.StatusCode()
}

func (s *startedServer) sendCardPackPurchased(t *testing.T) int {
	t.Helper()
	envelope := encodePushEnvelope(t, apishop.CardPackPurchasedEvent{
		EventType:  apishop.EventTypeCardPackPurchased,
		EventID:    testEventID,
		PlayerID:   testPlayerID,
		CardPackID: testPurchasePack,
	})
	response, err := s.client.PubsubCardPackPurchasedWithResponse(context.Background(), envelope)
	require.NoError(t, err)
	return response.StatusCode()
}

func listProductIDs(products []apicard.Product) []string {
	productIDs := []string{}
	for _, product := range products {
		productIDs = append(productIDs, product.ProductID)
	}
	return productIDs
}

func listInitiativeIDsByProduct(products []apicard.Product) map[string][]string {
	initiativeIDsByProduct := map[string][]string{}
	for _, product := range products {
		for _, initiative := range product.Initiatives {
			initiativeIDsByProduct[product.ProductID] = append(initiativeIDsByProduct[product.ProductID], initiative.InitiativeID)
		}
	}
	return initiativeIDsByProduct
}

func TestServerStartup(t *testing.T) {
	t.Run("[カードサービス起動]カードサービスの起動と終了", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("カード定義TST-0001からTST-0004、プロダクト定義TST-0001とTST-0002、プロダクトTST-0001の施策定義TST-0001とTST-0002、プロダクトTST-0002の施策定義TST-0003がデータベースに登録されているとき", func(t *testing.T) {
				t.Run("起動すると、/healthが200を返す", func(t *testing.T) {
					prepareMasterData(t, allMasterDataActive)
					server := startServer(t)

					response, err := server.client.GetHealthWithResponse(context.Background())

					require.NoError(t, err)
					assert.Equal(t, http.StatusOK, response.StatusCode())
				})

				t.Run("起動後に内部トークンを付けて/api/v1/cards/productsを呼ぶと", func(t *testing.T) {
					requestProducts := func(t *testing.T) []apicard.Product {
						prepareMasterData(t, allMasterDataActive)
						server := startServer(t)
						response := server.requestProducts(t, signInternalToken(t, server.internalAuthKey, testPlayerID))
						require.Equal(t, http.StatusOK, response.StatusCode())
						return *response.JSON200
					}

					t.Run("応答のプロダクトIDがTST-0001とTST-0002になる", func(t *testing.T) {
						products := requestProducts(t)

						assert.ElementsMatch(t, []string{"TST-0001", "TST-0002"}, listProductIDs(products))
					})

					t.Run("応答のTST-0001のinitiativesの施策IDがTST-0001とTST-0002、TST-0002のinitiativesの施策IDがTST-0003になる", func(t *testing.T) {
						products := requestProducts(t)

						initiativeIDsByProduct := listInitiativeIDsByProduct(products)
						assert.ElementsMatch(t, []string{"TST-0001", "TST-0002"}, initiativeIDsByProduct["TST-0001"])
						assert.ElementsMatch(t, []string{"TST-0003"}, initiativeIDsByProduct["TST-0002"])
					})
				})

				t.Run("基本パックがTST-0001を1枚とTST-0002を2枚、faction_set_sheがTST-0003を3枚含むとき", func(t *testing.T) {
					t.Run("初期陣営がSHEのプレイヤー登録完了イベントを/internal/v1/pubsub/player-onboardedへ配信すると、200が返る", func(t *testing.T) {
						prepareMasterData(t, allMasterDataActive)
						server := startServer(t)

						statusCode := server.sendPlayerOnboarded(t)

						assert.Equal(t, http.StatusOK, statusCode)
					})

					t.Run("初期陣営がSHEのプレイヤー登録完了イベントを配信したあと、/api/v1/cards/cardsの所持カード一覧で", func(t *testing.T) {
						t.Run("基本パックのTST-0001のcountが1、TST-0002のcountが2になる", func(t *testing.T) {
							prepareMasterData(t, allMasterDataActive)
							server := startServer(t)
							require.Equal(t, http.StatusOK, server.sendPlayerOnboarded(t))

							counts := server.requestOwnedCardCounts(t, testPlayerID)

							assert.Equal(t, 1, counts["TST-0001"])
							assert.Equal(t, 2, counts["TST-0002"])
						})

						t.Run("faction_set_sheのTST-0003のcountが3になる", func(t *testing.T) {
							prepareMasterData(t, allMasterDataActive)
							server := startServer(t)
							require.Equal(t, http.StatusOK, server.sendPlayerOnboarded(t))

							counts := server.requestOwnedCardCounts(t, testPlayerID)

							assert.Equal(t, 3, counts["TST-0003"])
						})
					})

					t.Run("初期陣営がSHEの同じプレイヤー登録完了イベントを2回続けて配信したあと、/api/v1/cards/cardsの所持カード一覧で", func(t *testing.T) {
						t.Run("基本パックのTST-0001のcountが1、TST-0002のcountが2になる", func(t *testing.T) {
							prepareMasterData(t, allMasterDataActive)
							server := startServer(t)
							require.Equal(t, http.StatusOK, server.sendPlayerOnboarded(t))
							require.Equal(t, http.StatusOK, server.sendPlayerOnboarded(t))

							counts := server.requestOwnedCardCounts(t, testPlayerID)

							assert.Equal(t, 1, counts["TST-0001"])
							assert.Equal(t, 2, counts["TST-0002"])
						})

						t.Run("faction_set_sheのTST-0003のcountが3になる", func(t *testing.T) {
							prepareMasterData(t, allMasterDataActive)
							server := startServer(t)
							require.Equal(t, http.StatusOK, server.sendPlayerOnboarded(t))
							require.Equal(t, http.StatusOK, server.sendPlayerOnboarded(t))

							counts := server.requestOwnedCardCounts(t, testPlayerID)

							assert.Equal(t, 3, counts["TST-0003"])
						})
					})
				})

				t.Run("カードパックTST-0100がTST-0004を2枚とTST-0003を1枚含むとき", func(t *testing.T) {
					t.Run("TST-0100のカードパック購入イベントを/internal/v1/pubsub/card-pack-purchasedへ配信すると、200が返る", func(t *testing.T) {
						prepareMasterData(t, allMasterDataActive)
						server := startServer(t)

						statusCode := server.sendCardPackPurchased(t)

						assert.Equal(t, http.StatusOK, statusCode)
					})

					t.Run("TST-0100のカードパック購入イベントを配信したあと、/api/v1/cards/cardsの所持カード一覧で、TST-0004のcountが2、TST-0003のcountが1になる", func(t *testing.T) {
						prepareMasterData(t, allMasterDataActive)
						server := startServer(t)
						require.Equal(t, http.StatusOK, server.sendCardPackPurchased(t))

						counts := server.requestOwnedCardCounts(t, testPlayerID)

						assert.Equal(t, 2, counts["TST-0004"])
						assert.Equal(t, 1, counts["TST-0003"])
					})

					t.Run("同じTST-0100のカードパック購入イベントを2回続けて配信すると、/api/v1/cards/cardsの所持カード一覧で、TST-0004のcountが2、TST-0003のcountが1になる", func(t *testing.T) {
						prepareMasterData(t, allMasterDataActive)
						server := startServer(t)
						require.Equal(t, http.StatusOK, server.sendCardPackPurchased(t))
						require.Equal(t, http.StatusOK, server.sendCardPackPurchased(t))

						counts := server.requestOwnedCardCounts(t, testPlayerID)

						assert.Equal(t, 2, counts["TST-0004"])
						assert.Equal(t, 1, counts["TST-0003"])
					})
				})

				t.Run("起動後に終了シグナルを受け取ると、エラーなしで終了する", func(t *testing.T) {
					prepareMasterData(t, allMasterDataActive)
					server := startServer(t)

					err := server.stop()

					assert.NoError(t, err)
				})

				t.Run("起動後に終了シグナルを受け取って終了したあとは、/healthへの接続が拒否される", func(t *testing.T) {
					prepareMasterData(t, allMasterDataActive)
					server := startServer(t)
					require.NoError(t, server.stop())

					_, err := server.client.GetHealthWithResponse(context.Background())

					assert.ErrorIs(t, err, syscall.ECONNREFUSED)
				})
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("カード定義・プロダクト定義・施策定義のうち1種だけが0件で、ほかの2種はデータベースに登録されているとき", func(t *testing.T) {
				tests := []struct {
					name     string
					activity masterDataActivity
					wantErr  error
				}{
					{
						name:     "カード定義が0件のとき、起動はカード定義が0件であることを示すエラーで失敗する",
						activity: masterDataActivity{isCardActive: false, isProductActive: true, isInitiativeActive: true},
						wantErr:  cache.ErrNoCardDefinitions,
					},
					{
						name:     "プロダクト定義が0件のとき、起動はプロダクト定義が0件であることを示すエラーで失敗する",
						activity: masterDataActivity{isCardActive: true, isProductActive: false, isInitiativeActive: true},
						wantErr:  cache.ErrNoProductDefinitions,
					},
					{
						name:     "施策定義が0件のとき、起動は施策定義が0件であることを示すエラーで失敗する",
						activity: masterDataActivity{isCardActive: true, isProductActive: true, isInitiativeActive: false},
						wantErr:  cache.ErrNoInitiativeDefinitions,
					},
				}
				for _, tt := range tests {
					t.Run(tt.name, func(t *testing.T) {
						prepareMasterData(t, tt.activity)

						err := runUntilFailure(t, nil)

						assert.ErrorIs(t, err, tt.wantErr)
					})
				}
			})

			t.Run("カード定義・プロダクト定義・施策定義がデータベースに登録されているとき", func(t *testing.T) {
				t.Run("設定した公開鍵に対応しない鍵で署名した内部トークンを付けて/api/v1/cards/productsを呼ぶと、401が返る", func(t *testing.T) {
					prepareMasterData(t, allMasterDataActive)
					server := startServer(t)

					response := server.requestProducts(t, signInternalToken(t, generateRSAKey(t), testPlayerID))

					assert.Equal(t, http.StatusUnauthorized, response.StatusCode())
				})

				t.Run("INTERNAL_AUTH_PUBLIC_KEYがPEM形式でない文字列not-a-pemのとき、起動はINTERNAL_AUTH_PUBLIC_KEYが不正であることを示すエラーで失敗する", func(t *testing.T) {
					prepareMasterData(t, allMasterDataActive)

					err := runUntilFailure(t, map[string]string{"INTERNAL_AUTH_PUBLIC_KEY": "not-a-pem"})

					assert.ErrorContains(t, err, "INTERNAL_AUTH_PUBLIC_KEY is invalid")
				})
			})

			t.Run("DATABASE_CONNが、待ち受けている相手のいないlocalhostのポート1を指すとき、起動はデータベースに接続できないことを示すエラーで失敗する", func(t *testing.T) {
				err := runUntilFailure(t, map[string]string{"DATABASE_CONN": "postgres://TST-user:TST-password@localhost:1/TST-database?sslmode=disable"})

				assert.ErrorIs(t, err, syscall.ECONNREFUSED)
			})
		})
	})
}
