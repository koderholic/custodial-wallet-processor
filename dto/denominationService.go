package dto

// AssetDenominations Response body to get asset denomination request.
type AssetDenominations struct {
	Denominations []AssetDenomination `json:"denominations"`
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
}

type TWDenomination struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	CoinId int64  `json:"coinId"`
}
