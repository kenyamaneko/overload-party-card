package usecase

import (
	"fmt"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// ProductInteractor はプロダクトマスターの読み取り操作を提供します。
type ProductInteractor struct {
	productCache    port.ProductCache
	initiativeCache port.InitiativeCache
}

// NewProductInteractor は ProductInteractor を生成します。
func NewProductInteractor(productCache port.ProductCache, initiativeCache port.InitiativeCache) *ProductInteractor {
	return &ProductInteractor{productCache: productCache, initiativeCache: initiativeCache}
}

// ListProducts は全プロダクトに、そのプロダクトの施策を付与して返します。
// 施策を 1 件も持たないプロダクトがあるときは、マスターデータの不整合としてエラーにします。
func (pi *ProductInteractor) ListProducts() ([]domain.ProductWithInitiatives, error) {
	initiativesByProductID := make(map[string][]domain.Initiative)
	for _, initiative := range pi.initiativeCache.FindAll() {
		initiativesByProductID[initiative.ProductID] = append(initiativesByProductID[initiative.ProductID], initiative)
	}

	products := pi.productCache.FindAll()
	productsWithInitiatives := make([]domain.ProductWithInitiatives, len(products))
	for i, product := range products {
		initiatives, ok := initiativesByProductID[product.ProductID]
		if !ok {
			return nil, fmt.Errorf("product %s has no initiatives", product.ProductID)
		}
		productsWithInitiatives[i] = domain.ProductWithInitiatives{Product: product, Initiatives: initiatives}
	}
	return productsWithInitiatives, nil
}
