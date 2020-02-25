package dto

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// UserAssetBalance ... Fetch  user balance with corresponding asset details
type UserAsset struct {
	BaseDTO
	UserID           uuid.UUID  `gorm:"type:VARCHAR(36);not null;index:user_id" json:"user_id"`
	DenominationID   uuid.UUID  `gorm:"type:VARCHAR(36);not null;index:denomination_id" json:"-"`
	AvailableBalance string     `gorm:"type:decimal(64,18) CHECK(available_balance >= 0);not null;" json:"available_balance"`
	DeletedAt        *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	Denomination
}
