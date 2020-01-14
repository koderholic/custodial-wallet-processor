package dto

// Denomination ... DTO definition for supported crypto assets on the system
type Denomination struct {
	BaseDTO
	Name          string         `json:"name,omitempty"`
	Symbol        string         `gorm:"unique_index;not null" json:"symbol,omitempty"`
	TokenType     string         `json:"token_type,omitempty"`
	Decimal       int            `json:"decimal,omitempty"`
	IsEnabled     bool           `gorm:"default:1;index:isEnabled" json:"is_enabled,omitempty"`
	Transactions  []Transaction  `json:"transactions,omitempty"`
	BatchRequests []BatchRequest `json:"batch_requests,omitempty"`
	UserAddresses []UserAddress  `json:"user_addresses,omitempty"`
	UserBalances  []UserBalance  `gorm:"foreignkey:asset_id" json:"user_balances,omitempty"`
}
