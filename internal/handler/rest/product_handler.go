package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/presenter"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"
)

// ProductHandler はプロダクトマスター配信の REST エンドポイントを処理します。
type ProductHandler struct {
	productInteractor *usecase.ProductInteractor
}

// NewProductHandler は ProductHandler を生成します。
func NewProductHandler(productInteractor *usecase.ProductInteractor) *ProductHandler {
	return &ProductHandler{productInteractor: productInteractor}
}

// ListAll はプロダクト定義に施策を合成して全件返します。
func (h *ProductHandler) ListAll(c *gin.Context) {
	products, err := h.productInteractor.ListProducts()
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, presenter.ToProducts(products))
}
