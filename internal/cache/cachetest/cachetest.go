// Package cachetest は定義キャッシュのテストで共有する偽の保管先を提供する。
package cachetest

import (
	"context"
	"errors"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

// ErrStoreUnreachable は偽の保管先が取得に失敗したときに返すエラーです。
var ErrStoreUnreachable = errors.New("store-unreachable")

// FakeCardRepository はカード定義の偽の保管先です。
type FakeCardRepository struct {
	Cards []*domain.Card
	Err   error
}

// NewFakeCardRepository は指定のカード ID のカード定義を返す FakeCardRepository を生成します。
func NewFakeCardRepository(cardIDs ...string) *FakeCardRepository {
	cards := make([]*domain.Card, 0, len(cardIDs))
	for _, cardID := range cardIDs {
		cards = append(cards, &domain.Card{CardID: cardID})
	}
	return &FakeCardRepository{Cards: cards}
}

// FindAll は設定済みのカード定義とエラーを返します。
func (f *FakeCardRepository) FindAll(_ context.Context) ([]*domain.Card, error) {
	return f.Cards, f.Err
}

// FakeProductRepository はプロダクト定義の偽の保管先です。
type FakeProductRepository struct {
	Products []*domain.Product
	Err      error
}

// NewFakeProductRepository は指定のプロダクト ID のプロダクト定義を返す FakeProductRepository を生成します。
func NewFakeProductRepository(productIDs ...string) *FakeProductRepository {
	products := make([]*domain.Product, 0, len(productIDs))
	for _, productID := range productIDs {
		products = append(products, &domain.Product{ProductID: productID})
	}
	return &FakeProductRepository{Products: products}
}

// FindAll は設定済みのプロダクト定義とエラーを返します。
func (f *FakeProductRepository) FindAll(_ context.Context) ([]*domain.Product, error) {
	return f.Products, f.Err
}

// FakeInitiativeRepository は施策定義の偽の保管先です。
type FakeInitiativeRepository struct {
	Initiatives []*domain.Initiative
	Err         error
}

// NewFakeInitiativeRepository は指定の施策 ID の施策定義を返す FakeInitiativeRepository を生成します。
func NewFakeInitiativeRepository(initiativeIDs ...string) *FakeInitiativeRepository {
	initiatives := make([]*domain.Initiative, 0, len(initiativeIDs))
	for _, initiativeID := range initiativeIDs {
		initiatives = append(initiatives, &domain.Initiative{InitiativeID: initiativeID})
	}
	return &FakeInitiativeRepository{Initiatives: initiatives}
}

// FindAll は設定済みの施策定義とエラーを返します。
func (f *FakeInitiativeRepository) FindAll(_ context.Context) ([]*domain.Initiative, error) {
	return f.Initiatives, f.Err
}
