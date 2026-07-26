# Card 機能仕様書

このドキュメントは card サービスがビジネス要件として満たすべき振る舞いを定義する。実装方法ではなく **何を保証するか** を記述する。テストはこの仕様に従っていることを確認する観点で書く。

関連ドキュメント:
- 内部動作・配線・運用設定: [ARCHITECTURE.md](ARCHITECTURE.md)
- HTTP エンドポイント契約: [data/openapi.yaml](../data/openapi.yaml) (SSoT)
- DB スキーマ: [DATA_DESIGN.md](DATA_DESIGN.md)
- カードデータ仕様: [CARDS.md](CARDS.md)

---

## サービス責務

card は以下の機能ドメインを所有する。

| 機能 | 主要な責務 |
|---|---|
| カードマスター配信 | 全カード定義を battle / gateway 向けに返す（SSoT） |
| プロダクトマスター配信 | プロダクト定義（陣営に N:1、施策の効果 DSL 込み）を battle / client 向けに返す（SSoT） |
| 所持カード照会 | プレイヤーの所持カード一覧・カード定義付き形式での返却 |
| デッキ CRUD | プレイヤー単位のデッキ作成・更新・削除・一覧（陣営を宣言して構築） |
| デッキバリデーション | 枚数・所持・制限・陣営整合ルールの検証（都度算出） |
| バトル前検証 | `validate-for-battle` でバトル開始前のゲートキーパー |
| カードパック配布 | オンボーディング完了時・ショップ購入時のカード付与 |
| Pub/Sub 消費 | `card-pack-purchased` / `player-onboarded` を購読しパック配布をイベント駆動で実行 |

card は **card スキーマの DB 行とカード定義 YAML (SSoT) を唯一の真実とし**、他サービスへカードマスターを複製しない。他サービスは `GET /internal/v1/cards` で起動時に読み取る read model を各自で持つ。

---

## カードマスター (`card_definitions`)

### SSoT

- `data/cards/*.yaml` がカードデータの SSoT
- `scripts/generate_cards.py` が YAML から `db/seed/cards_seed.sql` と `data/cards_gen.json` を生成する
- `card_definitions` テーブルは ops の `db-migrate` (psqldef) 経由でスキーマが適用され、seed が投入される
- 他サービスはカードマスターを複製しない（契約）

カードデータの意味定義は [CARDS.md](CARDS.md) を参照。

### 読み取り契約

| エンドポイント | 呼び出し元 | タイミング |
|---|---|---|
| `GET /internal/v1/cards` | battle | 起動時に 1 回、インメモリにロード |
| `GET /internal/v1/cards` | gateway | 起動時に 1 回、forward 用のキャッシュに入れる |

card 自身は `cache.CardCache` に起動時にロードし、以降はメモリから参照する。ロード失敗で card は起動しない（fail-fast）。

### プロダクトマスター

- `data/products.yaml`（プロダクト）と `data/initiatives.yaml`（施策）がプロダクトデータの SSoT
- `scripts/generate_products.py` が検証（選択可能な全陣営にプロダクト 1 つ以上、各プロダクトにルーチン / スペシャル各 1 つ以上）と生成（`data/cache/*_gen.json` / `db/seed/products_seed.sql` / `db/seed/initiatives_seed.sql` / [PRODUCTS.md](PRODUCTS.md)）を行う
- `card.products` / `card.initiatives` テーブルに seed が投入され、起動時に `cache.ProductCache` / `cache.InitiativeCache` へ全件ロードして配信する。ロード失敗で card は起動しない（fail-fast）
- `GET /internal/v1/initiatives`（battle 起動時ロード用、認証なし）と `GET /api/v1/cards/products`（client 用、InternalAuth）で配信する

プロダクトデータの意味定義は [PRODUCTS.md](PRODUCTS.md) を参照。

---

## 所持カード (`player_cards`)

所持カードは `(player_id, card_id, art_no)` を単位に枚数を管理する。`art_no` はカードの見た目バリエーション識別子で、同一カードでもアートが違えば別行。

### 取得 API

1. `GET /api/v1/cards/cards`：所持分のみ、カード定義付き形式 (`PlayerCardWithDef`)
2. `GET /api/v1/cards/cards/with-ownership`：全カード定義に所持フラグを付与

どちらも CardCache を経由して定義情報を埋める。CardCache に存在しないカードを所持しているケースは内部エラー（データ整合性異常）として扱う。

---

## デッキ (`decks` / `deck_cards`)

### デッキ制約

