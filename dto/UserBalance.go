package dto

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// UserBalance ... DTO definition for user crypto record and their respective balances
type UserBalance struct {
	BaseDTO
	UserID           uuid.UUID  `gorm:"type:VARCHAR(36);not null;index:user_id" json:"user_id"`
	AssetID          uuid.UUID  `gorm:"type:VARCHAR(36);not null;index:asset_id" json:"-"`
	Symbol           string     `sql:"-" json:"symbol"`
	AvailableBalance float64    `gorm:"type:BIGINT;not null" json:"available_balance"`
	ReservedBalance  float64    `gorm:"type:BIGINT;not null" json:"reserved_balance"`
	DeletedAt        *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// UserAssetBalance ... Fetch  user balance with corresponding asset details
type UserAssetBalance struct {
	BaseDTO
	UserID           uuid.UUID  `gorm:"type:VARCHAR(36);not null;index:user_id" json:"user_id"`
	AssetID          uuid.UUID  `gorm:"type:VARCHAR(36);not null;index:asset_id" json:"-"`
	AvailableBalance float64    `gorm:"type:BIGINT;not null" json:"available_balance"`
	ReservedBalance  float64    `gorm:"type:BIGINT;not null" json:"reserved_balance"`
	DeletedAt        *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	Asset
}

// TableName ... Change table name from default generated from UserAssetBalance
func (dto UserAssetBalance) TableName() string {
	return "user_balances"
}
