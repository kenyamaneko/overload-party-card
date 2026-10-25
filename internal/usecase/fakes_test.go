package usecase_test

import (
	"context"
	"maps"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/usecase"
	"github.com/kenyamaneko/overload-party-card/internal/usecase/usecasetest"
)

type fakeProcessedEventRepository struct {
	recorded  map[string]string
	insertErr error
}

func newFakeProcessedEventRepository() *fakeProcessedEventRepository {
	return &fakeProcessedEventRepository{recorded: map[string]string{}}
}

func (r *fakeProcessedEventRepository) takeSnapshot() map[string]string {
	return maps.Clone(r.recorded)
}

func (r *fakeProcessedEventRepository) restoreSnapshot(recorded map[string]string) {
	r.recorded = recorded
}

func (r *fakeProcessedEventRepository) Insert(_ context.Context, eventID, eventType string) (bool, error) {
	if r.insertErr != nil {
		return false, r.insertErr
	}
	_, isRecorded := r.recorded[eventID]
	if isRecorded {
		return false, nil
	}
	r.recorded[eventID] = eventType
	return true, nil
}

type fakeCardPackRepository struct {
	packs  map[string]domain.CardPack
	getErr error
}

func newFakeCardPackRepository(packs ...domain.CardPack) *fakeCardPackRepository {
	r := &fakeCardPackRepository{packs: map[string]domain.CardPack{}}
	for _, p := range packs {
		r.packs[p.PackID] = p
	}
	return r
}

func (r *fakeCardPackRepository) GetPack(_ context.Context, packID string) (*domain.CardPack, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	pack, isFound := r.packs[packID]
	if !isFound {
		return nil, port.ErrNotFound
	}
	return &pack, nil
}

type fakeGrantTransactionRunner struct {
	repositories    port.GrantRepositories
	processedEvents *fakeProcessedEventRepository
	playerCards     *usecasetest.FakePlayerCardRepository
}

func (r *fakeGrantTransactionRunner) RunInTransaction(_ context.Context, fn func(repositories port.GrantRepositories) error) error {
	eventsBefore := r.processedEvents.takeSnapshot()
	cardsBefore := r.playerCards.TakeSnapshot()
	err := fn(r.repositories)
	if err != nil {
		r.processedEvents.restoreSnapshot(eventsBefore)
		r.playerCards.RestoreSnapshot(cardsBefore)
	}
	return err
}

type grantHarness struct {
	interactor      *usecase.GrantInteractor
	processedEvents *fakeProcessedEventRepository
	cardPacks       *fakeCardPackRepository
	playerCards     *usecasetest.FakePlayerCardRepository
}

func newGrantHarness(packs ...domain.CardPack) *grantHarness {
	processedEvents := newFakeProcessedEventRepository()
	cardPacks := newFakeCardPackRepository(packs...)
	playerCards := usecasetest.NewFakePlayerCardRepository()
	runner := &fakeGrantTransactionRunner{
		repositories: port.GrantRepositories{
			ProcessedEvents: processedEvents,
			CardPacks:       cardPacks,
			PlayerCards:     playerCards,
		},
		processedEvents: processedEvents,
		playerCards:     playerCards,
	}
	return &grantHarness{
		interactor:      usecase.NewGrantInteractor(runner),
		processedEvents: processedEvents,
		cardPacks:       cardPacks,
		playerCards:     playerCards,
	}
}

func newPack(packID string, isActive bool, cards ...domain.CardPackCard) domain.CardPack {
	return domain.CardPack{PackID: packID, Description: "テストパック", IsActive: isActive, Cards: cards}
}

func newPackCard(cardID string, copies int) domain.CardPackCard {
	return domain.CardPackCard{CardID: cardID, Copies: copies}
}
