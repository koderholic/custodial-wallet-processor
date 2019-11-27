package model

// Asset ... model definition for supported rypto assets on the system
type Asset struct {
	BaseModel
	Name          string         `json:"name,omitempty"`
	Symbol        string         `gorm:"unique_index;not null" json:"symbol,omitempty"`
	TokenType     string         `json:"tokenType,omitempty"`
	Decimal       int            `json:"decimal,omitempty"`
	IsEnabled     bool           `gorm:"default:1;index:isEnabled" json:"isEnabled,omitempty"`
	Transactions  []Transaction  `json:"transactions,omitempty"`
	BatchRequests []BatchRequest `json:"batchRequests,omitempty"`
	UserAddresses []UserAddress  `json:"userAddresses,omitempty"`
	UserBalances  []UserBalance  `gorm:"foreignkey:asset_id" json:"userBalances,omitempty"`
}
