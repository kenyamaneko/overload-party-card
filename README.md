# overload-party-card

カードマスターデータ（SSoT）・プロダクトマスターデータ（SSoT）・所持カード・デッキ CRUD・デッキバリデーション・カードパック配布を担う内部マイクロサービス。ポート 9003 で起動する。

詳細は [API 契約 (OpenAPI)](data/openapi.yaml) / [データ設計書](docs/DATA_DESIGN.md) / [カードデータ仕様](docs/CARDS.md) / [プロダクトデータ仕様](docs/PRODUCTS.md) / [ブランチ・CI/CD](docs/BRANCHING.md) を参照。設計判断 (Why) は [common の ADR](https://github.com/kenyamaneko/overload-party-common/tree/main/docs/adr) に記録する。

[テスト観点カタログ](https://kenyamaneko.github.io/overload-party-card/): テスト名から生成した、テスト済みの観点の一覧。

## アーキテクチャ概要

```
Gateway
  └─ Card (:9003)
       ├─ PostgreSQL (card スキーマ)
       ├─ Account (デッキ作成・更新時の faction 所持照会)
       └─ Pub/Sub
            ├─ card-pack-purchased-card-sub ← shop が発行
            └─ player-onboarded-card-sub    ← scenario が発行

Battle (:9002)
  └─ GET /internal/v1/cards  ← 起動時 1 回のカードマスターロード
```

- gateway と battle が REST の呼び出し元。card はデッキ作成・更新時に account へ faction 所持を REST で照会する
- カードマスター (`card_definitions`) の SSoT は本サービス。他サービスには複製しない
- `card-pack-purchased` / `player-onboarded` を購読し、パック配布をイベント駆動で実行する

## ローカル開発

```bash
make run   # card server を起動（DATABASE_CONN 等の env 必須）
make test  # go test -race（Testcontainers で Postgres を立てるので Docker 必須）
make vet   # go vet
make fmt   # gofmt -s -w
```

`ENV` (`local` / `dev` / `stg` / `prod`) をはじめとする必須環境変数（一覧は [internal/config/config.go](internal/config/config.go)）は未設定なら起動時に fail する（デフォルトへの暗黙 fallback を行わない）。`make run` は `ENV=dev` をインラインで設定するので、残りの必須環境変数を export すればよい。

## カードマスター変更

```bash
python3 scripts/generate_cards.py  # data/cards/*.yaml → db/seed/cards_seed.sql
```

生成器は効果定義のリソースの括り (`group`) を展開し、`card_type` / `subtype` が正規値かを検証するために common のゲーム定数を読む。既定では兄弟ディレクトリの `../overload-party-common` を見るので、別の場所に置いているときは `COMMON_REPO` にそのパスを渡す。

既存カードの `restriction` 変更 (バンリスト改定) はプレイヤーの既存デッキを即日 invalid 化しうるため、リリースノートに明記した上で stg 環境での検証を行う。
