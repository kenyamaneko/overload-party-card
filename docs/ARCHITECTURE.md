# Card サービス設計

本ドキュメントは **コードを読んでも一見しては分からない設計意図** だけを残す。実装詳細（バリデーション順序・state 遷移の対応・エラー → HTTP ステータス変換・環境変数の一覧）は [FEATURE_SPEC.md](FEATURE_SPEC.md) と各ファイルの実装・コメントを一次情報とする。

サービス概要・起動手順は [../README.md](../README.md)、エンドポイント契約は [data/openapi.yaml](../data/openapi.yaml) (SSoT)、テーブル定義は [DATA_DESIGN.md](DATA_DESIGN.md) を参照。

## Card の責務境界 (SSoT 契約)

card は **カードマスターデータそのもの** の single source of truth である。他サービスはカード定義を DB テーブル/seed として持たず、card の `GET /internal/v1/cards` を起動時に呼び出してインメモリに保持する read model を各自で構築する。

エンドポイントは 2 系統に分かれる。`/internal/v1/cards` は battle / gateway が起動時に master データをロードする経路で player_id を要求しないため認証なし。`/api/v1/cards/...` は gateway 経由の player-scoped 操作 (デッキ管理・所持カード照会等) で `X-Internal-Auth` (HMAC JWT) を要求し、`sub` クレームから player_id を解決する (ADR-037)。

この契約により:

- カードマスターの複製が存在しないため、不整合が原理的に発生しない
- カードバランス調整は card サービスのみで完結する（他サービスの再デプロイ不要な改定範囲が広い）
- battle は試合中のあらゆる判定をローカルのキャッシュで行えるため、試合フローで card への同期リクエストが発生しない

battle / gateway がカードマスターを自 DB に持たないことは設計上の契約であり、パフォーマンス最適化ではない。

## カードデータ SSoT フロー

```
data/cards/*.yaml                 (game designer が直接編集する SSoT)
  │
  ▼
scripts/generate_cards.py   →  db/seed/cards_seed.sql
                              data/cards_gen.json
  │
  ▼
psqldef (ops/db-migrate)    →  card.card_definitions テーブル
  │
  ▼
cache.CardCache (起動時に全件ロード) → サービス内メモリ
```

- `data/cards/*.yaml` が人間が編集する SSoT。PR レビューの対象はここ
- `cards_seed.sql` / `cards_gen.json` は生成物。直接編集禁止
- schema migration は ops リポジトリ側の `db-migrate` が担当する（card からは走らない）

カードの意味定義（faction / card_type / subtype / stats JSON の構造等）は [CARDS.md](CARDS.md) を参照。

## インメモリ CardCache の fail-fast 判断

`cache.NewCardCache()` は起動時に `port.CardRepo` から全カード定義を読み込む。以後はリクエストごとの DB アクセスなしにカード定義を参照する。

- `Load(ctx, cardRepo)`: 起動時 1 回のみ呼び出し。**失敗は fatal**（サービス起動を止める）
- `Get(cardId)`: O(1) map lookup
- `All()`: 全件の slice を返す

**fail-fast を選んだ理由:**

1. CardCache に存在しないカードをプレイヤーが所持している状態は「データ整合性が壊れている」ことを意味し、中途半端に起動するとユーザー向けレスポンスに混入する
2. カードマスターが欠けた状態での `validate-for-battle` は false negative / false positive のどちらでも重大な体験事故になる（未所持カードでバトル開始できる／正当なデッキが弾かれる）
3. Readiness probe で失敗 → 新 Pod が ready にならず、旧 Pod でトラフィックを受け続けるほうが安全

CardCache は読み取り専用。マスター改定時は Pod 再起動で invalidation する（将来的に Pub/Sub invalidation を検討中）。ローリング再起動中に新旧バージョンが混在する窓があるが、カードマスター改定は破壊的変更ではない運用ルール（新カード追加・非アクティブ化のみ）で許容している。

## pack マスターはキャッシュしない

