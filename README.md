# overload-party-card

カードマスターデータ + デッキ CRUD + デッキバリデーション + カードデータ SSoT。Gateway と Battle から内部 REST で呼ばれる。Pub/Sub で faction-selected イベントを受信し、カードパック配布を行う。

## サービス間連携

```
Gateway (:9001)
  ├─ GET  /internal/v1/cards                           ← カード全件 (forward)
  ├─ GET  /internal/v1/players/:id/cards               ← 所持カード
  ├─ GET  /internal/v1/players/:id/cards/with-ownership ← 全カード+所持フラグ
  ├─ CRUD /internal/v1/players/:id/decks/*             ← デッキ CRUD
  ├─ POST /internal/v1/players/:id/decks/:id/validate-for-battle
  ├─ POST /internal/v1/players/:id/grant-initial-pack  ← 初回ファクション選択
  └─ POST /internal/v1/players/:id/grant-faction-pack  ← ショップ購入
              │
              ▼
Card (このサービス, :9003)
  ├─ PostgreSQL  card スキーマ (card_definitions / player_cards /
  │                             decks / deck_cards / processed_events)
  └─ Cloud Pub/Sub subscriber
        └─ faction-selected-card-sub  ← faction_selected
              │
Battle (:9002)
  └─ GET  /internal/v1/cards  ← 起動時 1 回のカードロード
```

- Gateway と Battle が呼び出し元。card 自身は他サービスを REST で呼び出さない
- `faction-selected` の Pub/Sub subscriber を持ち、カード配布をイベント駆動で実行

エンドポイント一覧は [docs/API_REFERENCE.md](docs/API_REFERENCE.md) を参照。

## 環境変数

全て必須。未設定なら起動時に即 fail する。

**Deployment env (インフラ層):**

| 変数名 | デフォルト | 説明 |
|---|---|---|
| `PORT` | `9003` | リッスンポート |
| `ENV` | `dev` | 動作環境 (`dev` / `stg` / `prod`) |
| `DATABASE_URL` | *(必須)* | PostgreSQL 接続文字列 (`card` スキーマ) |
| `PUBSUB_PROJECT_ID` | *(必須)* | GCP プロジェクト ID |

**ConfigMap (Pub/Sub):**

| 変数名 | デフォルト | 説明 |
|---|---|---|
| `FACTION_SELECTED_SUBSCRIPTION` | `faction-selected-card-sub` | faction-selected Pub/Sub サブスクリプション名 |

## 公開パッケージ

| パッケージ | 言語 | 説明 |
|---|---|---|
| `packages/api-card/` | Go | gateway が import する REST 型 |

SSoT: `data/models.yaml` → `python3 scripts/generate_types.py` で再生成。`*_gen.go` は自動生成 — 直接編集しない。

クライアント向け TypeScript 型は `@kenyamaneko/overload-party-api-gateway` に統合済み。
