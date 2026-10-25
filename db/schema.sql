-- overload-party-card - PostgreSQL DDL (service-owned)
--
-- 本ファイルは card サービスの SSoT。psqldef 互換。
-- 他サービスのデータは所有サービスの API 経由でだけ参照させ、DB を介した結合を作らないため、
-- 他スキーマへの参照（例: player_id が指す account.players）には FK を張らずアプリ層で整合性を担保する。

CREATE SCHEMA IF NOT EXISTS card;

CREATE OR REPLACE FUNCTION card.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE card.card_definitions (
  card_id        VARCHAR(10) NOT NULL,               -- カード識別子（例: SH-0001）
  card_name      VARCHAR(100) NOT NULL,              -- カード名
  resource_label VARCHAR(30) NOT NULL,                -- リソースラベル
  faction        VARCHAR(20) NOT NULL CHECK (faction IN ('SHE', 'Tenki', 'Sugar', 'Tuners', 'Neutral')), -- 陣営（SHE / Tenki / Sugar / Tuners / Neutral）
  card_type      VARCHAR(30) NOT NULL,               -- カードタイプ（Compute / DataResource / Platform / Attachment / Strategy / Reactive / Incident）
  subtype        VARCHAR(30),                        -- サブタイプ（Compute / DataResource のカードにのみ設定: VM / Container / Database 等）
  resizable      BOOLEAN NOT NULL,                   -- リサイザブル属性
  elastic        BOOLEAN NOT NULL,                   -- エラスティック属性
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

-- 陣営に属するプロダクトと、その施策 (ルーチン / スペシャル) の SSoT。
-- 陣営:プロダクト = 1:N（陣営は products.faction 列で表現）、
-- プロダクト:施策 = 1:N（initiatives.product_id で親を参照）。
-- decks は product_id / routine_id / special_id を論理参照として持ち、整合性は
-- アプリ層で担保する（card_id と同様、FK は張らない）。

CREATE TABLE card.products (
  product_id   VARCHAR(10)  NOT NULL,                -- プロダクト識別子（例: PD-0001）
  faction      VARCHAR(20)  NOT NULL,                -- 所属陣営
  product_name VARCHAR(100) NOT NULL,                -- プロダクト名
  is_active    BOOLEAN      NOT NULL DEFAULT true,   -- 有効フラグ（論理削除）
  created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),  -- 作成日時
  updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),  -- 更新日時
  CONSTRAINT products_faction_check CHECK (faction IN ('SHE', 'Tenki', 'Sugar', 'Tuners')),
  PRIMARY KEY (product_id)
);

CREATE INDEX idx_products_faction ON card.products(faction);
CREATE TRIGGER trg_products_updated_at BEFORE UPDATE ON card.products FOR EACH ROW EXECUTE FUNCTION card.update_updated_at();

CREATE TABLE card.initiatives (
  initiative_id VARCHAR(10)  NOT NULL,                -- 施策識別子（例: IN-0001）
  product_id    VARCHAR(10)  NOT NULL,                -- 親プロダクト識別子
  kind          VARCHAR(10)  NOT NULL,                -- 区分（routine: 1ターン1回 / special: 1ゲーム1回）
  name          VARCHAR(100) NOT NULL,                -- 施策名
  insight_cost  BIGINT       NOT NULL,                -- 発動インサイトコスト
  effect_text   VARCHAR(500) NOT NULL,                -- 効果テキスト（表示用）
  effect        JSONB        NOT NULL,                -- 効果定義（DSL）
  is_active     BOOLEAN      NOT NULL DEFAULT true,   -- 有効フラグ（論理削除）
  created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),  -- 作成日時
  updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),  -- 更新日時
  CONSTRAINT initiatives_kind_check CHECK (kind IN ('routine', 'special')),
  CONSTRAINT initiatives_insight_cost_check CHECK (insight_cost >= 0),
  PRIMARY KEY (initiative_id),
  CONSTRAINT initiatives_product_id_fkey FOREIGN KEY (product_id) REFERENCES card.products(product_id) ON DELETE CASCADE
);

CREATE INDEX idx_initiatives_product ON card.initiatives(product_id);
CREATE TRIGGER trg_initiatives_updated_at BEFORE UPDATE ON card.initiatives FOR EACH ROW EXECUTE FUNCTION card.update_updated_at();

CREATE TABLE card.player_cards (
  player_id  UUID NOT NULL, -- 所有プレイヤー (account.players への参照。FK は張らずアプリ層で整合性を担保する)
  card_id    VARCHAR(10) NOT NULL,                   -- カード識別子
  art_no     BIGINT NOT NULL,                        -- アート番号
  count      INT NOT NULL,                           -- 所持枚数
  PRIMARY KEY (player_id, card_id, art_no)
);

CREATE TABLE card.decks (
  player_id   UUID NOT NULL, -- 所有プレイヤー (account.players への参照。FK は張らずアプリ層で整合性を担保する)
  deck_id     BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY, -- デッキID（自動採番）
  deck_name   VARCHAR(50) NOT NULL,                  -- デッキ名
  faction     VARCHAR(20) NOT NULL,                  -- 宣言陣営（SHE / Tenki / Sugar / Tuners）
  product_id  VARCHAR(10) NOT NULL,                  -- 選択したプロダクトの ID（宣言陣営に属する）
  routine_id  VARCHAR(10) NOT NULL,                  -- セットしたルーチン施策の ID（選択プロダクトに属する）
  special_id  VARCHAR(10) NOT NULL,                  -- セットしたスペシャル施策の ID（選択プロダクトに属する）
  playmat_no  BIGINT,                                -- プレイマット番号
  sleeve_no   BIGINT,                                -- スリーブ番号
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
  CONSTRAINT deck_cards_player_id_deck_id_fkey FOREIGN KEY (player_id, deck_id) REFERENCES card.decks(player_id, deck_id) ON DELETE CASCADE
);

-- 配布パック (どのカードを何枚配るか) の SSoT。shop からは
-- shop.product_card_pack.card_pack_id で論理参照される (FK なし)。
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
  CONSTRAINT card_pack_cards_pack_id_fkey FOREIGN KEY (pack_id) REFERENCES card.card_pack(pack_id) ON DELETE CASCADE
);

-- 処理済みの Pub/Sub イベントの記録。再配信された同じイベントを二重に処理しないために使う。
CREATE TABLE card.processed_events (
  event_id     UUID PRIMARY KEY,                     -- Pub/Sub のイベント ID (発行側が生成する UUIDv4)
  event_type   TEXT NOT NULL,                        -- イベント種別 (card_pack_purchased / player_onboarded)
  processed_at TIMESTAMPTZ NOT NULL DEFAULT now()    -- 処理日時
);
