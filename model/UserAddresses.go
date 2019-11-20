package model

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type UserAddress struct {
	BaseModel
	UserId   uuid.UUID `gorm:"type:VARCHAR(36);not null;" json:"initiatorId"`
	AssetId  uuid.UUID `gorm:"type:VARCHAR(36);not null;" json:"assetId"`
	Address  string    `gorm:"not null" json:"address"`
	KeyId    string    `gorm:"not null" json:"keyId"`
	Validity time.Time `json:"validity"`
	IsValid  bool      `gorm:"default:1" json:"isValid"`
}
