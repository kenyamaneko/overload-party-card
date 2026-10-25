package port

import "github.com/kenyamaneko/overload-party-card/internal/domain"

// CardCache は読み込み済みのカード定義を参照するインターフェースです。
type CardCache interface {
	// FindByID は指定カード ID のカード定義を返します。存在しなければ nil を返します。
	FindByID(cardID string) *domain.Card
}

// ProductCache は読み込み済みのプロダクト定義を参照するインターフェースです。
type ProductCache interface {
	// FindByID は指定プロダクト ID のプロダクト定義を返します。存在しなければ nil を返します。
	FindByID(productID string) *domain.Product
	// FindAll は読み込み済みの全プロダクト定義を返します。
	FindAll() []domain.Product
}

// InitiativeCache は読み込み済みの施策定義を参照するインターフェースです。
type InitiativeCache interface {
	// FindByID は指定施策 ID の施策定義を返します。存在しなければ nil を返します。
	FindByID(initiativeID string) *domain.Initiative
	// FindAll は読み込み済みの全施策定義を施策 ID の昇順で返します。
	FindAll() []domain.Initiative
}