- デッキは陣営（`decks.faction`）を 1 つ宣言する。宣言できるのは選択可能な陣営（`SelectableFactions`）のみ
- 宣言陣営はプレイヤーが所持している陣営であること（account へ照会して検証）
- デッキは宣言陣営に属するプロダクト（`decks.product_id`）を 1 つ選ぶ。陣営はプロダクトを複数持てる（陣営:プロダクト = 1:N）
- 選んだプロダクトの施策から、ルーチン（`decks.routine_id`）とスペシャル（`decks.special_id`）をそれぞれ 1 つずつセットする。プロダクトは各区分の施策を複数持てる（数は区分ごとに異なってよい）。プロダクトを跨いだ施策の組み合わせは不可
- 構成カードは宣言陣営と Neutral のカードのみ（混成不可）
- デッキ枚数: ちょうど `constants.DeckSize`（= 30）枚
- 構成カード全てをプレイヤーが所持していること（`player_cards.count >= deck_cards.count`）
- 各カードが `card_definitions.restriction` の上限枚数以内であること

| restriction | 同一 `card_id` のコピー上限 |
|---|---|
| `unlimited` | 3 |
| `semi_limited` | 2 |
| `limited` | 1 |
| `forbidden` | 0（デッキに入れられない） |

### `is_valid` の算出

**`is_valid` は DB に保存しない。** レスポンス生成時にサービス層が都度算出する（`computeIsValid`）。

理由: 所持カードの変動や restriction 改定を即座に反映するため。保存すると配布直後や制限改定直後にスタッルレスポンスが発生する。

### デッキ作成・更新のバリデーション順序

`CreateDeck` / `UpdateDeck` は以下の順で検証し、失敗時点で即 return する。

1. 所持カード取得
2. `validateDeckCards`:
   1. 宣言陣営が `SelectableFactions` に含まれること（`ErrInvalidDeck`）
   2. 各エントリの `count > 0` チェック（`ErrInvalidDeck`）
   3. 総枚数 ≤ `DeckSize` チェック（`ErrInvalidDeck`）
   4. 各 `(card_id, art_no)` について所持枚数 ≥ 要求枚数（`ErrUnowned`）
   5. 各カードの陣営が宣言陣営または Neutral であること（`ErrInvalidDeck`）
   6. 各 `card_id` の合計枚数 ≤ restriction 上限（`ErrRestrictionExceeded`）
3. `validateFactionOwnership`: 宣言陣営をプレイヤーが所持しているか account に照会（`ErrInvalidDeck`）
4. `validateInitiatives`: `product_id` が宣言陣営のプロダクトであり、`routine_id` / `special_id` がそのプロダクトの該当区分の施策であること（`ErrInvalidDeck`）
5. 書き込み（デッキ行 + deck_cards）
6. 返却時に DeckCards を populate

**作成時の `is_valid`**: 総枚数 == `DeckSize` かつ validateDeckCards を通った場合のみ `true`。30 枚未満のドラフト状態でも保存はできるが `is_valid=false` になる。

### `validate-for-battle` の契約

`POST /api/v1/cards/decks/{deckId}/validate-for-battle` はバトル開始前のゲートキーパー。`is_valid=true` と同等のチェックを行い、不可の場合は具体的な理由をエラーで返す。gateway は matchmaking enqueue / NPC 対戦作成の前にこれを呼ぶ。

チェック順序:

1. デッキヘッダ取得（宣言陣営の解決）
2. `deck_cards` 取得
3. 総枚数 == `DeckSize`（`ErrInvalidDeck`）
4. 所持カード取得
5. `validateDeckCards`（「デッキ作成・更新のバリデーション順序」と同じ）
6. `validateInitiatives`（セットした施策が宣言陣営のものか）

### デッキ削除

`DELETE /api/v1/cards/decks/{deckId}` は `decks` 行を削除する。`deck_cards` は FK の `ON DELETE CASCADE` で同時に削除される。

---

## カードパック配布

### 配布種別

配布は内部 API `usecase.GrantInteractor.GrantPack(playerID, packID)` のみに集約され、`pack_id` をキーとして `card.card_pack` マスターから配布対象 (`card_id` × `copies`) を引く。subscriber 側で業務文脈に応じた `pack_id` を組み立てる:

| 文脈 | 呼び出し元 | 渡す `pack_id` |
|---|---|---|
| onboarding 初期配布 | `player-onboarded-card-sub` | `basic` と `faction_set_<initial_faction>` の 2 回 |
| shop 購入 | `card-pack-purchased-card-sub` | event payload の `card_pack_id` をそのまま |

REST 経由の同期配布エンドポイントは ADR-032 で完全廃止されている。

### 書き込みセマンティクス

- `player_cards` に UPSERT（加算）。既に所持済みでも pack 定義の `copies` ぶん加算する
- 単一トランザクションで完結
- 付与されたコピー総数を返す

### ファクション値の扱い

