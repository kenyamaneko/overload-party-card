package domain

// PlayerOnboardedEvent はプレイヤーのオンボーディング完了を表すイベントです。
type PlayerOnboardedEvent struct {
	EventID          string
	EventType        string
	PlayerID         string
	InitialFactionID string
}

// CardPackPurchasedEvent はプレイヤーによるカードパックの購入を表すイベントです。
type CardPackPurchasedEvent struct {
	EventID    string
	EventType  string
	PlayerID   string
	CardPackID string
}
