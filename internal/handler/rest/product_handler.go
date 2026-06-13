package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
)

// ProductHandler はプロダクトマスター配信の REST エンドポイントを処理します。
type ProductHandler struct {
	products *cache.ProductCache
}

// NewProductHandler は ProductHandler を生成します。
func NewProductHandler(products *cache.ProductCache) *ProductHandler {
	return &ProductHandler{products: products}
}

// ListAll はプロダクト定義全件を返します。
// battle 起動時のキャッシュロードと client のプロダクト表示で使用されます。
func (h *ProductHandler) ListAll(c *gin.Context) {
	c.JSON(http.StatusOK, h.products.All())
}
