package dto

import (
	uuid "github.com/satori/go.uuid"
)

// UserAddress ... DTO definitions for all user crypto addresses for fund deposit
type UserAddress struct {
	BaseDTO
	AssetID uuid.UUID `gorm:"type:VARCHAR(36);not null;index:asset_id" json:"asset_id"`
	Address string    `gorm:"VARCHAR(100);not null" json:"address"`
	IsValid bool      `gorm:"default:1" json:"is_valid"`
}
