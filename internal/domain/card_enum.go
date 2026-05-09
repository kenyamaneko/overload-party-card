package domain

// PassiveEffectType はカードに付与されるパッシブエフェクトの種別。本パッケージが SSoT。
const (
	PassiveTPPerBackendDB      = "tp_per_backend_db"
	PassiveTPPerBackendData    = "tp_per_backend_data"
	PassiveTPIfCardTypeOnField = "tp_if_card_type_on_field"
	PassiveYieldPerOtherDB     = "dv_per_other_db"
	PassiveYieldIfCardOnField  = "dv_if_card_on_field"
	PassiveAVBonus             = "av_bonus"
	PassiveScaleCostFree       = "scale_cost_free"
)

// PlatformEffectType は Platform カードが場に出ているとき他カードに付与するエフェクトの種別。
// 本パッケージが SSoT。
const (
	PlatformTPBonus    = "tp_bonus"
	PlatformYieldBonus = "yield_bonus"
	PlatformAVBonus    = "av_bonus"
)

// AttachmentEffectType は装着系カードが付与するエフェクトの種別。本パッケージが SSoT。
const (
	AttachmentStatBonus = "stat_bonus"
)
