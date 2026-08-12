# overload-party-card

カードマスターデータ（SSoT）・プロダクトマスターデータ（SSoT）・所持カード・デッキ CRUD・デッキバリデーション・カードパック配布を担う内部マイクロサービス。ポート 9003 で起動する。

詳細は [API 契約 (OpenAPI)](data/openapi.yaml) / [データ設計書](docs/DATA_DESIGN.md) / [カードデータ仕様](docs/CARDS.md) / [プロダクトデータ仕様](docs/PRODUCTS.md) / [ブランチ・CI/CD](docs/BRANCHING.md) を参照。設計判断 (Why) は [common の ADR](https://github.com/kenyamaneko/overload-party-common/tree/main/docs/adr)、サービス構成全体の図は [common のシステム構成図](https://github.com/kenyamaneko/overload-party-common#システム構成図) を参照。ローカル開発・カードマスター変更手順は [docs/SETUP.md](docs/SETUP.md) を参照。

[テスト観点カタログ](https://kenyamaneko.github.io/overload-party-card/): テスト名から生成した、テスト済みの観点の一覧。
