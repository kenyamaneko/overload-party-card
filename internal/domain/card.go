package domain

import "time"

// Card はカード定義を表します。
type Card struct {
	CardID        string
	CardName      string
	ResourceLabel string
	Faction       string
	CardType      string
	Subtype       *string
	IsResizable   bool
	IsElastic     bool
	Stats         CardStats
	EffectText    *string
	Effects       []map[string]any
	Restriction   string
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CardStats はリソースカードのステータスです。
// コンピュート系は Throughput、データ系は Yield を持ちます。リソースでないカードは全項目が未設定です。
type CardStats struct {
	Availability    *int64
	MaintenanceCost *int64
	SLAPenalty      *int64
	Throughput      *int64
	ThroughputMax   *int64
	Yield           *int64
	YieldMax        *int64
}

// CardWithOwnership はカード定義と、プレイヤーがそのカードを所持しているかの組です。
type CardWithOwnership struct {
	Card    *Card
	IsOwned bool
}

// GrantedCardArtNo はカードパックの配布で所持カードに付与するアート番号です。
const GrantedCardArtNo int64 = 0

// PlayerCard はプレイヤーが所持するカード 1 種類 (アート番号ごと) とその枚数を表します。
type PlayerCard struct {
	PlayerID string
	CardID   string
	ArtNo    int64
	Count    int
}

// PlayerCardWithDefinition は所持カードとそのカード定義の組です。
type PlayerCardWithDefinition struct {
	PlayerCard PlayerCard
	Card       *Card
}
