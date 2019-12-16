package dto

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// FloatBalance ... DTO definition for float crypto record and their respective balances
type FloatBalance struct {
	BaseDTO
	AssetID          uuid.UUID  `gorm:"type:VARCHAR(36);not null;index:asset_id" json:"asset_id"`
	AvailableBalance int64      `gorm:"type:BIGINT" json:"available_balance"`
	ReservedBalance  int64      `gorm:"type:BIGINT" json:"reserved_balance"`
	DeletedAt        *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// FloatAssetBalance ... Fetch float balance with corresponding asset details
type FloatAssetBalance struct {
	BaseDTO
	AssetID          uuid.UUID  `gorm:"type:VARCHAR(36);not null;index:asset_id" json:"asset_id"`
	AvailableBalance int64      `gorm:"type:BIGINT" json:"available_balance"`
	ReservedBalance  int64      `gorm:"type:BIGINT" json:"reserved_balance"`
	DeletedAt        *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	Asset
}

// TableName ... Change table name from the default generated to float_balances
func (dto FloatAssetBalance) TableName() string {
	return "float_balances"
}
