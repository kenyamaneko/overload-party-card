package usecase

import (
	"context"
	"fmt"

	gamedesign "github.com/kenyamaneko/overload-party-common/packages/game-design-constants"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// copiesPerGrant は配布カード1種あたりの付与コピー数です。
const copiesPerGrant = 3

// GrantInteractor はカードパック配布（初期ファクション選択・ショップ購入）を管理します。
// 読み取り専用の CardInteractor を汚さないよう書き込みパスを分離しています。
type GrantInteractor struct {
	cardRepo       port.CardRepo
	playerCardRepo port.PlayerCardRepo
}

// NewGrantInteractor は GrantInteractor を生成します。
func NewGrantInteractor(cardRepo port.CardRepo, playerCardRepo port.PlayerCardRepo) *GrantInteractor {
	return &GrantInteractor{cardRepo: cardRepo, playerCardRepo: playerCardRepo}
}

// GrantInitialPack は選択ファクション + Neutral のカード各3枚をプレイヤーに配布します。
// 付与されたコピー総数を返します。
func (s *GrantInteractor) GrantInitialPack(ctx context.Context, playerID, faction string) (int, error) {
	if err := validateSelectableFaction(faction); err != nil {
		return 0, err
	}
	factions := []string{faction, gamedesign.FactionNeutral}
	cardIDs, err := s.cardRepo.FindCardIDsByFactions(ctx, factions)
	if err != nil {
		return 0, fmt.Errorf("find card ids for initial pack: %w", err)
	}
	if len(cardIDs) == 0 {
		return 0, fmt.Errorf("%w: factions %v", port.ErrCardMasterEmpty, factions)
	}
	return s.playerCardRepo.AddCards(ctx, playerID, cardIDs, copiesPerGrant)
}

// GrantFactionPack は選択ファクションのカードのみ（Neutral なし）各3枚を配布します。
// ショップでの faction_set 購入後に呼ばれます。
func (s *GrantInteractor) GrantFactionPack(ctx context.Context, playerID, faction string) (int, error) {
	if err := validateSelectableFaction(faction); err != nil {
		return 0, err
	}
	factions := []string{faction}
	cardIDs, err := s.cardRepo.FindCardIDsByFactions(ctx, factions)
	if err != nil {
		return 0, fmt.Errorf("find card ids for faction pack: %w", err)
	}
	if len(cardIDs) == 0 {
		return 0, fmt.Errorf("%w: faction %q", port.ErrCardMasterEmpty, faction)
	}
	return s.playerCardRepo.AddCards(ctx, playerID, cardIDs, copiesPerGrant)
}

func validateSelectableFaction(faction string) error {
	for _, f := range gamedesign.SelectableFactions {
		if f == faction {
			return nil
		}
	}
	return fmt.Errorf("%w: faction %q is not one of %v", port.ErrInvalidArgument, faction, gamedesign.SelectableFactions)
}
