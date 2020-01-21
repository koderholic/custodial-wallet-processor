package database

import (
	"wallet-adapter/dto"
)

// RunDbMigrations ... This creates corresponding tables for dtos on the db and watches the dto for field additions
func (database *Database) RunDbMigrations() {
	database.DB.AutoMigrate(&dto.Denomination{}, &dto.BatchRequest{}, &dto.ChainTransaction{}, &dto.Transaction{}, &dto.UserAddress{}, &dto.UserBalance{})
	database.DB.Model(&dto.UserBalance{}).AddForeignKey("denomination_id", "denominations(id)", "CASCADE", "CASCADE")

	database.DB.DropTableIfExists("float_balances")
	database.DB.DropTableIfExists("assets")
	database.DB.Model(&dto.Transaction{}).DropColumn("reserved_balance")
	database.DB.Model(&dto.Transaction{}).DropColumn("volume")
	database.DB.Model(&dto.Transaction{}).DropColumn("reversed_balance")
	database.DB.Model(&dto.Transaction{}).DropColumn("denomination")
	database.DB.Model(&dto.Transaction{}).DropColumn("recipient")
	database.DB.Model(&dto.Transaction{}).DropColumn("asset_id")
	database.DB.Model(&dto.UserBalance{}).DropColumn("asset_id")
	database.DB.Model(&dto.UserBalance{}).DropColumn("reversed_balance")
	database.DB.Model(&dto.UserBalance{}).DropColumn("reserved_balance")
	database.DB.Model(&dto.UserAddress{}).DropColumn("denomination_id")
	database.DB.Model(&dto.UserAddress{}).DropColumn("key_id")
	database.DB.Model(&dto.UserAddress{}).DropColumn("user_id")
	database.DB.Model(&dto.BatchRequest{}).DropColumn("asset_id")
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
