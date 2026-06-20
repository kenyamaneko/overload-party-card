package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

// ProductHandler はプロダクトマスター配信の REST エンドポイントを処理します。
type ProductHandler struct {
	products    *cache.ProductCache
	initiatives *cache.InitiativeCache
}

// NewProductHandler は ProductHandler を生成します。
func NewProductHandler(products *cache.ProductCache, initiatives *cache.InitiativeCache) *ProductHandler {
	return &ProductHandler{products: products, initiatives: initiatives}
}

// productWithInitiatives はプロダクトに施策を合成した表現です。
type productWithInitiatives struct {
	domain.Product
	Initiatives []domain.Initiative `json:"initiatives"`
}

// ListAll はプロダクト定義に施策を合成して全件返します。
func (h *ProductHandler) ListAll(c *gin.Context) {
	byProduct := make(map[string][]domain.Initiative)
	for _, initiative := range h.initiatives.All() {
		byProduct[initiative.ProductID] = append(byProduct[initiative.ProductID], initiative)
	}

	products := h.products.All()
	resp := make([]productWithInitiatives, 0, len(products))
	for _, product := range products {
		resp = append(resp, productWithInitiatives{Product: product, Initiatives: byProduct[product.ProductID]})
	}
	c.JSON(http.StatusOK, resp)
}
