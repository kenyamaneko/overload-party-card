# Card Service API Reference

> 型テーブルは `data/models.yaml` から自動生成。エンドポイント説明は手書き。
> `python3 scripts/generate_api_docs.py` でマーカー間を更新。

## Internal REST

- **Base path:** `/internal/v1`
- **認証:** internal

### `GET /internal/v1/cards`

全カード定義を返す（battle / gateway 起動時キャッシュロード用）。

**レスポンス**: `[]CardDefinition`

<!-- BEGIN GENERATED: CardDefinition -->
| フィールド | 型 | JSON | 説明 |
|---|---|---|---|
| `CardID` | `string` | `card_id` |  |
| `CardName` | `string` | `card_name` |  |
| `ResourceLabel` | `string` | `resource_label` |  |
| `Faction` | `string` | `faction` |  |
| `CardType` | `string` | `card_type` |  |
| `DeployTurns` | `int64` | `deploy_turns` |  |
| `Resizable` | `bool` | `resizable` |  |
| `Elastic` | `bool` | `elastic` |  |
| `ElasticIncrement` | `int64` | `elastic_increment` |  |
| `FreeTier` | `int64` | `free_tier` |  |
| `CostPerRequest` | `int64` | `cost_per_request` |  |
| `Stats` | `json.RawMessage` | `stats` |  |
| `EffectText` | `*string` | `effect_text` |  |
| `Effects` | `json.RawMessage` | `effects` |  |
| `PassiveEffects` | `[]PassiveEffect` | `passive_effects` |  |
| `PlatformEffects` | `[]PlatformEffect` | `platform_effects` |  |
| `AttachmentEffects` | `[]AttachmentEffect` | `attachment_effects` |  |
| `Restriction` | `string` | `restriction` |  |
| `IsActive` | `bool` | `is_active` |  |
| `CreatedAt` | `time.Time` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` |  |
<!-- END GENERATED: CardDefinition -->

#### エラー

| ステータス | 説明 |
|---|---|
| 500 | DB 接続エラー |

---

### `GET /internal/v1/players/{playerId}/cards`

プレイヤーの所持カード一覧を返す（PlayerCardWithDef 形式）。

**レスポンス**: `[]PlayerCardWithDef`

<!-- BEGIN GENERATED: PlayerCardWithDef -->
| フィールド | 型 | JSON | 説明 |
|---|---|---|---|
| `CardID` | `string` | `card_id` | カードID |
| `ArtNo` | `int64` | `art_no` | アート番号 |
| `Count` | `int` | `count` | 所持枚数 |
| `CardName` | `string` | `card_name` | カード名 |
| `ResourceLabel` | `string` | `resource_label` | リソースラベル（AWS/Azure/GCP/Oracle のサービス名） |
| `Faction` | `string` | `faction` | ファクション（`SHE` / `Tenki` / `Sugar` / `Tuners` / `Neutral`） |
| `CardType` | `string` | `card_type` | カード種別 |
| `DeployTurns` | `int64` | `deploy_turns` | デプロイターン数（0=即時） |
| `Resizable` | `bool` | `resizable` | 手動スケール可能か |
| `Elastic` | `bool` | `elastic` | 自動スケール対応か |
| `Stats` | `json.RawMessage` | `stats` | スタッツ（ComputeStats または DataStats） |
| `EffectText` | `*string` | `effect_text` | エフェクト説明テキスト |
| `Restriction` | `string` | `restriction` | 制限（`unlimited` / `semi_limited` / `limited` / `forbidden`） |
<!-- END GENERATED: PlayerCardWithDef -->

#### エラー

| ステータス | 説明 |
|---|---|
| 400 | playerId が空 |
| 500 | DB 接続エラー |

---

### `GET /internal/v1/players/{playerId}/cards/with-ownership`

全カード定義にプレイヤーの所持状態を付与して返す。

**レスポンス**: `[]CardWithOwnership`

<!-- BEGIN GENERATED: CardWithOwnership -->
| フィールド | 型 | JSON | 説明 |
|---|---|---|---|
| `CardID` | `string` | `card_id` | カードID |
| `CardName` | `string` | `card_name` | カード名 |
| `ResourceLabel` | `string` | `resource_label` | リソースラベル |
| `Faction` | `string` | `faction` | ファクション |
| `CardType` | `string` | `card_type` | カード種別 |
| `DeployTurns` | `int64` | `deploy_turns` | デプロイターン数 |
| `Resizable` | `bool` | `resizable` | 手動スケール可能か |
| `Elastic` | `bool` | `elastic` | 自動スケール対応か |
| `ElasticIncrement` | `int64` | `elastic_increment` | 自動スケール増分 |
| `FreeTier` | `int64` | `free_tier` | 無料枠 |
| `CostPerRequest` | `int64` | `cost_per_request` | リクエスト単価 |
| `Stats` | `json.RawMessage` | `stats` | スタッツ JSON |
| `EffectText` | `*string` | `effect_text` | エフェクト説明テキスト |
| `Effects` | `json.RawMessage` | `effects` | エフェクト定義 JSON |
| `PassiveEffects` | `[]PassiveEffect` | `passive_effects` | パッシブエフェクト |
| `PlatformEffects` | `[]PlatformEffect` | `platform_effects` | プラットフォームエフェクト |
| `AttachmentEffects` | `[]AttachmentEffect` | `attachment_effects` | アタッチメントエフェクト |
| `Restriction` | `string` | `restriction` | 制限区分 |
| `IsActive` | `bool` | `is_active` | 有効フラグ |
| `CreatedAt` | `time.Time` | `created_at` | 作成日時 |
| `UpdatedAt` | `time.Time` | `updated_at` | 更新日時 |
| `IsOwned` | `bool` | `is_owned` | プレイヤーが所持しているか |
<!-- END GENERATED: CardWithOwnership -->

#### エラー

| ステータス | 説明 |
|---|---|
| 400 | playerId が空 |
| 500 | DB 接続エラー |

---

### `GET /internal/v1/players/{playerId}/decks`

プレイヤーのデッキ一覧を返す。

**レスポンス**: `[]Deck`

<!-- BEGIN GENERATED: Deck -->
| フィールド | 型 | JSON | 説明 |
|---|---|---|---|
| `PlayerID` | `string` | `player_id` |  |
| `DeckID` | `int64` | `deck_id` | デッキID（自動採番） |
| `DeckName` | `string` | `deck_name` | デッキ名 |
| `IsValid` | `bool` | `is_valid` | バトル使用可能か（都度算出: 30枚 + 全カード所持 + 制限枚数以内） |
| `PlaymatNo` | `*int64` | `playmat_no` | プレイマット番号（null: デフォルト） |
| `SleeveNo` | `*int64` | `sleeve_no` | スリーブ番号（null: デフォルト） |
| `CreatedAt` | `time.Time` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` |  |
| `DeckCards` | `[]DeckCard` | `deck_cards` | デッキのカード構成（`card_id`, `art_no`, `count`） |
<!-- END GENERATED: Deck -->

