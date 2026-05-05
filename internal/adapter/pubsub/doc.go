// Package pubsub は card サービスの Pub/Sub subscriber を管理します。
//
// faction-purchased-card-sub を購読し、shop でのファクション購入時に
// 対象 faction のカードのみ (Neutral 無し) を GrantInteractor 経由で配布します。
// 初期パック (faction + Neutral) の配布は player-onboarded-card-sub が担当します。
//
// 冪等性は event_id ベースの processed_events で担保します。
// 未知の event_type / malformed payload は Ack ではなく Nack します。
// リトライが無意味なケースは DLQ (faction-purchased-dlq) が拾うため、
// subscriber 側で握りつぶさずインフラに委ねる方針です。
package pubsub
