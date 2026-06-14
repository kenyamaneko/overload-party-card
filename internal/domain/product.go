package domain

import "encoding/json"

// Product は陣営に紐づくプロダクト定義。
type Product struct {
	ProductID   string       `json:"product_id"`
	Faction     string       `json:"faction"`
	ProductName string       `json:"product_name"`
	Initiatives []Initiative `json:"initiatives"`
}

// Initiative はプロダクトの施策 1 件分。
type Initiative struct {
	InitiativeID string          `json:"initiative_id"`
	Kind         string          `json:"kind"`
	Name         string          `json:"name"`
	InsightCost  int64           `json:"insight_cost"`
	EffectText   string          `json:"effect_text"`
	Effect       json.RawMessage `json:"effect"`
}

// FindInitiative は指定 ID・区分の施策を返す。
func (p *Product) FindInitiative(initiativeID, kind string) (Initiative, bool) {
	for _, i := range p.Initiatives {
		if i.InitiativeID == initiativeID && i.Kind == kind {
			return i, true
		}
	}
	return Initiative{}, false
}
