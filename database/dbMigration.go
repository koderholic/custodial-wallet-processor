package database

import (
	"wallet-adapter/dto"
)

// RunDbMigrations ... This creates corresponding tables for dtos on the db and watches the dto for field additions
func (database *Database) RunDbMigrations() {
	database.DB.AutoMigrate(&dto.Asset{}, &dto.BatchRequest{}, &dto.ChainTransaction{}, &dto.Transaction{}, &dto.UserAddress{}, &dto.UserBalance{})
	database.DB.Model(&dto.UserBalance{}).AddForeignKey("asset_id", "assets(id)", "CASCADE", "CASCADE")
}

// DBSeeder .. This seeds supported assets into the database
func (database *Database) DBSeeder() {

	assets := []dto.Asset{
		dto.Asset{Name: "Binance Coin", Symbol: "BNB", TokenType: "BNB", Decimal: 8},
		dto.Asset{Name: "Ethereum Coin", Symbol: "ETH", TokenType: "ETH", Decimal: 18},
		dto.Asset{Name: "Bitcoin", Symbol: "BTC", TokenType: "BTC", Decimal: 8},
	}

	for _, asset := range assets {
		if err := database.DB.FirstOrCreate(&asset, dto.Asset{Symbol: asset.Symbol}).Error; err != nil {
			database.Logger.Error("Error with creating asset record %s : %s", asset.Symbol, err)
		}
	}
	database.Logger.Info("Supported assets seeded successfully")
}
