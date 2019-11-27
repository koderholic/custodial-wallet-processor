package model

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// UserBalance ... Model definition for user crypto record and their respective balances
type UserBalance struct {
	BaseModel
	UserID           uuid.UUID  `gorm:"type:VARCHAR(36);not null;index:user_id" json:"userId"`
	AssetID          uuid.UUID  `gorm:"type:VARCHAR(36);not null;index:asset_id" json:"-"`
	Symbol           string     `sql:"-" json:"symbol"`
	AvailableBalance int64      `gorm:"type:BIGINT" json:"availableBalance"`
	ReservedBalance  int64      `gorm:"type:BIGINT" json:"reservedBalance"`
	DeletedAt        *time.Time `gorm:"index" json:"deletedAt,omitempty"`
}

// UserAssetBalance ... Fetch  user balance with corresponding asset details
type UserAssetBalance struct {
	BaseModel
	UserID           uuid.UUID  `gorm:"type:VARCHAR(36);not null;index:user_id" json:"userId"`
	AssetID          uuid.UUID  `gorm:"type:VARCHAR(36);not null;index:asset_id" json:"-"`
	AvailableBalance int64      `gorm:"type:BIGINT" json:"availableBalance"`
	ReservedBalance  int64      `gorm:"type:BIGINT" json:"reservedBalance"`
	DeletedAt        *time.Time `gorm:"index" json:"deletedAt,omitempty"`
	Asset
}

// TableName ... Change table name from default generated from UserAssetBalance
func (model UserAssetBalance) TableName() string {
	return "user_balances"
}
