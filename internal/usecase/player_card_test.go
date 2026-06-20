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

// TestPlayerCardInteractor_GetPlayerCards_MergesOwnershipAndDefinition は
// 所持カードの応答が「所持枚数は player_cards 由来・カード定義は CardCache 由来」で
// 構成される仕様 (FEATURE_SPEC §3.1) を検証する。
func TestPlayerCardInteractor_GetPlayerCards_MergesOwnershipAndDefinition(t *testing.T) {
	effectText := "対象を 1 体破壊する"
	def := &domain.Card{
		CardID: "SH-0001", CardName: "Fireball", ResourceLabel: "mana",
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
}

// TestPlayerCardInteractor_GetPlayerCards_EmptyWhenNoneOwned は所持カードが
// 無いプレイヤーに対し nil ではなく空スライスを返す仕様を検証する。
func TestPlayerCardInteractor_GetPlayerCards_EmptyWhenNoneOwned(t *testing.T) {
	svc := NewPlayerCardInteractor(newInMemoryPlayerCardRepo(), cardCacheWith())

	got, err := svc.GetPlayerCards(context.Background(), "player-1")
	require.NoError(t, err)
	assert.Equal(t, []*apicard.PlayerCardWithDef{}, got)
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
