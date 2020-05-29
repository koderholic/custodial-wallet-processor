package model

import (
	uuid "github.com/satori/go.uuid"
)

// UserAddress ... DTO definitions for all user crypto addresses for fund deposit
type UserAddress struct {
	BaseModel
	AssetID     uuid.UUID `gorm:"type:VARCHAR(36);not null" json:"asset_id"`
	Address     string    `gorm:"VARCHAR(100);" json:"address"`
	AddressType string    `gorm:"VARCHAR(50) NOT NULL DEFAULT 'Segwit';" json:"addressType"`
	V2Address   string    `gorm:"VARCHAR(255);" json:"v2Address"`
	Memo        string    `gorm:"VARCHAR(15);" json:"memo"`
	IsValid     bool      `gorm:"default:1" json:"is_valid"`
}
