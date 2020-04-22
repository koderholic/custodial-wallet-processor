package database

import (
	"wallet-adapter/dto"
)

// DBSeeder .. This seeds supported assets into the database
func (database *Database) DBSeeder() {

	assets := []dto.Denomination{
		dto.Denomination{Name: "Binance Coin", AssetSymbol: "BNB", CoinType: 714, Decimal: 8},
		dto.Denomination{Name: "Binance USD", AssetSymbol: "BUSD", CoinType: 714, Decimal: 8},
		dto.Denomination{Name: "Ethereum Coin", AssetSymbol: "ETH", CoinType: 60, Decimal: 18},
		dto.Denomination{Name: "Bitcoin", AssetSymbol: "BTC", CoinType: 0, Decimal: 8},
	}

	for _, asset := range assets {
		if err := database.DB.FirstOrCreate(&asset, dto.Denomination{AssetSymbol: asset.AssetSymbol}).Error; err != nil {
			database.Logger.Error("Error with creating asset record %s : %s", asset.AssetSymbol, err)
		}
	}
	database.Logger.Info("Supported assets seeded successfully")
}
