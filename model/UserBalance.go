package model

import uuid "github.com/satori/go.uuid"

type UserBalance struct {
	BaseModel
	UserId        uuid.UUID `gorm:"type:VARCHAR(36);not null;" json:"userId"`
	AssetId       uuid.UUID `gorm:"type:VARCHAR(36);not null;" json:"assetId"`
	LedgerBalance string    `json:"ledgerBalance"`
	BookBalance   string    `json:"bookBalance"`
}
