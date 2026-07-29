package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/handler/rest"

	internalauth "github.com/kenyamaneko/overload-party-gateway/packages/internalauth-go"
)

// New は card サービスの HTTP ルーターを構築します。
func New(
	cardH *rest.CardHandler,
	deckH *rest.DeckHandler,
	playerCardH *rest.PlayerCardHandler,
	productH *rest.ProductHandler,
	initiativeH *rest.InitiativeHandler,
	pubsubPushH *rest.PubSubPushHandler,
	authVerifier internalauth.Verifier,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 内部 master 配信 (battle / gateway 起動時のキャッシュロード用) と
	// Cloud Pub/Sub push 受け口。到達制御は Cloud Run の呼び出し IAM が担うため、
	// アプリ層では認証なしで開放する。
	internal := r.Group("/internal/v1")
	{
		internal.GET("/cards", cardH.ListAllRaw)
		internal.GET("/initiatives", initiativeH.ListAll)
		internal.POST("/pubsub/player-onboarded", pubsubPushH.HandlePlayerOnboarded)
		internal.POST("/pubsub/card-pack-purchased", pubsubPushH.HandleCardPackPurchased)
	}

	// gateway 経由の player-scoped API。internalauth middleware が JWT 検証して
	// sub クレームを context に注入し、handler は context 経由で player_id を取る。
	api := r.Group("/api/v1/cards")
	api.Use(internalauth.VerifyInternalAuth(authVerifier))
	{
		api.GET("/cards", playerCardH.GetPlayerCards)
		api.GET("/cards/with-ownership", cardH.ListForPlayer)
		api.GET("/products", productH.ListAll)
		api.GET("/decks", deckH.GetDecks)
		api.GET("/decks/:deckId", deckH.GetDeck)
		api.POST("/decks", deckH.CreateDeck)
		api.PUT("/decks/:deckId", deckH.UpdateDeck)
		api.DELETE("/decks/:deckId", deckH.DeleteDeck)
		api.POST("/decks/:deckId/validate-for-battle", deckH.ValidateDeckForBattle)
	}
	return r
}
