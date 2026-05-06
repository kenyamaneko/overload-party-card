package usecase

import (
	"context"
	"fmt"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// GrantInteractor はカードパック配布を管理します。
// 配布対象 (どのカードを何枚) は card_pack マスター (port.CardPackRepo) で定義し、
// 業務文脈 (initial / 購入 / 限定) は呼び出し側 (subscriber) が pack_id で表現します。
// 読み取り専用の CardInteractor を汚さないよう書き込みパスを分離しています。
type GrantInteractor struct {
	cardPackRepo   port.CardPackRepo
	playerCardRepo port.PlayerCardRepo
}

// NewGrantInteractor は GrantInteractor を生成します。
func NewGrantInteractor(
	cardPackRepo port.CardPackRepo,
	playerCardRepo port.PlayerCardRepo,
) *GrantInteractor {
	return &GrantInteractor{
		cardPackRepo:   cardPackRepo,
		playerCardRepo: playerCardRepo,
	}
}

// GrantPack は card_pack マスターから対象 pack を取得し、内包カードをそのまま
// プレイヤーへ配布します。
//
// 戻り値は配布されたコピー総数。pack 不在は port.ErrNotFound、運用停止 pack は
// port.ErrPackInactive、内包カードが 0 件は port.ErrEmptyPack を返します。
func (s *GrantInteractor) GrantPack(ctx context.Context, playerID, packID string) (int, error) {
	pack, err := s.cardPackRepo.GetPack(ctx, packID)
	if err != nil {
		return 0, fmt.Errorf("get card_pack %q: %w", packID, err)
	}
	if !pack.IsActive {
		return 0, fmt.Errorf("%w: pack %q", port.ErrPackInactive, packID)
	}
	if len(pack.Cards) == 0 {
		return 0, fmt.Errorf("%w: pack %q", port.ErrEmptyPack, packID)
	}
	return s.playerCardRepo.AddCards(ctx, playerID, pack.Cards)
}
