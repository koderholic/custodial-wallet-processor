package model

// Denomination ... DTO definition for supported crypto assets on the system
type Denomination struct {
	BaseModel
	Name                string         `json:"name,omitempty"`
	DefaultNetwork 	string `json:"defaultNetwork,omitempty"`
	AssetSymbol         string         `gorm:"unique_index;not null" json:"asset_symbol,omitempty"`
	TradeActivity       string          `json:"tradeActivity"`
	TransferActivity    string          `json:"transferActivity"`
	Networks []Network `json:"networks,omitempty"`
}
