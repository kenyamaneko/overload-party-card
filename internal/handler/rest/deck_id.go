package rest

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	deckIDContextKey = "deckID"

	deckIDBase    = 10
	deckIDBitSize = 64
)

// ParseDeckID は URL のデッキ ID を整数として読み取ってリクエストのコンテキストに保存し、読み取れないときは 400 の応答で処理を打ち切ります。
func ParseDeckID() gin.HandlerFunc {
	return func(c *gin.Context) {
		deckID, err := strconv.ParseInt(c.Param("deckId"), deckIDBase, deckIDBitSize)
		if err != nil {
			respondBadRequest(c, invalidDeckIDMessage)
			c.Abort()
			return
		}
		c.Set(deckIDContextKey, deckID)
	}
}
