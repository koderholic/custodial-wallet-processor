package dto

import "time"

type FloatManager struct {
	BaseDTO
	ResidualAmount        float64
	AssetSymbol           string
	TotalUserBalance      float64
	DepositSum            float64
	WithdrawalSum         float64
	FloatOnChainBalance   float64
	MaximumFloatRange     float64
	MinimumFloatRange     float64
	PercentageUserBalance float64
	Deficit               float64
	Action                string
	LastRunTime           time.Time
}

func (float FloatManager) TableName() string {
	return "float_manager_variables" // default table name
}
