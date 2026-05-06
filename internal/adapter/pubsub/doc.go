// Package pubsub は card サービスの Pub/Sub subscriber を管理します。
//
// player-onboarded-card-sub を購読し、オンボード完了プレイヤーへ
// basic pack と選択 faction の基本セットを GrantInteractor 経由で配布します。
// card-pack-purchased-card-sub を購読し、shop で購入された card_pack を
// 同じく GrantInteractor 経由で配布します。
//
// 冪等性は event_id ベースの processed_events で担保します。
// 未知の event_type / malformed payload は Ack ではなく Nack します。
// リトライが無意味なケースは DLQ (player-onboarded-dlq / card-pack-purchased-dlq)
// が拾うため、subscriber 側で握りつぶさずインフラに委ねる方針です。
package pubsub
