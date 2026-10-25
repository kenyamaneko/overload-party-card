package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/handler/rest"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"

	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
)

const healthStatusOK = "ok"

// New は card サービスの HTTP ルーターを構築します。
func New(
	cardHandler *rest.CardHandler,
	deckHandler *rest.DeckHandler,
	playerCardHandler *rest.PlayerCardHandler,
	productHandler *rest.ProductHandler,
	pubSubPushHandler *rest.PubSubPushHandler,
	authVerifier internalauth.Verifier,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), rest.LogInternalErrors())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, apicard.HealthResponse{Status: healthStatusOK})
	})

	// 到達制御は Cloud Run の呼び出し IAM が担うため、Pub/Sub の push 受け口はアプリ層では認証なしで開放する。
	internal := r.Group("/internal/v1")
	{
		internal.POST("/pubsub/player-onboarded", pubSubPushHandler.ReceivePlayerOnboarded)
		internal.POST("/pubsub/card-pack-purchased", pubSubPushHandler.ReceiveCardPackPurchased)
	}

	api := r.Group("/api/v1/cards")
	api.Use(internalauth.VerifyInternalAuth(authVerifier))
	{
		api.GET("/cards", playerCardHandler.GetPlayerCards)
		api.GET("/cards/with-ownership", cardHandler.ListForPlayer)
		api.GET("/products", productHandler.ListAll)

		decks := api.Group("/decks")
		decks.GET("", deckHandler.GetDecks)
		decks.POST("", deckHandler.CreateDeck)

		deck := decks.Group("/:deckId", rest.ParseDeckID())
		deck.GET("", deckHandler.GetDeck)
		deck.PUT("", deckHandler.UpdateDeck)
		deck.DELETE("", deckHandler.DeleteDeck)
		deck.POST("/validate-for-battle", deckHandler.ValidateDeckForBattle)
	}
	return r
}