- 配布経路では faction 値そのものを検証しない。`player-onboarded-card-sub` が `faction_set_<initial_faction>`（小文字化）の `pack_id` を組み立て、対応する pack が `card_pack` マスターに存在しなければ配布が失敗して `Nack` される

### 冪等性の方針

**本 API 自体は非冪等**。「呼ばれたら必ず pack 定義の `copies` ぶん加算する」セマンティクス。

冪等性は呼び出し側の Pub/Sub subscriber が「冪等性の契約」の節の event_id dedup で担保する。

---

## Pub/Sub subscriber (`card-pack-purchased` / `player-onboarded`)

shop 購入・オンボーディング完了をイベント駆動でパック配布に変換する。`card.processed_events` を使って at-least-once 配信を実質的に一度だけ適用する。

配信は Cloud Pub/Sub の push 経由で `/internal/v1/pubsub/<イベント名>` (`internal/handler/rest.PubSubPushHandler`) が受け、デコードした payload を以下の subscriber へ渡す。以下の `Ack` / `Nack` は subscriber の戻り値 (nil / error) を指し、push handler がそれぞれ HTTP 2xx / 5xx に変換して Cloud Pub/Sub へ応答する。

### 処理フロー (共通)

1. JSON デシリアライズ → 失敗は `Nack`
2. `event_type` チェック。異なれば `Nack`（publisher バグなので DLQ 行き）
3. `processed_events` に `event_id` を INSERT
   - 既存行があれば `inserted = false` で `Ack`（重複適用しない）
   - INSERT 自体が失敗したら `Nack`
4. subscriber 固有の配布を実行:
   - `card-pack-purchased-card-sub` → `GrantPack(playerID, ev.CardPackID)`
   - `player-onboarded-card-sub` → `GrantPack(playerID, "basic")` + `GrantPack(playerID, "faction_set_<faction>")` を順次
5. 配布成功 → `Ack`、配布失敗 → `Nack`（Pub/Sub がリトライ）

各 subscriber は自身の topic に紐づく副作用だけを実行する。

### 冪等性の契約

- **キー**: `event_id`（publisher が UUIDv4 で生成）
- **保証**: 同一 event_id で複数回配信されても `player_cards` に重複加算しない
- **fast-path 位置付け**: 「カードパック配布」の節の GrantInteractor は独自 UPSERT を持ち `processed_events` と同一 tx に含められないため、processed_events dedup は「再試行抑止用の前段ガード」として機能する。稀に同一イベントで 2 回加算される可能性は完全には排除していないが、publisher 側のリトライ窓が短いため実用上問題にならない
- **decode 失敗**: `Nack` して Pub/Sub DLQ に送る（自動廃棄しない。運用側で調査）

### 未知 event_type の扱い

`Nack` を返す。リトライしても結果は変わらないが、subscriber 側で握りつぶさず DLQ (`card-pack-purchased-dlq` / `player-onboarded-dlq`) に回収してオペレーション側で検出するポリシー。publisher の契約違反やイベントスキーマ拡張時の取りこぼしを早期に気付けるようにする意図。

---

## エラーセマンティクス

サービス層は HTTP ステータスを知らない。エラーはセンチネル (`internal/port/errors.go`) として返し、handler (`internal/handler/rest/errors.go`) が `errors.Is` で transport 層のステータスに変換する。

### 分類

| センチネル | HTTP | 意味 |
|---|---|---|
| `ErrNotFound` | 404 | デッキ・プレイヤー不存在 |
| `ErrUnowned` | 403 | 未所持カードをデッキに含めた |
| `ErrInvalidDeck` | 400 | 枚数不正・未知 card_id・count ≤ 0 |
| `ErrRestrictionExceeded` | 400 | restriction 上限超過 |
| `ErrInvalidArgument` | 400 | playerId / deckId / faction 等のパラメータ不正 |
| *(上記以外)* | 500 | DB 接続失敗等の内部エラー |

### `ErrUnowned` を 403 にしている理由

デッキバリデーションの観点では「所持していないカードを入れようとした」は 400 でも良いが、所有権の問題として 403（Forbidden）で区別している。クライアントはこれをもって「shop 導線への誘導」などを出し分けられる。

---

## イベント消費

| トピック | 購読名 | 発行元 | 契機 |
|---|---|---|---|
| `card-pack-purchased` | `card-pack-purchased-card-sub` | shop | プレイヤーが shop で card_pack を含む商品 (faction_set / 限定 pack 等) を購入した時 |
| `player-onboarded` | `player-onboarded-card-sub` | scenario | プレイヤーがオンボーディングを完了した時 |

card は **イベントを発行しない**。subscriber は source 分岐なしで自 topic 固有の副作用だけを処理する。
