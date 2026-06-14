package domain

import "encoding/json"

// Initiative はプロダクトに属する施策 1 件分。
type Initiative struct {
	InitiativeID string          `json:"initiative_id"`
	ProductID    string          `json:"product_id"`
	Kind         string          `json:"kind"`
	Name         string          `json:"name"`
	InsightCost  int64           `json:"insight_cost"`
	EffectText   string          `json:"effect_text"`
	Effect       json.RawMessage `json:"effect"`
	IsActive     bool            `json:"is_active"`
}