`card.card_pack` は配布リクエストごとに DB を引く（[ADR-032](https://github.com/kenyamaneko/overload-party-common/blob/main/docs/adr/032-card-pack-introduction-and-grant-unification.md)「配布 API を `GrantPack(pack_id)` に統一」で確定）。CardCache 同様の起動時 fail-fast ロード戦略は **採らない**。

理由:

1. **配布履歴の audit 性**: 各購入が「その時点の pack 定義に従って配布された」ことを再現可能にする必要があり、Pod 起動時スナップショットでは「いつどの定義で配ったか」が曖昧になる
2. **遡及配布の運用**: 後から pack 内容を変更したとき、既存購入者にも差分配布する運用要件を許容する。キャッシュだとローリング再起動中に「新 Pod は新定義 / 旧 Pod は旧定義」が混在し、遡及配布が予測不能になる
3. **配布タイミングの即時反映**: pack 定義変更が次の配布リクエストから直ちに効く（Pod 再起動不要）

CardCache は静的なカードマスターを参照する read-heavy ホットパス（デッキ構築・battle 判定）で使う前提で、性質が違う。配布リクエストは onboarding / shop 購入の頻度でホットパスではなく、`card_pack` は件数も小さい（数件オーダー）ため pack ヘッダ + 内包カード行の LEFT JOIN は ms オーダーで毎回 DB 参照のコストは無視できる。

## `is_valid` を DB に保存しない判断

デッキの `is_valid` は API レスポンス生成時に都度算出する。DB には持たない。

保存すると次の場合にスタッルレスポンスが発生する:

- プレイヤーが `restriction` を超過させるカードを所持し直した／失った
- game designer が `restriction` を `unlimited → limited` に引き下げた（バン）
- 新カード追加で過去デッキが依然合法であることを確認したい

算出はデッキ単位で O(|deck_cards| + |player_cards|) の map lookup。デッキ一覧取得でも実測でボトルネックにならないため、正確性を優先して常時再計算している。

## Pub/Sub subscriber の冪等性

card は `player-onboarded-card-sub` と `card-pack-purchased-card-sub` を push 配信で受信し、パック配布をイベント駆動で実行する (ADR-022 / ADR-031 / ADR-032 / ADR-057)。push の受け口は `internal/handler/rest` の `PubSubPushHandler` (`/internal/v1/pubsub/player-onboarded` / `/internal/v1/pubsub/card-pack-purchased`) で、デコードした payload を本節の subscriber (`internal/adapter/pubsub`) にそのまま渡す。到達制御はアプリ層で検証せず Cloud Run の呼び出し IAM に委ねる (ADR-057)。冪等性は `card.processed_events` テーブルに `event_id` を INSERT することで前段ガードとして機能させる。

ただし **完全な idempotency 保証ではない**:

- `GrantInteractor` の `player_cards` UPSERT は独自トランザクションを張る
- `processed_events` INSERT と UPSERT を同一 tx にまとめると、GrantInteractor が `port.PlayerCardRepo` を通じて内部で tx 管理するという責任分界が壊れる
- そこで processed_events は「**再試行の早期打ち切り用 fast-path**」と位置付け、grant 実行中のクラッシュ窓では稀に 2 重加算がありうる（publisher 側のリトライ間隔が短く、実用上の影響は限定的）

at-least-once を at-most-once 相当に近づけるための前段ガードであり、strict exactly-once を謳っていない点に注意。将来 GrantInteractor を processed_events と同一 tx に組み込む設計変更で完全 idempotent にできる（現状は実用上の優先度が低い）。

### subscriber の責務分離 (ADR-022 / ADR-031 / ADR-032)

ADR-022 で `FactionSelectedEvent` を業務事実単位に分解し、ADR-031 / ADR-032 で `faction-purchased` を `card-pack-purchased` (card 向け) + `faction-acquired` (account / gateway 向け) に再分解した結果、card は以下の subscriber を持つ:

| subscriber | 副作用 |
|---|---|
| `player-onboarded-card-sub` | onboarding 完了時に `basic` pack と `faction_set_<faction>` pack を順次配布 (内部で `GrantPack` を 2 回呼ぶ) |
| `card-pack-purchased-card-sub` | shop の card_pack 系商品 (faction_set / 限定パック等) 購入時に `card_pack_id` で指定された pack を配布 |

`player-onboarded` での 2 pack 配布は順次呼びのため、間でクラッシュすると「basic だけ配布された」「faction だけ配布された」「重複配布」のいずれかの状態が残りうる。これは下記「Pub/Sub subscriber の冪等性」で述べる at-most-once 相当の制約を 2 pack に拡張したもので、稼働前は許容する。

card は `faction-acquired` を購読**しない** (faction 所有権の SSoT は account の責務、ADR-031)。業務事実と topic が 1 対 1 に対応するため、subscriber は自身の topic に紐づく副作用だけを実行する。

旧 `faction-purchased-card-sub` (および `faction-selected-card-sub`) は廃止された。配布の業務文脈 (initial / 購入 / 限定) は subscriber が `pack_id` を組み立てる (or wire payload で受け取る) 形に集約され、配布の SSoT は `card.card_pack` マスターに移管された (ADR-032)。

### 握りつぶし禁止: 不明イベントも Nack

push 配信では Ack / Nack を SDK 呼び出しではなく HTTP 応答 (2xx / 5xx) で表す。subscriber の `Handle` が nil を返すと `PubSubPushHandler` が 2xx (Ack 相当) を、error を返すと 5xx (Nack 相当、Cloud Pub/Sub が再配信) を返す。

`event_type` が未知の場合、従来は Ack して捨てていたが、現在は **Nack して DLQ に回収する**方針に変えた。理由:

- CLAUDE.md「エラーは握りつぶさず根本解決する」との整合
- Ack で捨てると publisher 側のバグに気付けない
- DLQ は無限リトライを起こさないうえ、監視側でアラートを張れる

復旧不能な malformed payload も同じく Nack で DLQ に寄せる。at-least-once の契約は DLQ 配送後に自動で満たされる（DLQ から人間が invest → 必要なら再投入）。

## クロススキーマ参照の方針

`player_cards` / `decks` / `deck_cards` の `player_id` は `account.players` へのクロススキーマ参照だが **FK は張らない**。理由:

- account と card は別サービスで独立デプロイされる。外部 FK は デプロイ順序依存を生む
- 整合性は app 層で守る（プレイヤー作成前に card API が呼ばれた場合は 500 系ではなく INSERT 失敗扱い）
- `card_definitions.card_id` も `player_cards.card_id` から FK なしで参照される。カード論理削除（`is_active=false`）時に所持データが壊れないようにするため

`deck_cards` → `decks` は同一スキーマ内なので通常の FK + `ON DELETE CASCADE` を張っている（デッキ削除で構成カードを自動削除）。

## カード配布 API の非冪等契約

card の配布 API は `usecase.GrantInteractor.GrantPack(playerID, packID)` のみで、Pub/Sub subscriber 経由でだけ呼ばれる。pack に登録された `(card_id, copies)` ペアを **呼ばれた回数ぶん加算する**。自前で冪等性を持たない。

冪等性は `card.processed_events` の event_id 前段ガードで at-most-once 相当に近づける（上記「Pub/Sub subscriber の冪等性」を参照）。`processed_events` INSERT と `GrantPack` 内の `player_cards` UPSERT を同一 tx に乗せられない既存制約は、本ADR でも踏襲する（ADR-032）。

「配布 API 自体を冪等にする」方向に倒すと、「本当に 2 回目の配布がほしい」ケース（シーズン報酬・補填等）の実装が歪む。責務を publisher 側 (shop の outbox / scenario の outbox) に寄せるのは意図的な設計。

REST 経由の同期配布エンドポイント (`grant-initial-pack` / `grant-faction-pack`) は ADR-026 で account REST が縮退した時点で実 caller を失っており、ADR-032 で完全削除された。配布は scenario の `player-onboarded` event と shop の `card-pack-purchased` event (今後追加) を経由した Pub/Sub 駆動のみで行われる。

## Presenter 層の位置づけ

`internal/presenter/` は domain ↔ wire DTO (`packages/api-card`) の境界変換を集約するパッケージ。usecase / handler / repository から変換ロジックを物理的に分離し、wire 表現の変更が業務層に波及しないようにする。

**現状は厳密な Presenter パターンではない。** Uncle Bob クリーンアーキテクチャ原典の Presenter は output port (interface) を介して usecase が結果を「押し出す」構造を取り、usecase 層は wire DTO 型を一切 import しない。本サービスでは usecase が presenter 関数を直接呼び、戻り値で wire DTO を返すため、依存方向としては usecase → wire DTO 型への参照が残っている。実態は Mapper パターンに近い。

この折衷を選んだ理由:

- Go 慣用は「戻り値で返す」スタイルを好み、output port の副作用ベース設計とは噛み合わせが悪い
- wire 形式が REST のみで複数 wire (gRPC / GraphQL) の差し替え要件が現状ない
- 厳密な Presenter は endpoint ごとに output port interface と presenter struct が必要になり、3 リポ × N endpoint の規模では割に合わない

**将来の移行パス。** 複数 wire 形式の差し替えが必要になった時点で、以下の順で段階的に厳密 Presenter へ昇格できる:

1. presenter 関数のシグネチャを output port interface (`type XxxOutput interface { Present(...) }`) に置き換える
2. usecase struct に output port を依存注入し、戻り値返却を `s.output.Present(...)` 呼び出しに差し替える
3. handler 側で wire 形式ごとに presenter struct を実装 (`JSONPresenter` / `GRPCPresenter`)、endpoint 構築時に注入

現状の package 配置 (`internal/presenter/`) と命名はこの移行を阻害しない。usecase の wire DTO への依存を切り離す改修だけで Presenter パターンに到達できる。

## テスト戦略

usecase 層は in-memory mock で仕様ベースに単体テストする。repository 層は **Testcontainers で postgres:16-alpine を起動して実 PostgreSQL に対して検証する**。理由:

- `pgxpool.Pool.Query` はコンパイル時のカラム名チェックがなく、SQL 誤りは実 DB でしか検出できない
- `deck_cards` の `ON DELETE CASCADE`、`player_cards` の UPSERT (`ON CONFLICT ... DO UPDATE`)、`processed_events` の冪等性ガードなど、SQL レイヤーの仕様は実 DB でしか検証できない
- `db/schema.sql` の drift (production とテストでスキーマが乖離) を起動時に fail-fast で検出できる

helper は [internal/repository/postgrestest/postgres.go](../internal/repository/postgrestest/postgres.go)。`TestMain` でコンテナを 1 回起動し、各テストは `sharedPg.Truncate(t)` で状態をリセットする。`information_schema` を走査して BASE TABLE を動的に TRUNCATE するため、テーブル追加時に helper 側を更新する必要はない。

`search_path TO card, public` を `pgxpool.AfterConnect` で設定し、`FROM card_definitions` のような未修飾参照がスキーマ配下で解決されるようにしている（`processed_events` だけ `card.` 修飾で書かれているコード側の不整合をテスト側で吸収する形）。

## 運用

### 環境変数

環境変数の一覧と必須条件は [internal/config/config.go](../internal/config/config.go) が SSoT（`FromEnv` が起動時に検証、欠ければ即 fail）。

運用上の注意点のみ:

- **`PORT`**: 起動ポート。未設定で起動不可
- **`ENV`**: `local` / `dev` / `stg` / `prod` のいずれか。未設定で起動不可。`prod` / `stg` は Cloud Logging 互換の JSON slog、`local` / `dev` はテキスト slog にルーティングする
- **`DATABASE_CONN`**: card スキーマへの接続文字列。未設定で起動不可
- **`DATABASE_IAM_AUTH_ENABLED`**: Cloud SQL への接続を Cloud SQL Go Connector 経由の自動 IAM データベース認証で行うかどうか。`true` / `false` のいずれか必須で、フォールバックは無い
- **`CLOUDSQL_CONNECTION_NAME`**: Cloud SQL インスタンスの接続名 (`project:region:instance`)。`DATABASE_IAM_AUTH_ENABLED=true` のときのみ必須
- **`INTERNAL_AUTH_PUBLIC_KEY`**: gateway が発行する内部認証 JWT (RS256) を検証する公開鍵。PEM 形式。未設定で起動不可。`/api/v1/cards` 配下の player-scoped API は、対応する秘密鍵で署名された `X-Internal-Auth` header を要求する
- **`ACCOUNT_SERVICE_URL`**: デッキ作成・更新時に faction 所持を照会する account サービスの URL。未設定で起動不可

### カードデータ変更時

1. `data/cards/*.yaml` を編集
2. `python3 scripts/generate_cards.py` を実行して `db/seed/cards_seed.sql` / `data/cards_gen.json` を更新
3. コミット（CI の `codegen-sync` ジョブが drift を検出する）
4. デプロイ後に ops の `db-migrate` を走らせて seed を反映

### card_pack マスター変更時

1. `data/card_packs.yaml` を編集（pack の追加・カード追加 / 削除・配布枚数変更・`is_active` 切り替え等）
2. `python3 scripts/generate_card_packs.py` を実行して `db/seed/card_packs_seed.sql` を更新
3. コミット
4. デプロイ後に ops の `db-migrate` を走らせて seed を反映

`is_active=false` で pack を非アクティブ化する際の運用順序は ADR-032「pack 非アクティブ化の運用順序」に従う（shop product を先に停止 → outbox drain → card_pack を停止）。

### 型定義変更時

1. `data/openapi.yaml` を編集 (SSoT)
2. `scripts/generate_types.sh` を実行 (oapi-codegen / openapi-typescript / NSwag が `packages/api-card` / `api-card-npm` / `api-card-dotnet` の生成物を再生成)
3. enum 追加・削除があれば `internal/domain/card_enum.go` も更新 (drift test が乖離を検知)
4. コミット

### Pub/Sub トピックと subscriber

| トピック | 購読名 | push 先エンドポイント | publisher |
|---|---|---|---|
| `player-onboarded` | `player-onboarded-card-sub` | `/internal/v1/pubsub/player-onboarded` | scenario |
| `card-pack-purchased` | `card-pack-purchased-card-sub` | `/internal/v1/pubsub/card-pack-purchased` | shop (ADR-032 PR 2 で追加予定) |

publisher 列はこのリポジトリからは導けないので、変更時は各サービスの発行状況も確認すること。

旧 `faction-purchased` topic は ADR-031 で `card-pack-purchased` (card 向け) と `faction-acquired` (account / gateway 向け) に分割されたため、card 側は購読しない。
