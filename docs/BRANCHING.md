# Branching Strategy

本リポジトリのブランチ戦略とリリース運用を定義する。

> **Note**: このドキュメントは将来 `overload-party-common` に移動する予定。他リポジトリの main ブランチの品質が安定した段階で、共通ルールとして参照される形にする。

> **現状との差分**: card リポジトリは現時点で `develop` / `release/*` のブランチ保護と対応する CI ワークフロー (`deploy.yaml` / `release-tag.yaml`) が整備されていない。本ドキュメントは **目標状態** を定義する。段階的に shop と同じ構成に揃えていく前提で読むこと。

## 概要

GitFlow をベースに、環境とブランチを対応付けた運用を採用する。card はゲームバランスの改定（restriction 変更・新カード追加）が本番のプレイヤー体験に直結するため、stg 環境での実機検証を挟む昇格モデルを敷く。

## ブランチ一覧

| ブランチ | 環境 | 寿命 | 派生元 | マージ先 | 保護 |
|---|---|---|---|---|---|
| `main` | prod | 永続 | — | — | 最大 |
| `release/vX.Y.Z` | stg | 短命 | `develop` | `main` | あり |
| `develop` | dev | 永続 | `main` (初回のみ) | — | あり |
| `feature/xxx` | なし | 短命 | `develop` | `develop` | なし |
| `hotfix/xxx` | なし | 短命 | `main` | `main` + `develop` (+ `release` if exists) | なし |

## ブランチ運用ルール

### main

- **prod 環境のソース・オブ・トゥルース**。main の HEAD = prod で動作しているコード
- 直 push 禁止。PR 経由のマージのみ
- マージ元として許可するのは `release/*` と `hotfix/*` のみ
- `develop` や `feature/*` を直接 main にマージしない
- タグは CI が自動で打つ。手動タグ付けは禁止
- force push 禁止、履歴書き換え禁止

### develop

- **dev 環境のソース**。次リリースに向けた統合ブランチ
- 直 push 禁止。PR 経由のマージのみ
- マージ元として許可するのは `feature/*` と `hotfix/*` の back-merge
- CI green 必須。レビューは self-approve 可（速度優先）

### release/vX.Y.Z

- **stg 環境のソース**。リリース候補の検証ブランチ
- 短命。main にマージ後、削除する
- ブランチ名に候補バージョンを含める（例: `release/v1.2.0`）
- `develop` から切る。切った時点で feature の取り込みは停止する
- release 中に feature を追加で取り込みたい場合は、原則として次の release に回す
- バグ修正やリリース準備（カードマスターの最終調整等）のコミットは PR 経由で release に入れる
- release に入れた修正は、main マージ後に develop にも back-merge する（後述）

### feature/xxx

- 新機能・改善の作業ブランチ
- `develop` から切って `develop` にマージ
- 命名例: `feature/add-faction-pack-preview`, `feature/card/issue-123`
- PR マージ時にブランチ削除

### hotfix/xxx

- **prod 緊急修正**の作業ブランチ
- `main` から切る（develop からではない。develop には未リリース変更が混ざっているため）
- main と develop の両方にマージする（back-merge 必須）
- release ブランチが存在する場合は、release にもマージする
- 命名例: `hotfix/fix-deck-validation-crash`, `hotfix/critical-pack-grant-bug`

## リリースフロー

### 通常リリース

```
1. develop で feature を統合・dev 環境で検証
   └─ feature/xxx → develop (PR)

2. release ブランチを切る
   └─ git switch -c release/v1.2.0 develop
   └─ push → stg 環境に自動デプロイ

3. stg 環境で検証
   └─ デッキ CRUD、パック配布、card-pack-purchased / player-onboarded イベント疎通確認など
   └─ バグ発見時は PR 経由で release ブランチに修正を入れる

4. main にマージ
   └─ release/v1.2.0 → main (PR)
   └─ CI が自動でタグ v1.2.0 を打つ
   └─ main が prod 環境に自動デプロイ

5. develop に back-merge
   └─ release/v1.2.0 → develop (PR)
   └─ release 中に入れた修正を develop に戻す

6. release ブランチ削除
```

### hotfix リリース

```
1. hotfix ブランチを切る
   └─ git switch -c hotfix/fix-deck-500 main

2. 修正 → PR → main にマージ
   └─ hotfix/xxx → main (PR)
   └─ CI が自動でタグ v1.2.1 を打つ（patch bump）
   └─ prod 環境に自動デプロイ

3. develop に back-merge（必須）
   └─ hotfix/xxx → develop (PR)

4. release ブランチが存在する場合は release にも back-merge
   └─ hotfix/xxx → release/vX.Y.Z (PR)

5. hotfix ブランチ削除
```

