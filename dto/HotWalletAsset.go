package dto

import "time"

type HotWalletAsset struct {
	BaseDTO
	Address                 string    `gorm:"VARCHAR(100);not null" json:"address"`
	AssetSymbol             string    `gorm:"VARCHAR(30);not null" json:"asset_symbol"`
	Balance                 int64     `json:"balance"`
	IsDisabled              bool      `gorm:"default:1" json:"is_disabled"`
	ReservedBalance         int64     `json:"balance"`
	LastDepositCreatedAt    time.Time `json:"lastDepositCreatedAt"`
	LastWithdrawalCreatedAt time.Time `json:"lastWithdrawalCreatedAt"`
}
