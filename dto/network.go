package dto

import (
	uuid "github.com/satori/go.uuid"
)


type NetworkAsset struct {
	NativeAsset                string         `json:"nativeAsset,omitempty"`
	AssetSymbol         string         `json:"assetSymbol,omitempty"`
	CoinType            int64          `json:"coinType,omitempty"`
	RequiresMemo        bool           `json:"requiresMemo,omitempty"`
	NativeDecimals             int            `json:"nativeDecimal,omitempty"`
	ChainDenomId  	string `json:"chainDenomId,omitempty"`
	Network 	string `json:"network,omitempty"`
	AddressProvider string `json:"addressProvider"`
	IsBatchable         *bool          `gorm:"default:0" json:"is_batchable"`
	IsMultiAddresses    *bool          `gorm:"default:0" json:"is_multi_addresses"`
	IsToken             *bool          `gorm:"is_token default:0" json:"is_token"`
	MinimumSweepable    float64         `json:"minimum_sweepable"`
	SweepFee            int64           `json:"sweep_fee"`
	DepositActivity     string         `json:"depositActivity"`
	WithdrawActivity    string         `json:"withdrawActivity"`
	AssetID           uuid.UUID `json:"asset_id"`
	UserID           uuid.UUID `json:"user_id"`
	DenominationID   uuid.UUID `json:"-"`
	DefaultNetwork         string     `json:"defaultNetwork,omitempty"`
}