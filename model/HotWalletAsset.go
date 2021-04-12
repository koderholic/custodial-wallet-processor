package model

import "time"

type HotWalletAsset struct {
	BaseModel
	Address                 string     `gorm:"VARCHAR(100);not null" json:"address"`
	AssetSymbol             string     `gorm:"VARCHAR(30);not null" json:"asset_symbol"`
	Balance                 int64      `json:"balance"`
	IsDisabled              bool       `gorm:"default:0" json:"is_disabled"`
	ReservedBalance         int64      `json:"reserved_balance"`
	LastDepositCreatedAt    *time.Time `json:"last_deposit_created_at"`
	LastWithdrawalCreatedAt *time.Time `json:"last_withdrawal_created_at"`
}
