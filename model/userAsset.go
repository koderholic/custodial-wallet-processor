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
	DefaultNetwork         string     `gorm:"-" json:"defaultNetwork,omitempty"`
}

func (userAsset *UserAsset) AfterFind() {
	balance, _ := strconv.ParseFloat(userAsset.AvailableBalance, 64)
	userAsset.AvailableBalance = strconv.FormatFloat(balance, 'g', utility.DigPrecision, 64)
}
