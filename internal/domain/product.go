package domain

// Product は陣営に紐づくプロダクト定義。
type Product struct {
	ProductID   string `json:"product_id" db:"product_id"`
	Faction     string `json:"faction" db:"faction"`
	ProductName string `json:"product_name" db:"product_name"`
	IsActive    bool   `json:"is_active" db:"is_active"`
}
