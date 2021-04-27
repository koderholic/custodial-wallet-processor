package model

import (
	"strconv"
	"wallet-adapter/utility"

	uuid "github.com/satori/go.uuid"
)

// UserAsset ... Fetch  user balance with corresponding asset details
type UserAsset struct {
	BaseModel
	UserID           uuid.UUID `gorm:"type:VARCHAR(36);not null" json:"user_id"`
	DenominationID   uuid.UUID `gorm:"type:VARCHAR(36);not null" json:"-"`
	AvailableBalance string    `gorm:"type:decimal(64,18) CHECK(available_balance >= 0);not null;" json:"available_balance"`
	AssetSymbol      string    `gorm:"-" json:"asset_symbol,omitempty"`
	Decimal          int       `gorm:"-" json:"decimal,omitempty"`
	CoinType         int64     `gorm:"-" json:"coinType,omitempty"`
	RequiresMemo     bool      `gorm:"-" json:"requiresMemo,omitempty"`
	AddressProvider string    `gorm:"-" json:"address_provider,omitempty"`
	MainCoinAssetSymbol string         `json:"main_coin_asset_symbol"`
}

func (userAsset *UserAsset) AfterFind() {
	balance, _ := strconv.ParseFloat(userAsset.AvailableBalance, 64)
	userAsset.AvailableBalance = strconv.FormatFloat(balance, 'g', utility.DigPrecision, 64)
}
