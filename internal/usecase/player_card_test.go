package usecase

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

// cardCacheWith は渡したカード定義だけを保持する CardCache を組み立てる。
func cardCacheWith(cards ...*domain.Card) *cache.CardCache {
	cc := cache.NewCardCache()
	for _, c := range cards {
		cc.InjectForTest(c.CardID, c)
	}
	return cc
}

func TestPlayerCardInteractor_GetPlayerCards(t *testing.T) {
	t.Run("[所持カード]所持カードの取得", func(t *testing.T) {
		t.Run("所持カードがあるとき、所持枚数はplayer_cards・カード定義はCardCacheから構成して返す", func(t *testing.T) {
			effectText := "対象を 1 体破壊する"
			def := &domain.Card{
				CardID: "TST-0001", CardName: "Fireball", ResourceLabel: "mana",
				Faction: "SHE", CardType: "spell", Resizable: true, Elastic: false,
				Stats: json.RawMessage(`{"atk":3}`), EffectText: &effectText, Restriction: "limited",
			}
			owned := &domain.PlayerCard{PlayerID: "player-1", CardID: def.CardID, ArtNo: 7, Count: 3}

			repo := newInMemoryPlayerCardRepo()
			repo.Seed(owned.PlayerID, []*domain.PlayerCard{owned})
			svc := NewPlayerCardInteractor(repo, cardCacheWith(def))

			got, err := svc.GetPlayerCards(context.Background(), owned.PlayerID)
			require.NoError(t, err)

			want := []*apicard.PlayerCardWithDef{{
				CardID: owned.CardID, ArtNo: owned.ArtNo, Count: owned.Count,
				CardName: def.CardName, ResourceLabel: def.ResourceLabel, Faction: def.Faction,
				CardType: def.CardType, Resizable: def.Resizable, Elastic: def.Elastic,
				Stats: def.Stats, EffectText: def.EffectText, Restriction: def.Restriction,
			}}
			assert.Equal(t, want, got)
		})

		t.Run("所持カードが無いとき、nilではなく空スライスを返す", func(t *testing.T) {
			svc := NewPlayerCardInteractor(newInMemoryPlayerCardRepo(), cardCacheWith())

			got, err := svc.GetPlayerCards(context.Background(), "player-1")
			require.NoError(t, err)
			assert.Equal(t, []*apicard.PlayerCardWithDef{}, got)
		})

		t.Run("所持カードがCardCacheに無いとき、黙ってスキップせず不足カードIDを含むエラーを返す", func(t *testing.T) {
			tests := []struct {
				name        string
				cache       []*domain.Card
				seed        []*domain.PlayerCard
				missingCard string
			}{
				{
					name:  "唯一の所持カードがcacheに無いとき、そのカードIDを含むエラーになる",
					cache: nil,
					seed: []*domain.PlayerCard{
						{PlayerID: "player-1", CardID: "TST-0001", ArtNo: 1, Count: 1},
					},
					missingCard: "TST-0001",
				},
				{
					name: "複数所持のうち1枚がcacheに無いとき、そのカードIDを含むエラーになる",
					cache: []*domain.Card{
						{CardID: "TST-0001", CardName: "Fireball"},
					},
					seed: []*domain.PlayerCard{
						{PlayerID: "player-1", CardID: "TST-0001", ArtNo: 1, Count: 1},
						{PlayerID: "player-1", CardID: "TST-9999", ArtNo: 1, Count: 1},
					},
					missingCard: "TST-9999",
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					pcRepo := newInMemoryPlayerCardRepo()
					pcRepo.Seed("player-1", tt.seed)
					svc := NewPlayerCardInteractor(pcRepo, cardCacheWith(tt.cache...))

					got, err := svc.GetPlayerCards(context.Background(), "player-1")
					require.Error(t, err)
					assert.ErrorContains(t, err, tt.missingCard)
					assert.Nil(t, got)
				})
			}
		})
	})
}
