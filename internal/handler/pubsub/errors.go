package pubsub

import "errors"

var (
	// ErrMalformedPayload はイベントの内容を読み取れない場合に返します。
	ErrMalformedPayload = errors.New("bad payload")
	// ErrUnexpectedEventType はイベントの種別が、その購読が受け取る種別と異なる場合に返します。
	ErrUnexpectedEventType = errors.New("unexpected event_type")
)
