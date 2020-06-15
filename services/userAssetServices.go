package services

import (
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"
)

func SeedSupportedAssets(DB *gorm.DB, logger *utility.Logger) {

	assets := []model.Denomination{
		model.Denomination{Name: "Binance Coin", AssetSymbol: "BNB", CoinType: 714, Decimal: 8, RequiresMemo: true},
		model.Denomination{Name: "Binance USD", AssetSymbol: "BUSD", CoinType: 714, RequiresMemo: true, Decimal: 8, IsToken: true, MainCoinAssetSymbol: "BNB", SweepFee: 37500},
		model.Denomination{Name: "Ethereum Coin", AssetSymbol: "ETH", CoinType: 60, Decimal: 18},
		model.Denomination{Name: "Bitcoin", AssetSymbol: "BTC", CoinType: 0, Decimal: 8},
	}

	for _, asset := range assets {
		if err := DB.Where(model.Denomination{AssetSymbol: asset.AssetSymbol}).Assign(asset).FirstOrCreate(&asset).Error; err != nil {
			logger.Error("Error with creating asset record %s : %s", asset.AssetSymbol, err)
		}
	}
	logger.Info("Supported assets seeded successfully")
}
