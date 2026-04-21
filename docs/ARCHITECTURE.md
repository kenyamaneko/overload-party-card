# Card サービス設計

本ドキュメントは **コードを読んでも一見しては分からない設計意図** だけを残す。実装詳細（バリデーション順序・state 遷移の対応・エラー → HTTP ステータス変換・環境変数の一覧）は [FEATURE_SPEC.md](FEATURE_SPEC.md) と各ファイルの実装・コメントを一次情報とする。

サービス概要・起動手順は [../README.md](../README.md)、エンドポイントは [API_REFERENCE.md](API_REFERENCE.md)、テーブル定義は [DATA_DESIGN.md](DATA_DESIGN.md) を参照。

## Card の責務境界 (SSoT 契約)

card は **カードマスターデータそのもの** の single source of truth である。他サービスはカード定義を DB テーブル/seed として持たず、card の `GET /internal/v1/cards` を起動時に呼び出してインメモリに保持する read model を各自で構築する。

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

カードの意味定義（faction / card_type / subtype / stats JSON の構造等）は [game_design/CARDS.md](game_design/CARDS.md) を参照。

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

## `is_valid` を DB に保存しない判断

デッキの `is_valid` は API レスポンス生成時に都度算出する。DB には持たない。

保存すると次の場合にスタッルレスポンスが発生する:

- プレイヤーが `restriction` を超過させるカードを所持し直した／失った
- game designer が `restriction` を `unlimited → limited` に引き下げた（バン）
- 新カード追加で過去デッキが依然合法であることを確認したい

算出はデッキ単位で O(|deck_cards| + |player_cards|) の map lookup。デッキ一覧取得でも実測でボトルネックにならないため、正確性を優先して常時再計算している。

## `faction-selected` subscriber の冪等性

card は `faction-selected-card-sub` を購読してパック配布をイベント駆動で実行する。冪等性は `card.processed_events` テーブルに `event_id` を INSERT することで前段ガードとして機能させる。

ただし **完全な idempotency 保証ではない**:

- `GrantService` の `player_cards` UPSERT は独自トランザクションを張る
- `processed_events` INSERT と UPSERT を同一 tx にまとめると、GrantService が `port.PlayerCardRepo` を通じて内部で tx 管理するという責任分界が壊れる
- そこで processed_events は「**再試行の早期打ち切り用 fast-path**」と位置付け、grant 実行中のクラッシュ窓では稀に 2 重加算がありうる（publisher 側のリトライ間隔が短く、実用上の影響は限定的）

at-least-once を at-most-once 相当に近づけるための前段ガードであり、strict exactly-once を謳っていない点に注意。将来 GrantService を processed_events と同一 tx に組み込む設計変更で完全 idempotent にできる（現状は実用上の優先度が低い）。

## クロススキーマ参照の方針

`player_cards` / `decks` / `deck_cards` の `player_id` は `account.players` へのクロススキーマ参照だが **FK は張らない**。理由:

- account と card は別サービスで独立デプロイされる。外部 FK は デプロイ順序依存を生む
- 整合性は app 層で守る（プレイヤー作成前に card API が呼ばれた場合は 500 系ではなく INSERT 失敗扱い）
- `card_definitions.card_id` も `player_cards.card_id` から FK なしで参照される。カード論理削除（`is_active=false`）時に所持データが壊れないようにするため

`deck_cards` → `decks` は同一スキーマ内なので通常の FK + `ON DELETE CASCADE` を張っている（デッキ削除で構成カードを自動削除）。

## カード配布 API の非冪等契約

`grant-initial-pack` / `grant-faction-pack` は **呼ばれた回数ぶん 3 枚ずつ加算する**。自前で冪等性を持たない。

REST で直接呼ばれるケースでの冪等性は gateway のオーケストレーション層が担保する（`account.player_factions` の存在をもって重複呼び出しを抑止）。Pub/Sub 経由のケースは上述 processed_events で前段ガードする。

「配布 API 自体を冪等にする」方向に倒すと、「本当に 2 回目の配布がほしい」ケース（シーズン報酬・補填等）の実装が歪む。責務を呼び出し側に寄せるのは意図的な設計。

## game_config の Firestore 化

カードマスター以外の「ゲーム設計値」（配布コピー数・デッキサイズ上限等、頻繁に調整したいパラメータ）は Cloud Firestore から読み取る（`FIRESTORE_PROJECT_ID`）。card リポジトリのコードを触らずに ops 側から値変更ができるようにするため。

ローカル・CI では `FIRESTORE_EMULATOR_HOST` でエミュレーターに接続する（`validate.yaml` / `ci.yaml` が emulator を起動）。

カードマスター（`card_definitions`）は従来通り YAML + PostgreSQL seed のままで、Firestore には載せない。カード定義は PR レビュー対象とし、game_config は運用時パラメータとして分けている。

## 運用

### 環境変数

環境変数の一覧と必須条件は [internal/config/config.go](../internal/config/config.go) が SSoT（`FromEnv` が起動時に検証、欠ければ即 fail）。

運用上の注意点のみ:

- **`DATABASE_URL`**: card スキーマへの接続文字列。未設定で起動不可
- **`PUBSUB_PROJECT_ID`**: card は `faction-selected` subscriber を持つため必須
- **`FIRESTORE_PROJECT_ID`**: game_config 読み取り先。未設定で起動不可
- **`FACTION_SELECTED_SUBSCRIPTION`**: デフォルト `faction-selected-card-sub`。本番以外で別名を使う場合に上書き

### カードデータ変更時

1. `data/cards/*.yaml` を編集
2. `python3 scripts/generate_cards.py` を実行して `db/seed/cards_seed.sql` / `data/cards_gen.json` を更新
3. コミット（`validate.yaml` が drift を検出する）
4. デプロイ後に ops の `db-migrate` を走らせて seed を反映

### 型定義変更時

1. `data/models.yaml` を編集
2. `python3 scripts/generate_types.py` を実行
3. `packages/api-card/*_gen.go` と `docs/API_REFERENCE.md` のマーカー間を更新
4. コミット

### Pub/Sub トピックと subscriber

| トピック | 購読名 | publisher |
|---|---|---|
| `faction-selected` | `faction-selected-card-sub` | scenario / shop |

publisher 列はこのリポジトリからは導けないので、変更時は各サービスの発行状況も確認すること。
