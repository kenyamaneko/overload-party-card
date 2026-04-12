package model

// PlayerCard は所持カードバリアントの DB 行を表す内部型です。
// ArtNo/Count はストレージの関心事であり、API レスポンスでは
// PlayerCardWithDef でラップするため公開パッケージに含めません。
type PlayerCard struct {
	PlayerID string `json:"player_id" db:"player_id"`
	CardID   string `json:"card_id" db:"card_id"`
	ArtNo    int64  `json:"art_no" db:"art_no"`
	Count    int    `json:"count" db:"count"`
}
