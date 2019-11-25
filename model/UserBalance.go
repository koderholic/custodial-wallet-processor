package model

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// UserBalance ... Model definition for user crypto record and their respective balances
type UserBalance struct {
	BaseModel
	UserID           uuid.UUID  `gorm:"type:VARCHAR(36);not null;" json:"userId"`
	AssetID          uuid.UUID  `gorm:"type:VARCHAR(36);not null;" json:"assetId"`
	AvailableBalance int64      `gorm:"type:BIGINT" json:"availableBalance"`
	ReservedBalance  int64      `gorm:"type:BIGINT" json:"reservedBalance"`
	DeletedAt        *time.Time `gorm:"index" json:"deletedAt,omitempty"`
}

// UserAssetBalance ... Fetch  user balance with corresponding asset details
type UserAssetBalance struct {
	BaseModel
	UserID           uuid.UUID  `gorm:"type:VARCHAR(36);not null;" json:"userId"`
	AssetID          uuid.UUID  `gorm:"type:VARCHAR(36);not null;" json:"assetId"`
	AvailableBalance int64      `gorm:"type:BIGINT" json:"availableBalance"`
	ReservedBalance  int64      `gorm:"type:BIGINT" json:"reservedBalance"`
	DeletedAt        *time.Time `gorm:"index" json:"deletedAt,omitempty"`
	Asset
}
