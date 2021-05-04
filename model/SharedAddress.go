package model

import uuid "github.com/satori/go.uuid"

type SharedAddress struct {
	BaseModel
	UserId      uuid.UUID `gorm:"VARCHAR(36);not null" json:"userId"`
	Address     string    `gorm:"VARCHAR(100);not null" json:"address"`
	AssetSymbol string    `gorm:"VARCHAR(30);not null" json:"assetSymbol"`
	Network string    `gorm:"VARCHAR(30);not null" json:"network"`
	CoinType    int64     `gorm:"bigint" json:"coinType"`
}
