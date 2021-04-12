package model

// Denomination ... DTO definition for supported crypto assets on the system
type Denomination struct {
	BaseModel
	Name                string         `json:"name,omitempty"`
	AssetSymbol         string         `gorm:"unique_index;not null" json:"asset_symbol,omitempty"`
	CoinType            int64          `json:"coin_type,omitempty"`
	RequiresMemo        bool           `gorm:"requires_memo" json:"requiresMemo,omitempty"`
	Decimal             int            `json:"decimal,omitempty"`
	IsEnabled           bool           `gorm:"default:1;index:isEnabled" json:"is_enabled,omitempty"`
	Transactions        []Transaction  `json:"transactions,omitempty"`
	BatchRequests       []BatchRequest `json:"batch_requests,omitempty"`
	UserAddresses       []UserAddress  `json:"user_addresses,omitempty"`
	UserAssets          []UserAsset    `gorm:"foreignkey:asset_id" json:"user_balances,omitempty"`
	IsToken             *bool          `gorm:"default:0" json:"is_token"`
	IsBatchable         *bool          `gorm:"default:0" json:"is_batchable"`
	IsMultiAddresses    *bool          `gorm:"default:0" json:"is_multi_addresses"`
	MinimumSweepable    float64        `json:"minimum_sweepable"`
	MainCoinAssetSymbol string         `json:"main_coin_asset_symbol"`
	AddressProvider string `gorm:"VARCHAR(150) NOT NULL Default='Bundle';" json:"address_provider"`
	SweepFee            int64          `json:"sweep_fee"`
	TradeActivity       string         `json:"tradeActivity"`
	DepositActivity     string         `json:"depositActivity"`
	WithdrawActivity    string         `json:"withdrawActivity"`
	TransferActivity    string         `json:"transferActivity"`
}
