package domain

// Product は陣営に紐づくプロダクトです。
type Product struct {
	ProductID   string
	Faction     string
	ProductName string
	IsActive    bool
}

// ProductWithInitiatives はプロダクトと、そのプロダクトに属する施策の組です。
type ProductWithInitiatives struct {
	Product     Product
	Initiatives []Initiative
}
