package domain

import (
	"encoding/json"
	"time"
)

// Card は card_definitions テーブル 1 行に対応する domain 表現。
// stats / effects は JSONB をそのまま保持し、API 層で展開する。
type Card struct {
	CardID        string          `json:"card_id" db:"card_id"`
	CardName      string          `json:"card_name" db:"card_name"`
	ResourceLabel string          `json:"resource_label" db:"resource_label"`
	Faction       string          `json:"faction" db:"faction"`
	CardType      string          `json:"card_type" db:"card_type"`
	Subtype       *string         `json:"subtype,omitempty" db:"subtype"`
	Resizable     bool            `json:"resizable" db:"resizable"`
	Elastic       bool            `json:"elastic" db:"elastic"`
	Stats         json.RawMessage `json:"stats" db:"stats"`
	EffectText    *string         `json:"effect_text,omitempty" db:"effect_text"`
	Effects       json.RawMessage `json:"effects,omitempty" db:"effects"`
	Restriction   string          `json:"restriction" db:"restriction"`
	IsActive      bool            `json:"is_active" db:"is_active"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

// PlayerCard は player_cards テーブル 1 行に対応する domain 表現。
type PlayerCard struct {
	PlayerID string `db:"player_id"`
	CardID   string `db:"card_id"`
	ArtNo    int64  `db:"art_no"`
	Count    int    `db:"count"`
}
