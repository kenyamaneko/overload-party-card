package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrUnknownSelectionType は selection JSONB の type discriminator が
// 既知の値 (by_factions / by_card_ids) でない場合に返されます。
// 未知 type を握りつぶさず明示エラーで上に伝搬するための sentinel です。
var ErrUnknownSelectionType = errors.New("unknown card_pack selection type")

// CardPack は card.card_pack テーブル 1 行に対応する domain 表現。
// 配布パック (どのカードを何枚配るか) の SSoT。
type CardPack struct {
	PackID        string
	Description   string
	Selection     Selection
	CopiesPerCard int
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Selection は配布対象カードの指定方式を表す sum type。
// domain パッケージ内の private marker method で外部実装をブロックし、
// type switch の網羅性を担保する。
type Selection interface {
	isSelection()
}

// SelectionByFactions は指定 faction(s) に属する全 active card を配布対象とする。
type SelectionByFactions struct {
	Factions []string
}

func (SelectionByFactions) isSelection() {}

// SelectionByCardIDs は指定 card_id のみを配布対象とする (限定カード等)。
type SelectionByCardIDs struct {
	CardIDs []string
}

func (SelectionByCardIDs) isSelection() {}

// SelectionTypeByFactions は selection JSONB の type discriminator 値です。
const SelectionTypeByFactions = "by_factions"

// SelectionTypeByCardIDs は selection JSONB の type discriminator 値です。
const SelectionTypeByCardIDs = "by_card_ids"

// ParseSelection は selection JSONB を Selection 実装にパースします。
// 未知の type は ErrUnknownSelectionType を返し、握りつぶさず呼び出し側に伝搬します。
func ParseSelection(raw json.RawMessage) (Selection, error) {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, fmt.Errorf("parse selection head: %w", err)
	}
	switch head.Type {
	case SelectionTypeByFactions:
		var body struct {
			Factions []string `json:"factions"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, fmt.Errorf("parse selection by_factions: %w", err)
		}
		if len(body.Factions) == 0 {
			return nil, fmt.Errorf("selection by_factions: empty factions list")
		}
		return SelectionByFactions{Factions: body.Factions}, nil
	case SelectionTypeByCardIDs:
		var body struct {
			CardIDs []string `json:"card_ids"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, fmt.Errorf("parse selection by_card_ids: %w", err)
		}
		if len(body.CardIDs) == 0 {
			return nil, fmt.Errorf("selection by_card_ids: empty card_ids list")
		}
		return SelectionByCardIDs{CardIDs: body.CardIDs}, nil
	}
	return nil, fmt.Errorf("%w: %q", ErrUnknownSelectionType, head.Type)
}
