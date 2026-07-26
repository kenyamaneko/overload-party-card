package usecase

import (
	"context"
	"fmt"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

// PlayerCardInteractor はプレイヤーの所持カード照会を提供します。
// デッキ CRUD とは責務が独立するため DeckInteractor から分離しています。
type PlayerCardInteractor struct {
	playerCardRepo port.PlayerCardRepo
	cardCache      *cache.CardCache
}

// NewPlayerCardInteractor は PlayerCardInteractor を生成します。
func NewPlayerCardInteractor(playerCardRepo port.PlayerCardRepo, cardCache *cache.CardCache) *PlayerCardInteractor {
	return &PlayerCardInteractor{playerCardRepo: playerCardRepo, cardCache: cardCache}
}

// GetPlayerCards はプレイヤーの所持カードにカード定義を付与して返します。
// CardCache に存在しないカードを所持している状態は DB 整合性異常なので、
// 黙ってスキップせず内部エラーとして返します。
func (s *PlayerCardInteractor) GetPlayerCards(ctx context.Context, playerID string) ([]*apicard.PlayerCardWithDef, error) {
	pcs, err := s.playerCardRepo.GetPlayerCards(ctx, playerID)
	if err != nil {
		return nil, err
	}

	result := make([]*apicard.PlayerCardWithDef, 0, len(pcs))
	for _, pc := range pcs {
		cd := s.cardCache.Get(pc.CardID)
		if cd == nil {
			return nil, fmt.Errorf("player %s owns card %s but it is missing from the card cache; refresh cache or investigate inconsistent state", playerID, pc.CardID)
		}
		result = append(result, &apicard.PlayerCardWithDef{
			CardID:        pc.CardID,
			ArtNo:         pc.ArtNo,
			Count:         pc.Count,
			CardName:      cd.CardName,
			ResourceLabel: cd.ResourceLabel,
			Faction:       cd.Faction,
			CardType:      cd.CardType,
			Resizable:     cd.Resizable,
			Elastic:       cd.Elastic,
			Stats:         cd.Stats,
			EffectText:    cd.EffectText,
			Restriction:   cd.Restriction,
		})
	}
	return result, nil
}
