package presenter

import (
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

// ToProducts は施策つきのプロダクトの一覧を API の応答の型に変換します。
func ToProducts(products []domain.ProductWithInitiatives) []apicard.Product {
	result := make([]apicard.Product, len(products))
	for i, product := range products {
		result[i] = apicard.Product{
			ProductID:   product.Product.ProductID,
			Faction:     product.Product.Faction,
			ProductName: product.Product.ProductName,
			IsActive:    product.Product.IsActive,
			Initiatives: ToInitiatives(product.Initiatives),
		}
	}
	return result
}

// ToInitiatives は施策の一覧を API の応答の型に変換します。
func ToInitiatives(initiatives []domain.Initiative) []apicard.Initiative {
	result := make([]apicard.Initiative, len(initiatives))
	for i, initiative := range initiatives {
		result[i] = apicard.Initiative{
			InitiativeID: initiative.InitiativeID,
			ProductID:    initiative.ProductID,
			Kind:         apicard.InitiativeKind(initiative.Kind),
			Name:         initiative.Name,
			InsightCost:  initiative.InsightCost,
			EffectText:   initiative.EffectText,
			Effect:       initiative.Effect,
			IsActive:     initiative.IsActive,
		}
	}
	return result
}
