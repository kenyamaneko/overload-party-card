# Branching Strategy

ブランチ戦略そのものは
[repos.yaml](https://github.com/kenyamaneko/overload-party-common/blob/main/rules/repos.yaml) が
`flow: github-flow` と宣言しており、共通ルール (`keyandnotes-rules` の `rules/flow/github-flow.md`) に従う。

## ブランチ保護設定

GitHub Ruleset で `main` に以下を設定している。

- 直接プッシュ禁止、PR マージのみ
- 強制プッシュ禁止、削除禁止
- 必須ステータスチェック: `ci / lint`, `ci / test-unit`, `ci / test-integration`, `ci / image-scan`, `ci / codegen-sync`, `ci / test-python`
- 必須レビュー数: 0 (一人開発のため。複数人化する際に 1 へ引き上げる)

チェック名の `ci` は `ci.yaml` の呼び出し側ジョブ名で、続く名前は common の `go-service-ci.yaml` のジョブ名 (`test-python` はカードデータ生成スクリプトのテスト)。

## CI/CD パイプライン

| ワークフロー | トリガー | 役割 |
|---|---|---|
| [ci.yaml](../.github/workflows/ci.yaml) | PR: main / workflow_call (`test-catalog.yaml` が呼ぶ) | lint + テスト + 脆弱性スキャン + コード生成ドリフト検出。中身は common の `go-service-ci.yaml` に集約している |
| [test-catalog.yaml](../.github/workflows/test-catalog.yaml) | push: main | `ci.yaml` を呼び、そのテスト結果からテスト観点カタログを生成して GitHub Pages に公開 |
| [deploy.yaml](../.github/workflows/deploy.yaml) | push: main / タグ `v*.*.*` / workflow_dispatch (`version` に prod へ昇格するタグを指定) | main への push で dev、タグで stg へデプロイし、`version` で指定したタグを prod へ昇格する。stg と prod のデプロイの後に `publish-master-data.yaml` を呼ぶ |
| [publish.yaml](../.github/workflows/publish.yaml) | workflow_dispatch | `target` で選んだ `packages/api-card` (Go) / `packages/api-card-npm` / `packages/api-card-dotnet` のタグ付けと公開。上げる桁は `bump` で選ぶ |
| [publish-master-data.yaml](../.github/workflows/publish-master-data.yaml) | push: main (`data/**` / カード・商品生成スクリプト変更時) / workflow_dispatch / workflow_call (`deploy.yaml` が stg・prod のデプロイの後に呼ぶ) | カードマスターデータの再生成と、環境ごとのバケットへの公開 |

`feature/*` へ直接プッシュしても CI は走らない。main 宛の PR を作ると実行され、追加のプッシュのたびに再実行される。

サービス本体のタグ発行は common の `scripts/create-release-tag.sh` を人が手動実行する ([docs/operations/SERVICE_RELEASE.md](https://github.com/kenyamaneko/overload-party-common/blob/main/docs/operations/SERVICE_RELEASE.md))。
