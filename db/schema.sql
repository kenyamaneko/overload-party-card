-- overload-party-card - PostgreSQL DDL (service-owned)
--
-- Scope (ADR-014):
--   card.card_definitions             - カードマスター
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
  resource_label VARCHAR(30) NOT NULL DEFAULT '',     -- リソースラベル
  faction        VARCHAR(20) NOT NULL CHECK (faction IN ('SHE', 'Tenki', 'Sugar', 'Tuners', 'Neutral')), -- 陣営（SHE / Tenki / Sugar / Tuners / Neutral）
  card_type      VARCHAR(30) NOT NULL,               -- カードタイプ（Resource / Support）
  subtype        VARCHAR(30),                        -- サブタイプ（Compute/Data カテゴリのみ設定: VM/Container/Database 等）
  resizable      BOOLEAN NOT NULL DEFAULT false,     -- Resizable 属性
  elastic        BOOLEAN NOT NULL DEFAULT false,     -- Elastic 属性
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
-- Card & Deck Management (schema: card, children of players)
-- =============================================================================

CREATE TABLE card.player_cards (
  player_id  UUID NOT NULL, -- 所有プレイヤー (cross-schema reference to account.players; app-level integrity, not enforced by FK)
  card_id    VARCHAR(10) NOT NULL,                   -- カード識別子
  art_no     BIGINT NOT NULL DEFAULT 0,              -- アート番号 (Default: 0)
  count      INT NOT NULL DEFAULT 1,                 -- 所持枚数 (Default: 1)
  PRIMARY KEY (player_id, card_id, art_no)
);

CREATE TABLE card.decks (
  player_id   UUID NOT NULL, -- 所有プレイヤー (cross-schema reference to account.players; app-level integrity, not enforced by FK)
  deck_id     BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY, -- デッキID（自動採番）
  deck_name   VARCHAR(50) NOT NULL,                  -- デッキ名
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
  art_no     BIGINT NOT NULL DEFAULT 0,              -- アート番号 (Default: 0)
  count      INT NOT NULL DEFAULT 1,                 -- 枚数 (Default: 1)
  PRIMARY KEY (player_id, deck_id, card_id, art_no),
  FOREIGN KEY (player_id, deck_id) REFERENCES card.decks(player_id, deck_id) ON DELETE CASCADE
);

-- =============================================================================
-- card.processed_events (Pub/Sub subscriber idempotency)
-- =============================================================================

CREATE TABLE card.processed_events (
  event_id     UUID PRIMARY KEY,                     -- Pub/Sub EventID (publisher 生成の UUIDv4)
  event_type   TEXT NOT NULL,                        -- イベント種別 (faction_purchased / player_onboarded) - ADR-022
  processed_at TIMESTAMPTZ NOT NULL DEFAULT now()    -- 処理日時
);
