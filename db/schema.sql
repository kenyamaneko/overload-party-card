-- overload-party-card - PostgreSQL DDL (service-owned)
--
-- Scope (ADR-014):
--   card.card_definitions             - カードマスター
--   card.products                     - プロダクトマスター
--   card.initiatives                  - 施策マスター
--   card.player_cards                 - プレイヤーの所持カード
--   card.decks                        - プレイヤーのデッキヘッダー
--   card.deck_cards                   - デッキ内カード構成
--
-- This file is the card service's source of truth. psqldef 互換。
-- Cross-schema 参照（例: player_id -> account.players）は FK を張らず
-- アプリ層整合性で担保する（ADR-014）。

CREATE SCHEMA IF NOT EXISTS card;

-- =============================================================================
-- Schema-local helpers
-- =============================================================================

CREATE OR REPLACE FUNCTION card.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- Card Master (schema: card)
-- =============================================================================

CREATE TABLE card.card_definitions (
  card_id        VARCHAR(10) NOT NULL,               -- カード識別子（例: SH-0001）
  card_name      VARCHAR(100) NOT NULL,              -- カード名
  resource_label VARCHAR(30) NOT NULL,                -- リソースラベル
  faction        VARCHAR(20) NOT NULL CHECK (faction IN ('SHE', 'Tenki', 'Sugar', 'Tuners', 'Neutral')), -- 陣営（SHE / Tenki / Sugar / Tuners / Neutral）
  card_type      VARCHAR(30) NOT NULL,               -- カードタイプ（Resource / Support）
  subtype        VARCHAR(30),                        -- サブタイプ（Compute/Data カテゴリのみ設定: VM/Container/Database 等）
  resizable      BOOLEAN NOT NULL,                   -- Resizable 属性
  elastic        BOOLEAN NOT NULL,                   -- Elastic 属性
  stats          JSONB NOT NULL,                     -- ステータス定義
  effect_text    VARCHAR(500),                       -- 効果テキスト（表示用）
  effects        JSONB,                              -- 効果定義（JSON 配列）
  restriction    VARCHAR(20) NOT NULL,               -- 制限区分（unlimited / semi_limited / limited / forbidden）
  is_active      BOOLEAN NOT NULL,                   -- 有効フラグ
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(), -- 作成日時
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(), -- 更新日時
  PRIMARY KEY (card_id)
);

CREATE INDEX idx_cards_faction ON card.card_definitions(faction, card_type);
CREATE INDEX idx_cards_type ON card.card_definitions(card_type);
CREATE TRIGGER trg_card_definitions_updated_at BEFORE UPDATE ON card.card_definitions FOR EACH ROW EXECUTE FUNCTION card.update_updated_at();

-- =============================================================================
-- Product Master (schema: card)
-- =============================================================================
-- 陣営に属するプロダクトと、その施策 (ルーチン / スペシャル) の SSoT。
-- 陣営:プロダクト = 1:N（陣営は products.faction 列で表現）、
-- プロダクト:施策 = 1:N（initiatives.product_id で親を参照）。
-- decks は product_id / routine_id / special_id を論理参照として持ち、整合性は
-- アプリ層で担保する（card_id と同様、FK は張らない）。

CREATE TABLE card.products (
  product_id   VARCHAR(10) NOT NULL,               -- プロダクト識別子（例: PD-0001）
  faction      VARCHAR(20) NOT NULL CHECK (faction IN ('SHE', 'Tenki', 'Sugar', 'Tuners')), -- 所属陣営
  product_name VARCHAR(100) NOT NULL,              -- プロダクト名
  is_active    BOOLEAN NOT NULL DEFAULT true,       -- 有効フラグ（論理削除）
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(), -- 作成日時
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(), -- 更新日時
  PRIMARY KEY (product_id)
);

CREATE INDEX idx_products_faction ON card.products(faction);
CREATE TRIGGER trg_products_updated_at BEFORE UPDATE ON card.products FOR EACH ROW EXECUTE FUNCTION card.update_updated_at();

CREATE TABLE card.initiatives (
  initiative_id VARCHAR(10) NOT NULL,              -- 施策識別子（例: IN-0001）
  product_id    VARCHAR(10) NOT NULL,              -- 親プロダクト識別子
  kind          VARCHAR(10) NOT NULL CHECK (kind IN ('routine', 'special')), -- 区分（routine: 1ターン1回 / special: 1ゲーム1回）
  name          VARCHAR(100) NOT NULL,             -- 施策名
  insight_cost  BIGINT NOT NULL CHECK (insight_cost >= 0), -- 発動 Insight コスト
  effect_text   VARCHAR(500) NOT NULL,             -- 効果テキスト（表示用）
  effect        JSONB NOT NULL,                    -- 効果定義（DSL）
  is_active     BOOLEAN NOT NULL DEFAULT true,     -- 有効フラグ（論理削除）
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),-- 作成日時
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),-- 更新日時
  PRIMARY KEY (initiative_id),
  FOREIGN KEY (product_id) REFERENCES card.products(product_id) ON DELETE CASCADE
);

