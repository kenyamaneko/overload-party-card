# Card サービス設計

このドキュメントは Card サービスの内部動作を説明する。サービスの概要・エンドポイント・環境変数は [README.md](../README.md) を参照。

## カードデータ SSoT フロー

```
data/cards/*.yaml  (SSoT)
  │
  ▼
scripts/generate_cards.py → db/seed/cards_seed.sql
  │
  ▼
psqldef (ops/db-migrate) → card.card_definitions テーブル
  │
  ▼
CardCache (起動時 DB から全件ロード) → サービス内メモリ
```

- `data/cards/*.yaml` がカードデータの SSoT
- `generate_cards.py` が YAML から SQL seed と `cards_gen.json` を生成
- `card_definitions` テーブルの内容は CardCache に起動時にロードされ、以後はメモリから読み取り
- CardCache ロード失敗時はサービスが起動しない (fail-fast)

## インメモリ CardCache

`cache.NewCardCache()` は起動時に `cardRepo` (PostgreSQL) から全カード定義を読み込む。以後はリクエストごとの DB アクセスなしにカード定義を参照する。

- `Load(ctx, cardRepo)`: 起動時 1 回のみ呼び出し。失敗は fatal
- `Get(cardId)`: O(1) map lookup
- `All()`: 全件の slice を返す
- `Count()`: カード総数

CardCache は読み取り専用。カードマスター更新時は Pod 再起動で invalidation する (将来的に Pub/Sub invalidation を計画中)。

## デッキバリデーション

### is_valid の算出

デッキの `is_valid` はレスポンス生成時に都度算出する。DB には保存しない。

1. 総枚数がちょうど 30 枚であること
2. 構成カード全てをプレイヤーが所持していること (`player_cards.count >= deck_cards.count`)
3. 各カードが `card_definitions.restriction` の上限枚数以内であること
   - `unlimited`: 3 枚まで
   - `semi_limited`: 2 枚まで
   - `limited`: 1 枚まで
   - `forbidden`: 0 枚 (デッキに入れられない)

所持カードの変動や制限改定を即座に反映するため、キャッシュは行わない。

### validate-for-battle

`POST .../validate-for-battle` はバトル開始前のゲートキーパー。is_valid と同等のチェックを行い、不可の場合は 400 + 具体的な理由を返す。Gateway は matchmaking enqueue / NPC 対戦作成の前にこのエンドポイントを呼び出す。

## カードパック配布

### grant-initial-pack (初回ファクション選択)

配布対象: **選択 faction のカード全種類 x3 + Neutral のカード全種類 x3**

- `card_definitions` から `faction` カラムで動的にフィルタ (ハードコードなし)
- `player_cards` に UPSERT (加算)。既に所持済みでも 3 枚追加
- 単一トランザクション

### grant-faction-pack (ショップ購入)

配布対象: **選択 faction のカード全種類 x3 のみ** (Neutral は含めない)

- 初回配布の Neutral を除外した点以外は grant-initial-pack と同じロジック

### 冪等性の方針

本 API 自体は「呼ばれたら必ず 3 枚ずつ加算する」セマンティクス。冪等性は呼び出し側 (gateway のオーケストレーション、または Pub/Sub subscriber の event dedup) が担保する。

## Pub/Sub Subscriber

### faction-selected

| 項目 | 値 |
|---|---|
| Subscription | `faction-selected-card-sub` |
| ペイロード型 | `FactionSelectedEvent` (`common/packages/pubsub-events` で定義) |
| 冪等性 | `card.processed_events` テーブルで event_id による重複排除 |

処理フロー:

1. JSON デシリアライズ + `event_type == "faction_selected"` チェック
2. `processed_events` に event_id を INSERT (既存なら ack して return)
3. `source` フィールドで配布メソッドを分岐:
   - `scenario_initial` → `GrantInitialPack` (faction + Neutral)
   - `shop_purchase` → `GrantFactionPack` (faction のみ)
4. 配布成功 → ack、失敗 → nack (Pub/Sub がリトライ)

## エラーハンドリング

- `400 INVALID_ARGUMENT`: パラメータ不正 (UUID 形式エラー、未知の faction 等)
- `400 DECK_INVALID_CARD_COUNT`: 総枚数が 30 枚でない
- `400 DECK_CARD_NOT_OWNED`: 未所持カードがデッキに含まれている
- `400 DECK_CARD_LIMIT_EXCEEDED`: restriction 制限超過
- `404 DECK_NOT_FOUND / PLAYER_NOT_FOUND`: リソース不存在
- `500 INTERNAL_ERROR`: DB 接続失敗等の内部エラー

## データ構造

### card スキーマ

| テーブル | 用途 |
|---|---|
| `card_definitions` | カードマスター (YAML seed から投入) |
| `player_cards` | プレイヤーごとのカード所持枚数 |
| `decks` | デッキメタ (name / playmat / sleeve) |
| `deck_cards` | デッキ構成カード (ON DELETE CASCADE) |
| `processed_events` | Pub/Sub イベント重複排除用 |
