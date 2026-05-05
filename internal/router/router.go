package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/handler/rest"
)

// New は card サービスの HTTP ルーターを構築します。
//
// 配布 (grant) エンドポイントは ADR-032 / ADR-026 に伴い廃止されました。
// 配布は scenario の `player-onboarded` event と shop の `card-pack-purchased` event
// (今後追加) を経由した Pub/Sub 駆動のみで行われます。
func New(
	cardH *rest.CardHandler,
	deckH *rest.DeckHandler,
	playerCardH *rest.PlayerCardHandler,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	internal := r.Group("/internal/v1")
	{
		internal.GET("/cards", cardH.ListAllRaw)

		players := internal.Group("/players/:playerId")
		{
			players.GET("/cards", playerCardH.GetPlayerCards)
			players.GET("/cards/with-ownership", cardH.ListForPlayer)
			players.GET("/decks", deckH.GetDecks)
			players.GET("/decks/:deckId", deckH.GetDeck)
			players.POST("/decks", deckH.CreateDeck)
			players.PUT("/decks/:deckId", deckH.UpdateDeck)
			players.DELETE("/decks/:deckId", deckH.DeleteDeck)
			players.POST("/decks/:deckId/validate-for-battle", deckH.ValidateDeckForBattle)
		}
	}
	return r
}
