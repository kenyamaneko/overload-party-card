// Package pubsubtest は購読のテストで共有する偽の保管先とダミー値を提供する。
package pubsubtest

import (
	"context"
	"errors"
	"maps"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

const (
	// PlayerID はテスト用のプレイヤー ID です。
	PlayerID = "11111111-1111-4111-8111-111111111111"
	// EventID はテスト用のイベント ID です。
	EventID = "22222222-2222-4222-8222-222222222222"
	// OtherEventID は EventID と異なるテスト用のイベント ID です。
	OtherEventID = "33333333-3333-4333-8333-333333333333"
	// UnknownEventType はどの購読も受け取らないイベント種別です。
	UnknownEventType = "TST_UNKNOWN"
	// MalformedPayload は JSON として読めないイベントの内容です。
	MalformedPayload = "not-json"
)

// ErrProcessedEventStoreFailure は偽の処理済みイベントの保管先が保存に失敗したときに返すエラーです。
var ErrProcessedEventStoreFailure = errors.New("processed-event-store-failure")

type ownedCardKey struct {
	cardID string
	artNo  int64
}

// FakeGrantStore は処理済みイベント・カードパック・所持カードの偽の保管先と、トランザクション実行を兼ねます。
type FakeGrantStore struct {
	InsertErr       error
	packs           map[string]domain.CardPack
	processedEvents map[string]string
	ownedCards      map[string]map[ownedCardKey]int
}

// NewFakeGrantStore は指定のカードパックを持つ FakeGrantStore を生成します。
func NewFakeGrantStore(packs ...domain.CardPack) *FakeGrantStore {
	store := &FakeGrantStore{
		packs:           map[string]domain.CardPack{},
		processedEvents: map[string]string{},
		ownedCards:      map[string]map[ownedCardKey]int{},
	}
	for _, pack := range packs {
		store.packs[pack.PackID] = pack
	}
	return store
}

// SeedOwnedCard はプレイヤーの所持カードを指定の 1 種類だけにします。
func (s *FakeGrantStore) SeedOwnedCard(playerID, cardID string, count int) *FakeGrantStore {
	s.ownedCards[playerID] = map[ownedCardKey]int{{cardID: cardID, artNo: 0}: count}
	return s
}

// FailProcessedEventInsert は処理済みイベントの保存を失敗させます。
func (s *FakeGrantStore) FailProcessedEventInsert() *FakeGrantStore {
	s.InsertErr = ErrProcessedEventStoreFailure
	return s
}

// Insert はイベントを処理済みとして記録します。InsertErr があるときはそのエラーを返します。
func (s *FakeGrantStore) Insert(_ context.Context, eventID, eventType string) (bool, error) {
	_, isRecorded := s.processedEvents[eventID]
	s.processedEvents[eventID] = eventType
	return !isRecorded, s.InsertErr
}

// GetPack はパック ID のカードパックを返します。無いときは port.ErrNotFound を返します。
func (s *FakeGrantStore) GetPack(_ context.Context, packID string) (*domain.CardPack, error) {
	pack, isFound := s.packs[packID]
	if !isFound {
		return nil, port.ErrNotFound
	}
	return &pack, nil
}

// AddCards は所持カードの枚数を加算します。
func (s *FakeGrantStore) AddCards(_ context.Context, cards []domain.PlayerCard) error {
	for _, card := range cards {
		owned := s.ownedCards[card.PlayerID]
		if owned == nil {
			owned = map[ownedCardKey]int{}
			s.ownedCards[card.PlayerID] = owned
		}
		owned[ownedCardKey{cardID: card.CardID, artNo: card.ArtNo}] += card.Count
	}
	return nil
}

// GetPlayerCards はプレイヤーの所持カードをカード ID の昇順で返します。
func (s *FakeGrantStore) GetPlayerCards(_ context.Context, playerID string) ([]*domain.PlayerCard, error) {
	cards := []*domain.PlayerCard{}
	for key, count := range s.ownedCards[playerID] {
		cards = append(cards, &domain.PlayerCard{PlayerID: playerID, CardID: key.cardID, ArtNo: key.artNo, Count: count})
	}
	sort.Slice(cards, func(i, j int) bool { return cards[i].CardID < cards[j].CardID })
	return cards, nil
}

// RunInTransaction は fn を実行し、fn がエラーを返したときは保管先の変更を取り消します。
func (s *FakeGrantStore) RunInTransaction(_ context.Context, fn func(repositories port.GrantRepositories) error) error {
	savedProcessedEvents := maps.Clone(s.processedEvents)
	savedOwnedCards := map[string]map[ownedCardKey]int{}
	for playerID, owned := range s.ownedCards {
		savedOwnedCards[playerID] = maps.Clone(owned)
	}
	err := fn(port.GrantRepositories{ProcessedEvents: s, CardPacks: s, PlayerCards: s})
	if err != nil {
		s.processedEvents = savedProcessedEvents
		s.ownedCards = savedOwnedCards
	}
	return err
}

// ReadOwnedCardCounts は保管先から読み直したプレイヤーの所持枚数を、カード ID ごとに返します。
func ReadOwnedCardCounts(t *testing.T, store *FakeGrantStore, playerID string) map[string]int {
	t.Helper()
	cards, err := store.GetPlayerCards(context.Background(), playerID)
	require.NoError(t, err)
	counts := map[string]int{}
	for _, card := range cards {
		counts[card.CardID] += card.Count
	}
	return counts
}
