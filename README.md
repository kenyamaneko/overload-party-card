# overload-party-card

カードゲーム Overload Party のカードマスターデータ（SSoT）・プロダクトマスターデータ（SSoT）・所持カード・デッキ CRUD・デッキバリデーション・カードパック配布を担うマイクロサービス。

## 技術スタック

| レイヤー | 技術 |
|---|---|
| 言語 | Go |
| フレームワーク | Gin |
| データベース | Cloud SQL PostgreSQL |
| 同期通信 | REST |
| 非同期通信 | Cloud Pub/Sub |

## ドキュメント

| ドキュメント | 内容 |
|---|---|
| [セットアップ](docs/SETUP.md) | ローカル開発環境の起動手順とカードマスター変更手順 |
| [API仕様書](data/openapi.yaml) | REST API のエンドポイント定義 |
| [データ設計書](docs/DATA_DESIGN.md) | テーブル定義 |
| [カードデータ仕様](docs/CARDS.md) | 全カードの一覧 |
| [プロダクトデータ仕様](docs/PRODUCTS.md) | 全プロダクトの一覧 |
| [ブランチ・CI/CD](docs/BRANCHING.md) | ブランチ運用と CI/CD の構成 |
| [ADR](https://github.com/kenyamaneko/overload-party-common/tree/main/docs/adr)（commonリポジトリ） | 設計判断の背景・理由・結果 |
| [システム構成図](https://github.com/kenyamaneko/overload-party-common#システム構成図)（commonリポジトリ） | Overload Party 全体の構成図 |
| [テスト観点カタログ](https://kenyamaneko.github.io/overload-party-card/) | テスト名から自動生成した、テスト済みの観点一覧 |
