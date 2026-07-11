package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
)

type ownershipExpectation struct {
	cardID  string
	isOwned bool
}

func TestListCardsWithOwnership(t *testing.T) {
	t.Run("所持状態付きカード一覧", func(t *testing.T) {
		tests := []struct {
			name  string
			cards map[string]*domain.Card
			seed  []*domain.PlayerCard
			want  []ownershipExpectation
		}{
			{
				name: "一部のカードを所持するとき、所持フラグが対応して返る",
				cards: map[string]*domain.Card{
					"TST-0001": {CardID: "TST-0001", CardName: "Fireball"},
					"TST-0002": {CardID: "TST-0002", CardName: "Shield"},
					"TST-0003": {CardID: "TST-0003", CardName: "Heal"},
				},
				seed: []*domain.PlayerCard{
					{PlayerID: "p1", CardID: "TST-0001", ArtNo: 1, Count: 1},
					{PlayerID: "p1", CardID: "TST-0003", ArtNo: 1, Count: 2},
				},
				want: []ownershipExpectation{
					{"TST-0001", true},
					{"TST-0002", false},
					{"TST-0003", true},
				},
			},
			{
				name:  "マスターにカードが無いとき、空になる",
				cards: map[string]*domain.Card{},
				seed:  nil,
				want:  nil,
			},
			{
				name: "所持カードが無いとき、全て未所持で返る",
				cards: map[string]*domain.Card{
					"TST-0001": {CardID: "TST-0001", CardName: "Fireball"},
					"TST-0002": {CardID: "TST-0002", CardName: "Shield"},
				},
				seed: nil,
				want: []ownershipExpectation{
					{"TST-0001", false},
					{"TST-0002", false},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				pcRepo := newInMemoryPlayerCardRepo()
				pcRepo.Seed("p1", tt.seed)
				svc := NewCardInteractor(newInMemoryCardRepo(tt.cards), pcRepo)

				got, err := svc.ListCardsWithOwnership(context.Background(), "p1")
				require.NoError(t, err)
				require.Len(t, got, len(tt.want))
				for i, w := range tt.want {
					assert.Equal(t, w.cardID, got[i].CardID)
					assert.Equal(t, w.isOwned, got[i].IsOwned)
				}
			})
		}
	})
}

func TestListCards(t *testing.T) {
	t.Run("カード一覧", func(t *testing.T) {
		t.Run("マスターにカードがあるとき、card_id 昇順で全件返る", func(t *testing.T) {
			cards := map[string]*domain.Card{
				"TST-0001": {CardID: "TST-0001", CardName: "Fireball"},
				"TST-0002": {CardID: "TST-0002", CardName: "Shield"},
			}
			svc := NewCardInteractor(newInMemoryCardRepo(cards), newInMemoryPlayerCardRepo())

			result, err := svc.ListCards(context.Background())
			require.NoError(t, err)

			gotIDs := make([]string, len(result))
			for i, c := range result {
				gotIDs[i] = c.CardID
			}
			assert.Equal(t, []string{"TST-0001", "TST-0002"}, gotIDs)
		})
	})
}
