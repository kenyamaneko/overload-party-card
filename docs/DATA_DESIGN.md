# card スキーマ - データ設計

> **DDL の SSoT:** `db/schema.sql`

## 設計概要

card スキーマはカードマスターデータ・プレイヤーの所持カード・デッキ構築を管理する。battle サービスはカードマスターを直接 SELECT せず、card サービスの内部 REST API `GET /internal/v1/cards` でインメモリキャッシュに取得する。

---

## テーブル構成

### card_definitions

カードマスター。全 126 枚のカード定義を格納する。

- **PK:** `card_id` (VARCHAR(10))
- **INDEX:** `idx_cards_faction` ON `(faction, card_type)`, `idx_cards_type` ON `(card_type)`
- **TRIGGER:** `updated_at` 自動更新

<!-- BEGIN GENERATED: card_definitions -->
| カラム名 | 型 | Nullable | 説明 |
|---|---|---|---|
| `card_id` | VARCHAR(10) | No | カード識別子（例: SH-0001） |
| `card_name` | VARCHAR(100) | No | カード名 |
| `resource_label` | VARCHAR(30) | No | リソースラベル |
| `faction` | VARCHAR(20) | No | 陣営（SHE / Tenki / Sugar / Tuners / Neutral） |
| `card_type` | VARCHAR(30) | No | カードタイプ（Resource / Support） |
| `subtype` | VARCHAR(30) | Yes | サブタイプ（Compute/Data カテゴリのみ設定: VM/Container/Database 等） |
| `resizable` | BOOLEAN | No | Resizable 属性 |
| `elastic` | BOOLEAN | No | Elastic 属性 |
| `stats` | JSONB | No | ステータス定義 |
| `effect_text` | VARCHAR(500) | Yes | 効果テキスト（表示用） |
| `effects` | JSONB | Yes | 効果定義（JSON 配列） |
| `restriction` | VARCHAR(20) | No | 制限区分（unlimited / semi_limited / limited / forbidden） |
| `is_active` | BOOLEAN | No | 有効フラグ |
| `created_at` | TIMESTAMPTZ | No | 作成日時 |
| `updated_at` | TIMESTAMPTZ | No | 更新日時 |
<!-- END GENERATED: card_definitions -->

**`stats` JSON の構造:**

コンピュート系リソース:
```json
{
  "throughput": 300,
  "availability": 500,
  "maintenance_cost": 100,
  "free_tier": null,
  "cost_per_request": null,
  "sla_penalty": 200
}
```

DB系・オブジェクトストレージ:
```json
{
  "yield": 200,
  "availability": 400,
  "maintenance_cost": 80,
  "free_tier": null,
  "cost_per_request": null,
  "sla_penalty": 150
}
```

Platform / Attachment / Strategy / Incident / Reactive には `stats` フィールドなし。

### products

陣営に属するプロダクトマスター。陣営:プロダクト = 1:N。

- **PK:** `product_id` (VARCHAR(10))
- **INDEX:** `idx_products_faction` ON `(faction)`
- **TRIGGER:** `updated_at` 自動更新

<!-- BEGIN GENERATED: products -->
| カラム名 | 型 | Nullable | 説明 |
|---|---|---|---|
| `product_id` | VARCHAR(10) | No | プロダクト識別子（例: PD-0001） |
| `faction` | VARCHAR(20) | No | 所属陣営 |
| `product_name` | VARCHAR(100) | No | プロダクト名 |
| `is_active` | BOOLEAN | No | 有効フラグ（論理削除） |
| `created_at` | TIMESTAMPTZ | No | 作成日時 |
| `updated_at` | TIMESTAMPTZ | No | 更新日時 |
<!-- END GENERATED: products -->

### initiatives

プロダクトの施策マスター。プロダクト:施策 = 1:N。`kind` で routine（1ターン1回）/ special（1ゲーム1回）を区別する。

- **PK:** `initiative_id` (VARCHAR(10))
- **FK:** `product_id` → `card.products(product_id)` ON DELETE CASCADE
- **INDEX:** `idx_initiatives_product` ON `(product_id)`
- **TRIGGER:** `updated_at` 自動更新

<!-- BEGIN GENERATED: initiatives -->
| カラム名 | 型 | Nullable | 説明 |
|---|---|---|---|
| `initiative_id` | VARCHAR(10) | No | 施策識別子（例: IN-0001） |
| `product_id` | VARCHAR(10) | No | 親プロダクト識別子 |
| `kind` | VARCHAR(10) | No | 区分（routine: 1ターン1回 / special: 1ゲーム1回） |
| `name` | VARCHAR(100) | No | 施策名 |
| `insight_cost` | BIGINT | No | 発動 Insight コスト |
| `effect_text` | VARCHAR(500) | No | 効果テキスト（表示用） |
| `effect` | JSONB | No | 効果定義（DSL） |
| `is_active` | BOOLEAN | No | 有効フラグ（論理削除） |
| `created_at` | TIMESTAMPTZ | No | 作成日時 |
| `updated_at` | TIMESTAMPTZ | No | 更新日時 |
<!-- END GENERATED: initiatives -->

### player_cards

プレイヤーの所持カード。

- **PK:** `(player_id, card_id, art_no)`
- `player_id` は `account.players` へのクロススキーマ参照（FK 無し、アプリ層整合性）

