package model

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// UserAddress ... Model definitions for all user crypto addresses for fund deposit
type UserAddress struct {
	BaseModel
	UserID   uuid.UUID `gorm:"type:VARCHAR(36);not null;" json:"initiatorId"`
	AssetID  uuid.UUID `gorm:"type:VARCHAR(36);not null;" json:"assetId"`
	Address  string    `gorm:"not null" json:"address"`
	KeyID    string    `gorm:"not null" json:"keyId"`
	Validity time.Time `json:"validity"`
	IsValid  bool      `gorm:"default:1" json:"isValid"`
}
