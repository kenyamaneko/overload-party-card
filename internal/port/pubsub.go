package port

import "context"

// MessageHandler は 1 件の Pub/Sub イベント payload (デコード済み bytes) の
// 処理結果を ack / nack の形で返す。nil = ack (正常処理 or 責務外として
// ACK できるケース)、非 nil = nack (ペイロード不正や副作用失敗で再配信を
// 要するケース)。
//
// 戻り値の意味論を「error か否か」に一本化することで、呼び出し側 (push
// delivery の HTTP handler) は単一戻り値を見て応答の成否 (2xx / 5xx) を
// 決められる。
type MessageHandler = func(ctx context.Context, data []byte) error
