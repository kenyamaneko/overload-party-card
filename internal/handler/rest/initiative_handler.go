package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
)

// InitiativeHandler は施策マスター配信の REST エンドポイントを処理します。
type InitiativeHandler struct {
	initiatives *cache.InitiativeCache
}

// NewInitiativeHandler は InitiativeHandler を生成します。
func NewInitiativeHandler(initiatives *cache.InitiativeCache) *InitiativeHandler {
	return &InitiativeHandler{initiatives: initiatives}
}

// ListAll は施策定義全件を返します。
func (h *InitiativeHandler) ListAll(c *gin.Context) {
	c.JSON(http.StatusOK, h.initiatives.All())
}