<!-- BEGIN GENERATED: player_cards -->
| カラム名 | 型 | Nullable | 説明 |
|---|---|---|---|
| `player_id` | UUID | No | 所有プレイヤー (cross-schema reference to account.players; app-level integrity, not enforced by FK) |
| `card_id` | VARCHAR(10) | No | カード識別子 |
| `art_no` | BIGINT | No | アート番号 |
| `count` | INT | No | 所持枚数 |
<!-- END GENERATED: player_cards -->

### decks

デッキヘッダー。

- **PK:** `(player_id, deck_id)`
- **INDEX:** `idx_decks_player` ON `(player_id, updated_at DESC)`
- **TRIGGER:** `updated_at` 自動更新
- `player_id` は `account.players` へのクロススキーマ参照（FK 無し）

<!-- BEGIN GENERATED: decks -->
| カラム名 | 型 | Nullable | 説明 |
|---|---|---|---|
| `player_id` | UUID | No | 所有プレイヤー (cross-schema reference to account.players; app-level integrity, not enforced by FK) |
| `deck_id` | BIGINT (IDENTITY) | No | デッキID（自動採番） |
| `deck_name` | VARCHAR(50) | No | デッキ名 |
| `faction` | VARCHAR(20) | No | 宣言陣営（SHE / Tenki / Sugar / Tuners） |
| `product_id` | VARCHAR(10) | No | 選択したプロダクトの ID（宣言陣営に属する） |
| `routine_id` | VARCHAR(10) | No | セットしたルーチン施策の ID（選択プロダクトに属する） |
| `special_id` | VARCHAR(10) | No | セットしたスペシャル施策の ID（選択プロダクトに属する） |
| `playmat_no` | BIGINT | Yes | プレイマット番号（NULL: デフォルト） |
| `sleeve_no` | BIGINT | Yes | スリーブ番号（NULL: デフォルト） |
| `created_at` | TIMESTAMPTZ | No | 作成日時 |
| `updated_at` | TIMESTAMPTZ | No | 更新日時 |
<!-- END GENERATED: decks -->

**設計判断:**
- `is_valid` カラムは意図的に持たない。所持カード変更や制限改定に追従するため、API レスポンス時にサービス層が都度算出する
- `playmat_no` / `sleeve_no` をデッキに直接持たせているのは、対戦開始時にデッキ情報と合わせて 1 クエリで取得するため

### deck_cards

デッキ内カード構成。

- **PK:** `(player_id, deck_id, card_id, art_no)`
- **FK:** `(player_id, deck_id)` → `decks` (CASCADE)

<!-- BEGIN GENERATED: deck_cards -->
| カラム名 | 型 | Nullable | 説明 |
|---|---|---|---|
| `player_id` | UUID | No | ルート親参照 |
| `deck_id` | BIGINT | No | 親テーブル参照 |
| `card_id` | VARCHAR(10) | No | カード識別子 |
| `art_no` | BIGINT | No | アート番号 |
| `count` | INT | No | 枚数 |
<!-- END GENERATED: deck_cards -->

### processed_events

Pub/Sub subscriber の冪等性を保証するテーブル。

- **PK:** `event_id` (UUID)

<!-- BEGIN GENERATED: processed_events -->
| カラム名 | 型 | Nullable | 説明 |
|---|---|---|---|
| `event_id` | UUID | No | Pub/Sub EventID (publisher 生成の UUIDv4) |
| `event_type` | TEXT | No | イベント種別 (card_pack_purchased / player_onboarded) |
| `processed_at` | TIMESTAMPTZ | No | 処理日時 |
<!-- END GENERATED: processed_events -->

**設計判断:**
- card は `card-pack-purchased` / `player-onboarded` イベントを subscribe し、shop 購入時・オンボーディング完了時に `card_pack` マスター定義に従ってカードを `player_cards` に付与する

---

## テーブル間リレーション

```
card_definitions (PK: card_id)
  （player_cards.card_id が参照するが FK は張らない — カード削除時に所持データが壊れないよう is_active で制御）

products (PK: product_id)
  └── 1:N ── initiatives (FK: product_id → products, CASCADE)
  （decks.product_id / routine_id / special_id が参照するが FK は張らない — アプリ層整合性）

[account.players] ─ ─ ─ (cross-schema, app-level)
  │
  ├── 1:N ── player_cards (PK: player_id, card_id, art_no)
  └── 1:N ── decks        (PK: player_id, deck_id)
                │
                └── 1:N ── deck_cards (FK: player_id, deck_id → decks, CASCADE)

processed_events (独立、FK なし)
```

---

## インデックス戦略

| インデックス | 対象 | 用途 |
|---|---|---|
| `idx_cards_faction` | `card_definitions(faction, card_type)` | 陣営別カード一覧取得。フィルタリング高速化 |
| `idx_cards_type` | `card_definitions(card_type)` | タイプ別検索 |
| `idx_products_faction` | `products(faction)` | 陣営別プロダクト一覧取得 |
| `idx_initiatives_product` | `initiatives(product_id)` | プロダクトの施策一括取得 |
| `idx_decks_player` | `decks(player_id, updated_at DESC)` | プレイヤーのデッキ一覧を更新日降順で取得 |
