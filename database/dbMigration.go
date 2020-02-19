package database

import (
	"wallet-adapter/dto"
	// "wallet-adapter/utility"

	// "github.com/trustwallet/blockatlas/pkg/logger"
)

// RunDbMigrations ... This creates corresponding tables for dtos on the db and watches the dto for field additions
func (database *Database) RunDbMigrations() {
	database.DB.AutoMigrate(&dto.Denomination{}, &dto.BatchRequest{}, &dto.ChainTransaction{}, &dto.Transaction{}, &dto.UserAddress{}, &dto.UserBalance{}, &dto.HotWalletAsset{}, &dto.TransactionQueue{})
	database.DB.Model(&dto.UserBalance{}).AddForeignKey("denomination_id", "denominations(id)", "CASCADE", "CASCADE")
	database.DB.Model(&dto.UserBalance{}).AddForeignKey("denomination_id", "denominations(id)", "CASCADE", "CASCADE")


	// database.CreateTables()
}

// DBSeeder .. This seeds supported assets into the database
func (database *Database) DBSeeder() {

	assets := []dto.Denomination{
		dto.Denomination{Name: "Binance Coin", Symbol: "BNB", TokenType: "BNB", Decimal: 8},
		dto.Denomination{Name: "Ethereum Coin", Symbol: "ETH", TokenType: "ETH", Decimal: 18},
		dto.Denomination{Name: "Bitcoin", Symbol: "BTC", TokenType: "BTC", Decimal: 8},
	}

	for _, asset := range assets {
		if err := database.DB.FirstOrCreate(&asset, dto.Denomination{Symbol: asset.Symbol}).Error; err != nil {
			database.Logger.Error("Error with creating asset record %s : %s", asset.Symbol, err)
		}
	}
	database.Logger.Info("Supported assets seeded successfully")
}
