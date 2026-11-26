package port

import "errors"

var (
	// ErrNotFound は対象が存在しない場合に返します。
	ErrNotFound = errors.New("not found")
	// ErrUnowned はプレイヤーがカードまたはデッキを所持していない場合に返します。
	ErrUnowned = errors.New("unowned")
	// ErrInvalidDeck はデッキの構成が不正な場合 (枚数の不一致・未知のカードなど) に返します。
	ErrInvalidDeck = errors.New("invalid deck")
	// ErrRestrictionExceeded はカードの制限区分によるコピー上限を超えた場合に返します。
	ErrRestrictionExceeded = errors.New("restriction exceeded")
	// ErrEmptyPack は配布パックに配布対象のカードが 1 件も無い場合に返します。
	ErrEmptyPack = errors.New("card pack has no cards")
	// ErrPackInactive は配布パックが運用停止中の場合に返します。
	ErrPackInactive = errors.New("card pack is inactive")
)
