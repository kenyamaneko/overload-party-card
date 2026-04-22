# Card 機能仕様書

このドキュメントは card サービスがビジネス要件として満たすべき振る舞いを定義する。実装方法ではなく **何を保証するか** を記述する。テストはこの仕様に従っていることを確認する観点で書く。

関連ドキュメント:
- 内部動作・配線・運用設定: [ARCHITECTURE.md](ARCHITECTURE.md)
- HTTP エンドポイント契約: [API_REFERENCE.md](API_REFERENCE.md)
- DB スキーマ: [DATA_DESIGN.md](DATA_DESIGN.md)
- カードデータ仕様: [CARDS.md](CARDS.md)

---

## 1. サービス責務

card は以下の機能ドメインを所有する。

| 機能 | 主要な責務 |
|---|---|
| カードマスター配信 | 全カード定義を battle / gateway 向けに返す（SSoT） |
| 所持カード照会 | プレイヤーの所持カード一覧・カード定義付き形式での返却 |
| デッキ CRUD | プレイヤー単位のデッキ作成・更新・削除・一覧 |
| デッキバリデーション | 枚数・所持・制限ルールの検証（都度算出） |
| バトル前検証 | `validate-for-battle` でバトル開始前のゲートキーパー |
| カードパック配布 | 初回ファクション選択時・ショップ購入時のカード付与 |
| Pub/Sub 消費 | `faction-purchased` / `player-onboarded` を購読しパック配布をイベント駆動で実行 (ADR-022) |

card は **card スキーマの DB 行とカード定義 YAML (SSoT) を唯一の真実とし**、他サービスへカードマスターを複製しない。他サービスは `GET /internal/v1/cards` で起動時に読み取る read model を各自で持つ。

---

## 2. カードマスター (`card_definitions`)

### 2.1 SSoT

- `data/cards/*.yaml` がカードデータの SSoT
- `scripts/generate_cards.py` が YAML から `db/seed/cards_seed.sql` と `data/cards_gen.json` を生成する
- `card_definitions` テーブルは ops の `db-migrate` (psqldef) 経由でスキーマが適用され、seed が投入される
- 他サービスはカードマスターを複製しない（契約）

カードデータの意味定義は [CARDS.md](CARDS.md) を参照。

### 2.2 読み取り契約

| エンドポイント | 呼び出し元 | タイミング |
|---|---|---|
| `GET /internal/v1/cards` | battle | 起動時に 1 回、インメモリにロード |
| `GET /internal/v1/cards` | gateway | 起動時に 1 回、forward 用のキャッシュに入れる |

card 自身は `cache.CardCache` に起動時にロードし、以降はメモリから参照する。ロード失敗で card は起動しない（fail-fast）。

---

## 3. 所持カード (`player_cards`)

所持カードは `(player_id, card_id, art_no)` を単位に枚数を管理する。`art_no` はカードの見た目バリエーション識別子で、同一カードでもアートが違えば別行。

### 3.1 取得 API

1. `GET /internal/v1/players/{playerId}/cards` — 所持分のみ、カード定義付き形式 (`PlayerCardWithDef`)
2. `GET /internal/v1/players/{playerId}/cards/with-ownership` — 全カード定義に所持フラグを付与

どちらも CardCache を経由して定義情報を埋める。CardCache に存在しないカードを所持しているケースは内部エラー（データ整合性異常）として扱う。

---

## 4. デッキ (`decks` / `deck_cards`)

### 4.1 デッキ制約

- デッキ枚数: ちょうど `constants.DeckSize`（= 30）枚
- 構成カード全てをプレイヤーが所持していること（`player_cards.count >= deck_cards.count`）
- 各カードが `card_definitions.restriction` の上限枚数以内であること

| restriction | 同一 `card_id` のコピー上限 |
|---|---|
| `unlimited` | 3 |
| `semi_limited` | 2 |
| `limited` | 1 |
| `forbidden` | 0（デッキに入れられない） |

### 4.2 `is_valid` の算出

**`is_valid` は DB に保存しない。** レスポンス生成時にサービス層が都度算出する（`computeIsValid`）。

理由: 所持カードの変動や restriction 改定を即座に反映するため。保存すると配布直後や制限改定直後にスタッルレスポンスが発生する。

### 4.3 デッキ作成・更新のバリデーション順序

`CreateDeck` / `UpdateDeck` は以下の順で検証し、失敗時点で即 return する。

1. 所持カード取得
2. `validateDeckCards`:
   1. 各エントリの `count > 0` チェック（`ErrInvalidDeck`）
   2. 総枚数 ≤ `DeckSize` チェック（`ErrInvalidDeck`）
   3. 各 `(card_id, art_no)` について所持枚数 ≥ 要求枚数（`ErrUnowned`）
   4. 各 `card_id` の合計枚数 ≤ restriction 上限（`ErrRestrictionExceeded`）
3. 書き込み（デッキ行 + deck_cards）
4. 返却時に DeckCards を populate

**作成時の `is_valid`**: 総枚数 == `DeckSize` かつ validateDeckCards を通った場合のみ `true`。30 枚未満のドラフト状態でも保存はできるが `is_valid=false` になる。

### 4.4 `validate-for-battle` の契約

`POST /internal/v1/players/{playerId}/decks/{deckId}/validate-for-battle` はバトル開始前のゲートキーパー。`is_valid=true` と同等のチェックを行い、不可の場合は具体的な理由をエラーで返す。gateway は matchmaking enqueue / NPC 対戦作成の前にこれを呼ぶ。

チェック順序:

