# overload-party-card

カードマスターデータ（SSoT）・所持カード・デッキ CRUD・デッキバリデーション・カードパック配布を担う内部マイクロサービス。ポート 9003 で起動する。

詳細は [機能仕様書](docs/FEATURE_SPEC.md) / [サービス設計書](docs/ARCHITECTURE.md) / [API 契約 (OpenAPI)](data/openapi.yaml) / [データ設計書](docs/DATA_DESIGN.md) / [カードデータ仕様](docs/CARDS.md) / [ブランチ運用](docs/BRANCHING.md) を参照。

## アーキテクチャ概要

```
Gateway
  └─ Card (:9003)
       ├─ PostgreSQL (card スキーマ)
       └─ Pub/Sub
            ├─ card-pack-purchased-card-sub ← shop が発行
            └─ player-onboarded-card-sub    ← scenario が発行

Battle (:9002)
  └─ GET /internal/v1/cards  ← 起動時 1 回のカードマスターロード
```

- gateway と battle が REST の呼び出し元。card 自身は他サービスを REST で呼び出さない
- カードマスター (`card_definitions`) の SSoT は本サービス。他サービスには複製しない
- `card-pack-purchased` / `player-onboarded` を購読し、パック配布をイベント駆動で実行する

## ローカル開発

```bash
make run   # card server を起動（DATABASE_URL 等の env 必須）
make test  # go test -race（Testcontainers で Postgres を立てるので Docker 必須）
make vet   # go vet
make fmt   # gofmt -s -w
```

`ENV` (`dev` / `stg` / `prod`) / `DATABASE_URL` / `GOOGLE_CLOUD_PROJECT_ID` は未設定なら起動時に fail する（デフォルトへの暗黙 fallback を行わない）。`make run` は `ENV=dev` をインラインで設定するので、その他 2 つを export すればよい。

## 公開パッケージ

[packages/api-card/](packages/api-card/) に gateway が import する REST 契約型を公開している。[data/openapi.yaml](data/openapi.yaml) (SSoT) を編集後に以下で再生成する。

```bash
scripts/generate_types.sh
```

`openapi_gen.go` は oapi-codegen の出力 — 直接編集しない。クライアント向け TypeScript 型は `@kenyamaneko/overload-party-api-gateway` に統合済み。

## カードマスター変更

```bash
python3 scripts/generate_cards.py  # data/cards/*.yaml → db/seed/cards_seed.sql
```

詳細は [ARCHITECTURE.md#カードデータ-ssot-フロー](docs/ARCHITECTURE.md#カードデータ-ssot-フロー) を参照。
