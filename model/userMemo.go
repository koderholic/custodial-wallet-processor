package model

import (
	uuid "github.com/satori/go.uuid"
)

// UserMemo ... User unique memo
type UserMemo struct {
	UserID uuid.UUID `gorm:"type:VARCHAR(36);not null" json:"user_id"`
	Memo   string    `gorm:"type:VARCHAR(100);not null" json:"memo,omitempty"`
	IsPrimaryAddress bool `json:"is_primary_address"`
}
