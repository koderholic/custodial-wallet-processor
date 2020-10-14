package dto

//BatchBTCRequest object for BTC batch transaction
type BatchRequest struct {
	AssetSymbol   string            `json:"assetSymbol"`
	ChangeAddress string            `json:"changeAddress"`
	IsSweep       bool              `json:"isSweep"`
	Origins       []string          `json:"origins"`
	Recipients    []BatchRecipients `json:"recipients"`
	ProcessType   string            `json:"processType"`
	Reference     string            `json:"reference"`
}

// BatchRecipients object for BTC batch transactions
type BatchRecipients struct {
	Address string `json:"address"`
	Value   int64  `json:"value"`
}
