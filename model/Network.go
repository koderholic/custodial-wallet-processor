package model

type Network struct {
	BaseModel
	NativeAsset                string         `json:"nativeAsset,omitempty"`
	AssetSymbol         string         `json:"assetSymbol,omitempty"`
	CoinType            int64          `json:"coinType,omitempty"`
	RequiresMemo        bool           `json:"requiresMemo,omitempty"`
	NativeDecimals             int            `json:"nativeDecimals,omitempty"`
	ChainDenomId  	string `json:"chainDenomId,omitempty"`
	Network 	string `json:"network,omitempty"`
	AddressProvider string `json:"addressProvider"`
	IsBatchable         *bool          `gorm:"default:0" json:"isBatchable"`
	IsMultiAddresses    *bool          `gorm:"default:0" json:"isMultiAddresses"`
	IsToken             *bool          `gorm:"is_token default:0" json:"isToken"`
	MinimumSweepable    float64         `json:"minimumSweepable"`
	SweepFee            int64           `json:"sweepFee"`
	DepositActivity     string         `json:"depositActivity"`
	WithdrawActivity    string         `json:"withdrawActivity"`
}
