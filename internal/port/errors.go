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
	// ErrCardMasterEmpty は配布対象 faction に有効なカードが 1 件も存在しない場合に
	// 返します。カードマスターの不整合を示す運用エラーです。
	ErrCardMasterEmpty = errors.New("card master empty for requested factions")
)
