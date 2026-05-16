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

// TestPlayerCardInteractor_GetPlayerCards は所持カードに CardCache のカード定義を
// 付与し、所持順を保ったまま返す仕様を検証する。
func TestPlayerCardInteractor_GetPlayerCards(t *testing.T) {
	effect := "対象を 1 体破壊する"

	tests := []struct {
		name  string
		cache []*domain.Card
		seed  []*domain.PlayerCard
		want  []*apicard.PlayerCardWithDef
	}{
		{
			name: "所持カードに定義フィールドが付与され、所持順で返る",
			cache: []*domain.Card{
				{
					CardID: "SH-0001", CardName: "Fireball", ResourceLabel: "mana",
					Faction: "SHE", CardType: "spell", Resizable: true, Elastic: false,
					Stats: json.RawMessage(`{"atk":3}`), EffectText: &effect, Restriction: "limited",
				},
				{
					CardID: "SH-0002", CardName: "Shield", ResourceLabel: "mana",
					Faction: "SHE", CardType: "trap", Resizable: false, Elastic: true,
					Stats: json.RawMessage(`{"def":5}`), Restriction: "unlimited",
				},
			},
			seed: []*domain.PlayerCard{
				{PlayerID: "player-1", CardID: "SH-0001", ArtNo: 1, Count: 3},
				{PlayerID: "player-1", CardID: "SH-0002", ArtNo: 0, Count: 1},
			},
			want: []*apicard.PlayerCardWithDef{
				{
					CardID: "SH-0001", ArtNo: 1, Count: 3, CardName: "Fireball",
					ResourceLabel: "mana", Faction: "SHE", CardType: "spell",
					Resizable: true, Elastic: false, Stats: json.RawMessage(`{"atk":3}`),
					EffectText: &effect, Restriction: "limited",
				},
				{
					CardID: "SH-0002", ArtNo: 0, Count: 1, CardName: "Shield",
					ResourceLabel: "mana", Faction: "SHE", CardType: "trap",
					Resizable: false, Elastic: true, Stats: json.RawMessage(`{"def":5}`),
					Restriction: "unlimited",
				},
			},
		},
		{
			name:  "所持カード 0 件なら空スライス",
			cache: nil,
			seed:  nil,
			want:  []*apicard.PlayerCardWithDef{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pcRepo := newInMemoryPlayerCardRepo()
			pcRepo.Seed("player-1", tt.seed)
			svc := NewPlayerCardInteractor(pcRepo, cardCacheWith(tt.cache...))

			got, err := svc.GetPlayerCards(context.Background(), "player-1")
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestPlayerCardInteractor_GetPlayerCards_CacheMiss は CardCache に存在しない
// カードを所持している場合に、黙ってスキップせず内部エラーを返す仕様を検証する。
func TestPlayerCardInteractor_GetPlayerCards_CacheMiss(t *testing.T) {
	tests := []struct {
		name        string
		cache       []*domain.Card
		seed        []*domain.PlayerCard
		missingCard string
	}{
		{
			name:  "唯一の所持カードが cache に無い",
			cache: nil,
			seed: []*domain.PlayerCard{
				{PlayerID: "player-1", CardID: "SH-0001", ArtNo: 1, Count: 1},
			},
			missingCard: "SH-0001",
		},
		{
			name: "複数所持のうち 1 枚が cache に無い",
			cache: []*domain.Card{
				{CardID: "SH-0001", CardName: "Fireball"},
			},
			seed: []*domain.PlayerCard{
				{PlayerID: "player-1", CardID: "SH-0001", ArtNo: 1, Count: 1},
				{PlayerID: "player-1", CardID: "SH-9999", ArtNo: 1, Count: 1},
			},
			missingCard: "SH-9999",
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
}
