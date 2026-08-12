# セットアップ

## ローカル開発

```bash
make run   # card server を起動（DATABASE_CONN 等の env 必須）
make test  # go test -race（Testcontainers で Postgres を立てるので Docker 必須）
make vet   # go vet
make fmt   # gofmt -s -w
```

`ENV` (`local` / `dev` / `stg` / `prod`) をはじめとする必須環境変数（一覧は [internal/config/config.go](../internal/config/config.go)）は未設定なら起動時に失敗する（デフォルトへの暗黙のフォールバックを行わない）。`make run` は `ENV=dev` をインラインで設定するので、残りの必須環境変数を export すればよい。

## カードマスター変更

```bash
python3 scripts/generate_cards.py  # data/cards/*.yaml → db/seed/cards_seed.sql
```

生成器は効果定義のリソースの括り (`group`) を展開し、`card_type` / `subtype` が正規値かを検証するために common のゲーム定数を読む。既定では同じ階層の `../overload-party-common` を見るので、別の場所に置いているときは `COMMON_REPO` にそのパスを渡す。

既存カードの `restriction` 変更 (バンリスト改定) はプレイヤーの既存デッキを即日無効化しうるため、リリースノートに明記した上で stg 環境での検証を行う。
