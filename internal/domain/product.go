package domain

// Product は陣営に紐づくプロダクト定義。
type Product struct {
	ProductID   string `json:"product_id"`
	Faction     string `json:"faction"`
	ProductName string `json:"product_name"`
}