### hotfix の back-merge 忘れ対策

hotfix を main にマージしたが develop に戻し忘れると、次のリリースでバグが再発する。

対策:

- PR テンプレートに back-merge チェックリストを入れる
- main に hotfix が入ったら、CI で develop への back-merge PR を自動生成する workflow を用意する（未作成）

### カードマスター変更のリリース扱い

`data/cards/*.yaml` の変更は通常の feature フローで扱う。ただし既存カードの `restriction` 変更（いわゆるバンリスト改定）は **プレイヤーの既存デッキを即日 invalid 化しうる** ため、リリースノートに明記し release で stg 検証を行うこと。新カード追加は既存デッキに影響しないので単純な MINOR bump で良い。

## バージョニング

Semantic Versioning (SemVer) を採用する。

- **MAJOR**: 破壊的変更（REST API スキーマ破壊、DB マイグレーション等、既存クライアントが動かなくなる変更）
- **MINOR**: 後方互換のある機能追加（新カード追加含む）
- **PATCH**: バグ修正、ドキュメント修正、内部リファクタ

サービス本体のタグは release マージ・hotfix マージ時に CI が自動で打つ前提（`release-tag.yaml` 相当は未整備）。

- release マージ時: ブランチ名からバージョンを取得（`release/v1.2.0` → `v1.2.0`）
- hotfix マージ時: 最新タグから patch を自動 bump（`v1.2.0` → `v1.2.1`）

`packages/api-card` / `api-card-npm` / `api-card-dotnet` のタグは [publish.yaml](../.github/workflows/publish.yaml) の `workflow_dispatch` で人がタイミングを判断して発行する（patch / minor / major bump を指定）。

### バージョンと Go module の関係

`packages/api-card` は Go module として独立のバージョンを持つ。サービス本体のバージョンと必ずしも一致しないが、破壊的変更を含む release では api-card も major bump することを推奨する。

## ブランチ保護設定（目標状態）

GitHub Rulesets で以下を設定する。

### main

- 直 push 禁止
- PR マージのみ許可（linear history）
- force push 禁止、削除禁止
- 履歴書き換え禁止
- 必須ステータスチェック: ci / lint, ci / test-unit, ci / test-integration, ci / image-scan, ci / codegen-sync, ci / test-python が green
- required reviews: 1（self-approve 不可）
- マージ元ブランチ制限: `release/*` と `hotfix/*` のみ

チェック名の `ci` は `ci.yaml` の呼び出し側ジョブ名で、続く名前は common の `go-service-ci.yaml` のジョブ名。

### release/*

- 直 push 禁止。PR 経由のマージのみ
- force push 禁止、削除は手動で可
- 必須ステータスチェック: ci / lint, ci / test-unit, ci / test-integration, ci / codegen-sync が green

### develop

- 直 push 禁止
- PR マージのみ許可
- 必須ステータスチェック: ci / lint, ci / test-unit, ci / test-integration, ci / codegen-sync が green
- required reviews: 不要（一人開発での速度優先）

## CI/CD パイプライン

現状 card リポジトリに存在するワークフロー:

| ワークフロー | トリガー | 役割 |
|---|---|---|
| [ci.yaml](../.github/workflows/ci.yaml) | PR: main | lint + テスト + 脆弱性スキャン + codegen drift 検出。中身は common の `go-service-ci.yaml` に集約している |
| [test-catalog.yaml](../.github/workflows/test-catalog.yaml) | push: main | `ci.yaml` を呼び、そのテスト結果からテスト観点カタログを生成して GitHub Pages に公開 |
| [deploy.yaml](../.github/workflows/deploy.yaml) | push: main | Docker イメージのビルド・push |
| [publish.yaml](../.github/workflows/publish.yaml) | workflow_dispatch | api-card (Go) / api-card-npm (npm) / api-card-dotnet (NuGet) のタグ付け・公開 |

目標状態との差分 (**未整備**):

- `ci.yaml` / `deploy.yaml` のトリガに `develop`, `release/*` が含まれていない
- `release-tag.yaml` 相当が未整備（release/hotfix マージ時のサービスタグ自動発行なし）

段階的に shop と同じ構成に近づけていく。

### feature / hotfix ブランチの CI

feature/* や hotfix/* ブランチへの push では CI は走らない。これらのブランチで CI を実行するには、対象ブランチ（develop / main）宛の PR を作成する。PR 更新時（追加 push）にも CI が再実行される。
