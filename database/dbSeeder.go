package database

import (
	"wallet-adapter/dto"
)


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
