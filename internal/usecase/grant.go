package usecase

import (
	"context"
	"fmt"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// GrantInteractor はカードパック配布を管理します。
// 配布対象 (どのカードを何枚) は card_pack マスター (port.CardPackRepo) で定義し、
// 業務文脈 (initial / 購入 / 限定) は呼び出し側 (subscriber) が pack_id で表現します。
// 読み取り専用の CardInteractor を汚さないよう書き込みパスを分離しています。
type GrantInteractor struct {
	cardPackRepo   port.CardPackRepo
	cardRepo       port.CardRepo
	playerCardRepo port.PlayerCardRepo
}

// NewGrantInteractor は GrantInteractor を生成します。
func NewGrantInteractor(
	cardPackRepo port.CardPackRepo,
	cardRepo port.CardRepo,
	playerCardRepo port.PlayerCardRepo,
) *GrantInteractor {
	return &GrantInteractor{
		cardPackRepo:   cardPackRepo,
		cardRepo:       cardRepo,
		playerCardRepo: playerCardRepo,
	}
}

// GrantPack は card_pack マスターから対象 pack を取得し、selection に従って
// プレイヤーへカードを配布します。pack マスターは配布リクエストごとに DB を引きます
// (キャッシュなし: ADR-032 §2 / docs/ARCHITECTURE.md「pack マスターはキャッシュしない」)。
//
// 戻り値は配布されたコピー総数。pack 不在は port.ErrNotFound、運用停止 pack は
// port.ErrPackInactive、selection が不正は port.ErrInvalidPackSelection を返します。
func (s *GrantInteractor) GrantPack(ctx context.Context, playerID, packID string) (int, error) {
	pack, err := s.cardPackRepo.GetPack(ctx, packID)
	if err != nil {
		return 0, fmt.Errorf("get card_pack %q: %w", packID, err)
	}
	if !pack.IsActive {
		return 0, fmt.Errorf("%w: pack %q", port.ErrPackInactive, packID)
	}
	cardIDs, err := s.resolveSelection(ctx, pack.Selection)
	if err != nil {
		return 0, fmt.Errorf("resolve selection for pack %q: %w", packID, err)
	}
	if len(cardIDs) == 0 {
		return 0, fmt.Errorf("%w: pack %q resolved 0 cards", port.ErrCardMasterEmpty, packID)
	}
	return s.playerCardRepo.AddCards(ctx, playerID, cardIDs, pack.CopiesPerCard)
}

// resolveSelection は Selection 実装ごとに対象 card_id 集合を解決します。
// 未知の Selection 実装が来た場合は port.ErrInvalidPackSelection を返し、
// 握りつぶさず明示エラーで上に伝搬します。
func (s *GrantInteractor) resolveSelection(ctx context.Context, sel domain.Selection) ([]string, error) {
	switch sel := sel.(type) {
	case domain.SelectionByFactions:
		ids, err := s.cardRepo.FindCardIDsByFactions(ctx, sel.Factions)
		if err != nil {
			return nil, fmt.Errorf("find card ids by factions: %w", err)
		}
		return ids, nil
	case domain.SelectionByCardIDs:
		return sel.CardIDs, nil
	}
	return nil, fmt.Errorf("%w: unknown selection type %T", port.ErrInvalidPackSelection, sel)
}

