package model

type Asset struct {
	BaseModel
	Name          string         `gorm:"index" json:"name"`
	Symbol        string         `gorm:"unique_index;not null" json:"symbol"`
	TokenType     string         `json:"tokenType"`
	Decimal       int            `json:"decimal"`
	IsEnabled     bool           `gorm:"default:1" json:"isEnabled"`
	Transactions  []Transaction  `json:"transactions"`
	BatchRequests []BatchRequest `json:"batchRequests"`
	UserAddresses []UserAddress  `json:"userAddresses"`
	UserBalances  []UserBalance  `json:"userBalances"`
}
