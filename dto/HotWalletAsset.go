package dto

type HotWalletAsset struct {
	BaseDTO
	Address     string `gorm:"VARCHAR(100);not null" json:"address"`
	AssetSymbol string `gorm:"VARCHAR(100);not null" json:"asset_symbol"`
	IsDisabled  bool   `gorm:"default:1" json:"is_disabled"`
}
