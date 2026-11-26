package cache

import "errors"

var (
	// ErrNoCardDefinitions はカード定義の保管先に有効なカード定義が 1 件も無い場合に返します。
	ErrNoCardDefinitions = errors.New("no card definitions")
	// ErrNoProductDefinitions はプロダクト定義の保管先に有効なプロダクト定義が 1 件も無い場合に返します。
	ErrNoProductDefinitions = errors.New("no product definitions")
	// ErrNoInitiativeDefinitions は施策定義の保管先に有効な施策定義が 1 件も無い場合に返します。
	ErrNoInitiativeDefinitions = errors.New("no initiative definitions")
)