#### エラー

| ステータス | 説明 |
|---|---|
| 400 | playerId が空 |
| 404 | プレイヤーが存在しない |
| 500 | DB 接続エラー |

---

### `GET /internal/v1/players/{playerId}/decks/{deckId}`

指定デッキの詳細（カード構成を含む）を返す。

**レスポンス**: `DeckDetailResponse`

<!-- BEGIN GENERATED: DeckDetailResponse -->
| フィールド | 型 | JSON | 説明 |
|---|---|---|---|
| `Deck` | `Deck` | `deck` | デッキ本体 |
| `Cards` | `[]DeckCard` | `cards` | デッキ内のカード一覧 |
<!-- END GENERATED: DeckDetailResponse -->

#### エラー

| ステータス | 説明 |
|---|---|
| 400 | deckId が不正 |
| 404 | デッキが存在しない |
| 500 | DB 接続エラー |

---

### `POST /internal/v1/players/{playerId}/decks`

新しいデッキを作成する。

> 成功時 201 Created を返す

**リクエスト**: `DeckCreateRequest`

<!-- BEGIN GENERATED: DeckCreateRequest -->
| フィールド | 型 | JSON | 説明 |
|---|---|---|---|
| `DeckName` | `string` | `deck_name` | デッキ名 |
| `Cards` | `[]DeckCardEntry` | `cards` | デッキのカード構成 |
| `PlaymatNo` | `*int64` | `playmat_no` | プレイマット番号 |
| `SleeveNo` | `*int64` | `sleeve_no` | スリーブ番号 |
<!-- END GENERATED: DeckCreateRequest -->

