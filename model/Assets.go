package model

// Asset ... model definition for supported rypto assets on the system
type Asset struct {
	BaseModel
	Name          string         `json:"name"`
	Symbol        string         `gorm:"unique_index;not null" json:"symbol"`
	TokenType     string         `json:"tokenType"`
	Decimal       int            `json:"decimal"`
	IsEnabled     bool           `gorm:"default:1;index:isEnabled" json:"isEnabled"`
	Transactions  []Transaction  `json:"transactions,omitempty"`
	BatchRequests []BatchRequest `json:"batchRequests,omitempty"`
	UserAddresses []UserAddress  `json:"userAddresses,omitempty"`
	UserBalances  []UserBalance  `gorm:"foreignkey:asset_id" json:"userBalances,omitempty"`
}
