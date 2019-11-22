package model

type Asset struct {
	BaseModel
	Name          string         `gorm:"index" json:"name"`
	Symbol        string         `gorm:"unique_index;not null" json:"symbol"`
	TokenType     string         `json:"tokenType"`
	Decimal       int            `json:"decimal"`
	IsEnabled     bool           `gorm:"default:1" json:"isEnabled"`
	Transactions  []Transaction  `json:"transactions,omitempty"`
	BatchRequests []BatchRequest `json:"batchRequests,omitempty"`
	UserAddresses []UserAddress  `json:"userAddresses,omitempty"`
	UserBalances  []UserBalance  `json:"userBalances,omitempty"`
}