**レスポンス**: `Deck`

#### エラー

| ステータス | 説明 |
|---|---|
| 400 | リクエストボディ不正 / デッキバリデーション失敗 |
| 403 | 未所持カードを含む |
| 500 | DB 接続エラー |

---

### `PUT /internal/v1/players/{playerId}/decks/{deckId}`

既存デッキを更新する。

**リクエスト**: `DeckUpdateRequest`

<!-- BEGIN GENERATED: DeckUpdateRequest -->
| フィールド | 型 | JSON | 説明 |
|---|---|---|---|
| `DeckName` | `string` | `deck_name` | デッキ名 |
| `Cards` | `[]DeckCardEntry` | `cards` | デッキのカード構成 |
| `PlaymatNo` | `*int64` | `playmat_no` | プレイマット番号 |
| `SleeveNo` | `*int64` | `sleeve_no` | スリーブ番号 |
<!-- END GENERATED: DeckUpdateRequest -->

**レスポンス**: `Deck`

#### エラー

| ステータス | 説明 |
|---|---|
| 400 | deckId 不正 / リクエストボディ不正 / デッキバリデーション失敗 / 制限枚数超過 |
| 403 | 未所持カードを含む |
| 404 | デッキが存在しない |
| 500 | DB 接続エラー |

---

### `DELETE /internal/v1/players/{playerId}/decks/{deckId}`

デッキを削除する。

> 成功時 204 No Content を返す（レスポンスボディなし）

#### エラー

| ステータス | 説明 |
|---|---|
| 400 | deckId が不正 |
| 404 | デッキが存在しない |
| 500 | DB 接続エラー |

---

### `POST /internal/v1/players/{playerId}/decks/{deckId}/validate-for-battle`

デッキがバトル使用可能か検証する（カード所持・制限・枚数チェック）。

> 成功時 200 OK（ボディなし）。バリデーション失敗は 400 で返す

#### エラー

| ステータス | 説明 |
|---|---|
| 400 | deckId 不正 / デッキバリデーション失敗 / 制限枚数超過 |
| 403 | 未所持カードを含む |
| 404 | デッキが存在しない |
| 500 | DB 接続エラー |

---

### `POST /internal/v1/players/{playerId}/grant-initial-pack`

初期ファクション選択時にファクションカード + Neutral カードを各 3 枚付与する。

> 冪等性は呼び出し元（gateway）が player_factions で保証する

**リクエスト**: `GrantPackRequest`

<!-- BEGIN GENERATED: GrantPackRequest -->
| フィールド | 型 | JSON | 説明 |
|---|---|---|---|
| `Faction` | `string` | `faction` | 付与するファクション（SHE / Tenki / Sugar / Tuners） |
<!-- END GENERATED: GrantPackRequest -->

**レスポンス**: `GrantPackResponse`

<!-- BEGIN GENERATED: GrantPackResponse -->
| フィールド | 型 | JSON | 説明 |
|---|---|---|---|
| `CardsGranted` | `int` | `cardsGranted` | 付与されたカード枚数（カード種別数 x 3） |
<!-- END GENERATED: GrantPackResponse -->

#### エラー

| ステータス | 説明 |
|---|---|
| 400 | playerId が空 / リクエストボディ不正 / 無効なファクション |
| 500 | DB 接続エラー |

---

### `POST /internal/v1/players/{playerId}/grant-faction-pack`

ショップ購入時に指定ファクションのカードを各 3 枚付与する（Neutral なし）。

**リクエスト**: `GrantPackRequest`

**レスポンス**: `GrantPackResponse`

#### エラー

| ステータス | 説明 |
|---|---|
| 400 | playerId が空 / リクエストボディ不正 / 無効なファクション |
| 500 | DB 接続エラー |
