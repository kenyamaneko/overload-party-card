# セットアップ

## ローカル開発

```bash
make run              # card サービスを起動（DATABASE_CONN 等の環境変数が必須）
make test             # go test -race（単体テスト）
make test-integration # 単体テストと結合テスト（Testcontainers で Postgres を立てるので Docker 必須）
make vet              # go vet
make fmt              # gofmt -s -w
```

`ENV` (`local` / `dev` / `stg` / `prod`) をはじめとする必須環境変数（一覧は [internal/config/config.go](../internal/config/config.go)）は未設定なら起動時に失敗する。`make run` は `ENV=dev` をインラインで設定するので、残りの必須環境変数を export すればよい。

## カードマスター変更

```bash
COMMON_REPO=../overload-party-common BATTLE_REPO=../overload-party-battle python3 scripts/generate_cards.py  # data/cards/*.yaml → db/seed/cards_seed.sql
```

生成器は効果定義のリソースの括り (`group`) を展開し、`card_type` / `subtype` が正規値かを検証するために common のゲーム定数を、`trigger` / `op` / `buff` / `selector` / `owner` / `duration` が定義された値かを検証するために battle のゲームロジック定数を読む。common と battle を置いた場所を環境変数 `COMMON_REPO` と `BATTLE_REPO` に渡すことが必須で、未設定だとエラーで終了する。

既存カードの `restriction` 変更 (バンリスト改定) はプレイヤーの既存デッキを即日無効化しうるため、リリースノートに明記した上で stg 環境で検証する。
