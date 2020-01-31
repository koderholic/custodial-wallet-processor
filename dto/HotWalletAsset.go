package dto

type HotWalletAsset struct {
	BaseDTO
	Address     string `gorm:"VARCHAR(100);not null" json:"address"`
	AssetSymbol string `gorm:"VARCHAR(30);not null;unique_index" json:"asset_symbol"`
	Balance     int64  `json:"balance"`
	IsDisabled  bool   `gorm:"default:1" json:"is_disabled"`
}
