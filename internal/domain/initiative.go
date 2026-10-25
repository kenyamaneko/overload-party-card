package domain

const (
	// InitiativeKindRoutine はルーチン (1 ターン 1 回使用できる施策) の区分です。
	InitiativeKindRoutine = "routine"
	// InitiativeKindSpecial はスペシャル (1 ゲーム 1 回使用できる施策) の区分です。
	InitiativeKindSpecial = "special"
)

// Initiative はプロダクトに属する施策です。
type Initiative struct {
	InitiativeID string
	ProductID    string
	Kind         string
	Name         string
	InsightCost  int64
	EffectText   string
	Effect       map[string]any
	IsActive     bool
}
