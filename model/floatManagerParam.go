package model

// FloatManagerParam...
type FloatManagerParam struct {
	BaseModel
	MinPercentMaxUserBalance       float64
	MaxPercentMaxUserBalance       float64
	MinPercentTotalUserBalance     float64
	AveragePercentTotalUserBalance float64
	MaxPercentTotalUserBalance     float64
	PercentMinimumTriggerLevel     float64
	PercentMaximumTriggerLevel     float64
	AssetSymbol                    string
	Network                    string
}
