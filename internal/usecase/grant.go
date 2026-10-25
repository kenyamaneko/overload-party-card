package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
)

const (
	basicPackID            = "basic"
	factionSetPackIDPrefix = "faction_set_"
)

// GrantInteractor はイベントに応じたカードパックの配布を提供します。
type GrantInteractor struct {
	transactionRunner port.GrantTransactionRunner
}

// NewGrantInteractor は GrantInteractor を生成します。
func NewGrantInteractor(transactionRunner port.GrantTransactionRunner) *GrantInteractor {
	return &GrantInteractor{transactionRunner: transactionRunner}
}

// GrantInitialPacks はオンボーディングを完了したプレイヤーへ、基本パックと初期陣営の陣営セットを配布します。
// 配布したカードの合計枚数を返します。処理済みのイベントのときは何も配布せず 0 を返します。
func (gi *GrantInteractor) GrantInitialPacks(ctx context.Context, event domain.PlayerOnboardedEvent) (int, error) {
	// 陣営セットのパック ID は、陣営名を小文字にした形で定義されているため、小文字にして組み立てる。
	factionSetPackID := factionSetPackIDPrefix + strings.ToLower(event.InitialFactionID)
	return gi.grantPacksOncePerEvent(ctx, event.EventID, event.EventType, event.PlayerID, []string{basicPackID, factionSetPackID})
}

// GrantPurchasedPack は購入されたカードパックをプレイヤーへ配布します。
// 配布したカードの合計枚数を返します。処理済みのイベントのときは何も配布せず 0 を返します。
func (gi *GrantInteractor) GrantPurchasedPack(ctx context.Context, event domain.CardPackPurchasedEvent) (int, error) {
	return gi.grantPacksOncePerEvent(ctx, event.EventID, event.EventType, event.PlayerID, []string{event.CardPackID})
}

// grantPacksOncePerEvent はイベントを処理済みとして記録し、指定のパックをすべて配布します。
// 記録と配布は 1 つのトランザクションで行い、記録済みのイベントのときは何も配布しません。
func (gi *GrantInteractor) grantPacksOncePerEvent(ctx context.Context, eventID, eventType, playerID string, packIDs []string) (int, error) {
	totalCopies := 0
	err := gi.transactionRunner.RunInTransaction(ctx, func(repositories port.GrantRepositories) error {
		isInserted, err := repositories.ProcessedEvents.Insert(ctx, eventID, eventType)
		if err != nil {
			return fmt.Errorf("insert processed event %q: %w", eventID, err)
		}
		if !isInserted {
			return nil
		}

		for _, packID := range packIDs {
			copies, err := grantPack(ctx, repositories, playerID, packID)
			if err != nil {
				return fmt.Errorf("grant pack %q: %w", packID, err)
			}
			totalCopies += copies
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return totalCopies, nil
}

// grantPack はパックの内包カードをそのままプレイヤーの所持カードに加え、加えた枚数の合計を返します。
// パックが存在しないときは port.ErrNotFound、運用停止中のときは port.ErrPackInactive、内包カードが 0 件のときは port.ErrEmptyPack を返します。
func grantPack(ctx context.Context, repositories port.GrantRepositories, playerID, packID string) (int, error) {
	pack, err := repositories.CardPacks.GetPack(ctx, packID)
	if err != nil {
		return 0, fmt.Errorf("get card_pack %q: %w", packID, err)
	}
	if !pack.IsActive {
		return 0, fmt.Errorf("%w: pack %q", port.ErrPackInactive, packID)
	}
	if len(pack.Cards) == 0 {
		return 0, fmt.Errorf("%w: pack %q", port.ErrEmptyPack, packID)
	}

	grantedCards := make([]domain.PlayerCard, len(pack.Cards))
	totalCopies := 0
	for i, packCard := range pack.Cards {
		grantedCards[i] = domain.PlayerCard{
			PlayerID: playerID,
			CardID:   packCard.CardID,
			ArtNo:    domain.GrantedCardArtNo,
			Count:    packCard.Copies,
		}
		totalCopies += packCard.Copies
	}
	if err := repositories.PlayerCards.AddCards(ctx, grantedCards); err != nil {
		return 0, fmt.Errorf("add cards: %w", err)
	}
	return totalCopies, nil
}
