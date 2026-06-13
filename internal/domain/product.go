package domain

import "encoding/json"

// Product は陣営に 1:1 で紐づくプロダクト定義。カードではなくデッキのメタ情報で、
// デッキが宣言した陣営のプロダクトが使用できる施策を規定する。
type Product struct {
	ProductID   string       `json:"product_id"`
	Faction     string       `json:"faction"`
	ProductName string       `json:"product_name"`
	Initiatives []Initiative `json:"initiatives"`
}

// Initiative はプロダクトの施策 1 件分。
// Effect は battle エンジンの効果 DSL をそのまま保持し、card サービスでは解釈しない。
type Initiative struct {
	Kind        string          `json:"kind"`
	Name        string          `json:"name"`
	InsightCost int64           `json:"insight_cost"`
	EffectText  string          `json:"effect_text"`
	Effect      json.RawMessage `json:"effect"`
}
