package port

import "errors"

var (
	// ErrNotFound はリソースが存在しない場合に返します。
	ErrNotFound = errors.New("not found")
	// ErrUnowned はプレイヤーがカードまたはデッキを所持していない場合に返します。
	ErrUnowned = errors.New("unowned")
	// ErrInvalidDeck はデッキバリデーション違反（枚数不正、未知カード等）時に返します。
	ErrInvalidDeck = errors.New("invalid deck")
	// ErrRestrictionExceeded はカードの制限カテゴリによるコピー上限超過時に返します。
	ErrRestrictionExceeded = errors.New("restriction exceeded")
	// ErrInvalidArgument はリクエストパラメータのバリデーション失敗時に返します。
	ErrInvalidArgument = errors.New("invalid argument")
	// ErrEmptyPack は配布パックに対象カードが 1 件も登録されていない場合に返します。
	// pack マスターのデータ不整合 (例: card_pack 行はあるが card_pack_cards 行が 0 件)
	// を示す運用エラーです。
	ErrEmptyPack = errors.New("card pack has no cards")
	// ErrPackInactive は配布対象パックが is_active=false で運用停止中の場合に返します。
	// 呼び出し側はこのエラーを Nack 等で上位に伝搬し、DLQ で人間が運用順序の
	// 不整合 (例: shop product より先に card_pack を停止) を検知できるようにします。
	ErrPackInactive = errors.New("card pack is inactive")
)
