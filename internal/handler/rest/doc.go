// Package rest は card の REST 配信層と Cloud Pub/Sub push 受け口を提供する。
// gin handler、usecase の sentinel エラーを HTTP ステータスへ変換する責務、
// push 配信された Pub/Sub イベントをデコードして internal/adapter/pubsub の
// port.MessageHandler へ渡す責務を持つ。
package rest
