package dto

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// UserAddress ... DTO definitions for all user crypto addresses for fund deposit
type UserAddress struct {
	BaseDTO
	UserID   uuid.UUID `gorm:"type:VARCHAR(36);not null;index:user_id" json:"initiatorId"`
	AssetID  uuid.UUID `gorm:"type:VARCHAR(36);not null;index:asset_id" json:"assetId"`
	Address  string    `gorm:"not null" json:"address"`
	KeyID    string    `gorm:"not null" json:"keyId"`
	Validity time.Time `json:"validity"`
	IsValid  bool      `gorm:"default:1" json:"isValid"`
}
