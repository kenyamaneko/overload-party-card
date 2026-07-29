// Package pubsub は card サービスが受信する Pub/Sub イベントの処理ロジックを
// 管理します。配信自体は HTTP push (internal/handler/rest の受け口) が担い、
// 本パッケージはデコード済み payload を受け取って処理する port.MessageHandler
// を提供します。
//
// player-onboarded-card-sub のイベントを処理し、オンボード完了プレイヤーへ
// basic pack と選択 faction の基本セットを GrantInteractor 経由で配布します。
// card-pack-purchased-card-sub のイベントを処理し、shop で購入された card_pack を
// 同じく GrantInteractor 経由で配布します。
//
// 冪等性は event_id ベースの processed_events で担保します。
// 未知の event_type / malformed payload は ack ではなく nack 相当のエラーを返します。
// リトライが無意味なケースは DLQ (player-onboarded-dlq / card-pack-purchased-dlq)
// が拾うため、subscriber 側で握りつぶさずインフラに委ねる方針です。
package pubsub
