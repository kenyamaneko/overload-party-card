package domain

// Product は陣営に紐づくプロダクト定義。
type Product struct {
	ProductID   string       `json:"product_id"`
	Faction     string       `json:"faction"`
	ProductName string       `json:"product_name"`
	Initiatives []Initiative `json:"initiatives"`
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
