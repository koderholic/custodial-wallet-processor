package dto

// AssetDenominations Response body to get asset denomination request.
type AssetDenominations struct {
	Denominations []AssetDenomination `json:"denominations"`
}

type AdditionalNetwork struct {
	NativeAsset                string         `json:"nativeAsset,omitempty"`
	CoinType            int64          `json:"coinType,omitempty"`
	RequiresMemo        bool           `json:"requiresMemo,omitempty"`
	NativeDecimal             int            `json:"nativeDecimal,omitempty"`
	ChainDenomId  	string `json:"chainDenomId,omitempty"`
	Network 	string `json:"network,omitempty"`
	DepositActivity     string         `json:"depositActivity"`
	WithdrawActivity    string         `json:"withdrawActivity"`
}

type AssetDenomination struct {
	TradeActivity    string `json:"tradeActivity"`
	DepositActivity  string `json:"depositActivity"`
	WithdrawActivity string `json:"withdrawActivity"`
	TransferActivity string `json:"transferActivity"`
	Name             string `json:"name"`
	Symbol           string `json:"symbol"`
	NativeDecimals   int    `json:"nativeDecimals"`
	CoinType         int64  `json:"coinType"`
	TokenType        string `json:"tokenType"`
	RequiresMemo     bool   `json:"requiresMemo"`
	Enabled          bool   `json:"enabled"`
	Network           string `json:"network"`
	AdditionalNetworks []AdditionalNetwork
}

type TWDenomination struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	CoinId int64  `json:"coinId"`
}
