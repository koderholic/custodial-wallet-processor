package dto

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// FloatBalance ... DTO definition for float crypto record and their respective balances
type FloatBalance struct {
	BaseDTO
	DenominationID   uuid.UUID  `gorm:"type:VARCHAR(36);not null;index:denomination_id" json:"asset_id"`
	AvailableBalance string     `gorm:"type:decimal(64,18);" json:"available_balance"`
	ReservedBalance  string     `gorm:"type:decimal(64,18);" json:"reserved_balance"`
	DeletedAt        *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

// FloatAssetBalance ... Fetch float balance with corresponding asset details
type FloatAssetBalance struct {
	BaseDTO
	DenominationID   uuid.UUID  `gorm:"type:VARCHAR(36);not null;index:denomination_id" json:"asset_id"`
	AvailableBalance string     `gorm:"type:decimal(64,18);" json:"available_balance"`
	ReservedBalance  string     `gorm:"type:decimal(64,18);" json:"reserved_balance"`
	DeletedAt        *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	Denomination
}

// TableName ... Change table name from the default generated to float_balances
func (dto FloatAssetBalance) TableName() string {
	return "float_balances"
}