CREATE INDEX idx_initiatives_product ON card.initiatives(product_id);
CREATE TRIGGER trg_initiatives_updated_at BEFORE UPDATE ON card.initiatives FOR EACH ROW EXECUTE FUNCTION card.update_updated_at();

-- =============================================================================
-- Card & Deck Management (schema: card, children of players)
-- =============================================================================

CREATE TABLE card.player_cards (
  player_id  UUID NOT NULL, -- 所有プレイヤー (cross-schema reference to account.players; app-level integrity, not enforced by FK)
  card_id    VARCHAR(10) NOT NULL,                   -- カード識別子
  art_no     BIGINT NOT NULL,                        -- アート番号
  count      INT NOT NULL,                           -- 所持枚数
  PRIMARY KEY (player_id, card_id, art_no)
);

CREATE TABLE card.decks (
  player_id   UUID NOT NULL, -- 所有プレイヤー (cross-schema reference to account.players; app-level integrity, not enforced by FK)
  deck_id     BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY, -- デッキID（自動採番）
  deck_name   VARCHAR(50) NOT NULL,                  -- デッキ名
  faction     VARCHAR(20) NOT NULL,                  -- 宣言陣営（SHE / Tenki / Sugar / Tuners）
  product_id  VARCHAR(10) NOT NULL,                  -- 選択したプロダクトの ID（宣言陣営に属する）
  routine_id  VARCHAR(10) NOT NULL,                  -- セットしたルーチン施策の ID（選択プロダクトに属する）
  special_id  VARCHAR(10) NOT NULL,                  -- セットしたスペシャル施策の ID（選択プロダクトに属する）
  playmat_no  BIGINT,                                -- プレイマット番号（NULL: デフォルト）
  sleeve_no   BIGINT,                                -- スリーブ番号（NULL: デフォルト）
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),    -- 作成日時
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),    -- 更新日時
  PRIMARY KEY (player_id, deck_id)
);

CREATE INDEX idx_decks_player ON card.decks(player_id, updated_at DESC);
CREATE TRIGGER trg_decks_updated_at BEFORE UPDATE ON card.decks FOR EACH ROW EXECUTE FUNCTION card.update_updated_at();

CREATE TABLE card.deck_cards (
  player_id  UUID NOT NULL,                          -- ルート親参照
  deck_id    BIGINT NOT NULL,                        -- 親テーブル参照
  card_id    VARCHAR(10) NOT NULL,                   -- カード識別子
  art_no     BIGINT NOT NULL,                        -- アート番号
  count      INT NOT NULL,                           -- 枚数
  PRIMARY KEY (player_id, deck_id, card_id, art_no),
  FOREIGN KEY (player_id, deck_id) REFERENCES card.decks(player_id, deck_id) ON DELETE CASCADE
);

-- =============================================================================
-- Card Pack Master (schema: card)
-- =============================================================================
-- 配布パック (どのカードを何枚配るか) の SSoT。shop からは
-- shop.product_card_pack_refs.card_pack_id で論理参照される (FK なし)。
-- 配布対象カードと枚数は子テーブル card.card_pack_cards に正規化する。

CREATE TABLE card.card_pack (
  pack_id     VARCHAR(50) NOT NULL,                    -- パック識別子（例: faction_set_she）
  description VARCHAR(200) NOT NULL DEFAULT '',         -- 運営用説明
  is_active   BOOLEAN NOT NULL DEFAULT true,           -- 配布有効フラグ
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),      -- 作成日時
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),      -- 更新日時
  PRIMARY KEY (pack_id)
);

CREATE TRIGGER trg_card_pack_updated_at BEFORE UPDATE ON card.card_pack FOR EACH ROW EXECUTE FUNCTION card.update_updated_at();

CREATE TABLE card.card_pack_cards (
  pack_id VARCHAR(50) NOT NULL,                            -- 親パック識別子
  card_id VARCHAR(10) NOT NULL,                            -- 配布対象カード識別子
  copies  INT         NOT NULL CHECK (copies > 0),         -- 当該パック当該カードの配布枚数
  PRIMARY KEY (pack_id, card_id),
  FOREIGN KEY (pack_id) REFERENCES card.card_pack(pack_id) ON DELETE CASCADE
);

-- =============================================================================
-- card.processed_events (Pub/Sub subscriber idempotency)
-- =============================================================================

CREATE TABLE card.processed_events (
  event_id     UUID PRIMARY KEY,                     -- Pub/Sub EventID (publisher 生成の UUIDv4)
  event_type   TEXT NOT NULL,                        -- イベント種別 (card_pack_purchased / player_onboarded)
  processed_at TIMESTAMPTZ NOT NULL DEFAULT now()    -- 処理日時
);
