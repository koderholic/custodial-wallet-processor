package dto

import (
	uuid "github.com/satori/go.uuid"
)

// UserAsset ... Fetch  user balance with corresponding asset details
type UserAsset struct {
	BaseDTO
	UserID           uuid.UUID `gorm:"type:VARCHAR(36);not null" json:"user_id"`
	DenominationID   uuid.UUID `gorm:"type:VARCHAR(36);not null" json:"-"`
	AvailableBalance string    `gorm:"type:decimal(64,18) CHECK(available_balance >= 0);not null;" json:"available_balance"`
	AssetSymbol      string    `gorm:"-" json:"asset_symbol,omitempty"`
	Decimal          int       `gorm:"-" json:"decimal,omitempty"`
}