1. `deck_cards` 取得
2. 総枚数 == `DeckSize`（`ErrInvalidDeck`）
3. 所持カード取得
4. `validateDeckCards`（3.3 と同じ）

### 4.5 デッキ削除

`DELETE /internal/v1/players/{playerId}/decks/{deckId}` は `decks` 行を削除する。`deck_cards` は FK の `ON DELETE CASCADE` で同時に削除される。

---

## 5. カードパック配布

### 5.1 配布種別

| API | 配布対象 | 呼び出し契機 |
|---|---|---|
| `POST .../grant-initial-pack` | 選択 faction + Neutral の全カード種類 × 3 枚 | onboarding 完了 (`player-onboarded` subscriber 経由) |
| `POST .../grant-faction-pack` | 選択 faction の全カード種類 × 3 枚（Neutral なし） | ショップでの faction_set 購入 (`faction-purchased` subscriber 経由) |

配布対象の faction は `card_definitions.faction` カラムで動的にフィルタする（ハードコードしない）。

### 5.2 書き込みセマンティクス

- `player_cards` に UPSERT（加算）。既に所持済みでも 3 枚追加する
- 単一トランザクションで完結
- 付与されたコピー総数を返す

### 5.3 ファクション値バリデーション

- `faction` は `gamedesign.SelectableFactions`（SHE / Tenki / Sugar / Tuners）のいずれかであること
- Neutral は `grant-initial-pack` の配布対象には含まれるが、`faction` パラメータとしては受け付けない

### 5.4 冪等性の方針

**本 API 自体は非冪等**。「呼ばれたら必ず 3 枚ずつ加算する」セマンティクス。

冪等性は呼び出し側が担保する:

- REST 経由の場合: gateway のオーケストレーション層が重複呼び出しを抑止
- Pub/Sub subscriber 経由の場合: §6 の event_id dedup で担保

---

## 6. Pub/Sub subscriber (`faction-purchased` / `player-onboarded`)

ファクション取得・オンボーディング完了をイベント駆動でパック配布に変換する (ADR-022)。`card.processed_events` を使って at-least-once 配信を実質的に一度だけ適用する。

### 6.1 処理フロー (共通)

1. JSON デシリアライズ → 失敗は `Nack`
2. `event_type` チェック。異なれば `Nack`（publisher バグなので DLQ 行き）
3. `processed_events` に `event_id` を INSERT
   - 既存行があれば `inserted = false` で `Ack`（重複適用しない）
   - INSERT 自体が失敗したら `Nack`
4. subscriber 固有の配布を実行:
   - `faction-purchased-card-sub` → `GrantFactionPack`（faction のみ）
   - `player-onboarded-card-sub` → `GrantInitialPack`（faction + Neutral）
5. 配布成功 → `Ack`、配布失敗 → `Nack`（Pub/Sub がリトライ）

ADR-022 で業務事実単位に topic を分解したため、旧 `source` フィールド分岐は撤廃された。各 subscriber は自身の topic に紐づく副作用だけを実行する。

### 6.2 冪等性の契約

- **キー**: `event_id`（publisher が UUIDv4 で生成）
- **保証**: 同一 event_id で複数回配信されても `player_cards` に重複加算しない
- **fast-path 位置付け**: §5 の GrantService は独自 UPSERT を持ち `processed_events` と同一 tx に含められないため、processed_events dedup は「再試行抑止用の前段ガード」として機能する。稀に同一イベントで 2 回加算される可能性は完全には排除していないが、publisher 側のリトライ窓が短いため実用上問題にならない
- **decode 失敗**: `Nack` して Pub/Sub DLQ に送る（自動廃棄しない。運用側で調査）

### 6.3 未知 event_type の扱い

`Nack` を返す。リトライしても結果は変わらないが、subscriber 側で握りつぶさず DLQ (`faction-purchased-dlq` / `player-onboarded-dlq`) に回収してオペレーション側で検出するポリシー。publisher の契約違反やイベントスキーマ拡張時の取りこぼしを早期に気付けるようにする意図。

---

## 7. エラーセマンティクス

サービス層は HTTP ステータスを知らない。エラーはセンチネル (`internal/port/errors.go`) として返し、handler (`internal/handler/rest/errors.go`) が `errors.Is` で transport 層のステータスに変換する。

### 7.1 分類

| センチネル | HTTP | 意味 |
|---|---|---|
| `ErrNotFound` | 404 | デッキ・プレイヤー不存在 |
| `ErrUnowned` | 403 | 未所持カードをデッキに含めた |
| `ErrInvalidDeck` | 400 | 枚数不正・未知 card_id・count ≤ 0 |
| `ErrRestrictionExceeded` | 400 | restriction 上限超過 |
| `ErrInvalidArgument` | 400 | playerId / deckId / faction 等のパラメータ不正 |
| *(上記以外)* | 500 | DB 接続失敗等の内部エラー |

### 7.2 `ErrUnowned` を 403 にしている理由

デッキバリデーションの観点では「所持していないカードを入れようとした」は 400 でも良いが、所有権の問題として 403（Forbidden）で区別している。クライアントはこれをもって「shop 導線への誘導」などを出し分けられる。

---

## 8. イベント消費

| トピック | 購読名 | 発行元 | 契機 |
|---|---|---|---|
| `faction-purchased` | `faction-purchased-card-sub` | shop | プレイヤーが shop で faction を購入した時 |
| `player-onboarded` | `player-onboarded-card-sub` | scenario | プレイヤーがオンボーディングを完了した時 |

card は **イベントを発行しない**。ADR-022 で業務事実単位に topic を分離したため、subscriber は source 分岐なしで自 topic 固有の副作用だけを処理する。
