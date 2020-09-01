package services

import (
	Config "wallet-adapter/config"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"
)

func SeedSupportedAssets(DB *gorm.DB, logger *utility.Logger, config Config.Data, cache *utility.MemoryCache) {

	// Get assets from rate service
	rateService := NewService(cache, logger, config)
	assetDenominations, err := rateService.GetAssetDenominations()
	if err != nil {
		logger.Fatal("Supported assets could not be seeded, err : %s", err)
	}

	assets := normalizeAsset(assetDenominations.Denominations)

	for _, asset := range assets {
		if err := DB.Where(model.Denomination{AssetSymbol: asset.AssetSymbol}).Assign(asset).FirstOrCreate(&asset).Error; err != nil {
			logger.Error("Error with creating asset record %s : %s", asset.AssetSymbol, err)
		}
	}
	logger.Info("Supported assets seeded successfully")
}

func normalizeAsset(denominations []dto.AssetDenomination) []model.Denomination {

	normalizedAssets := []model.Denomination{}

	for _, denom := range denominations {
		var isToken bool

		if denom.TokenType != "NATIVE" {
			isToken = true
		}

		normalizedAsset := model.Denomination{
			Name:                denom.Name,
			AssetSymbol:         denom.Symbol,
			CoinType:            denom.CoinType,
			RequiresMemo:        denom.RequiresMemo,
			Decimal:             denom.NativeDecimals,
			IsEnabled:           denom.Enabled,
			IsToken:             isToken,
			MainCoinAssetSymbol: getMainCoinAssetSymbol(denom.CoinType),
			SweepFee:            getAssetSweepFee(denom.CoinType),
			TradeActivity:       denom.TradeActivity,
			DepositActivity:     denom.DepositActivity,
			WithdrawActivity:    denom.WithdrawActivity,
			TransferActivity:    denom.TransferActivity,
		}
		normalizedAssets = append(normalizedAssets, normalizedAsset)
	}

	return normalizedAssets

}

func getMainCoinAssetSymbol(coinType int64) string {
	switch coinType {
	case 0:
		return utility.COIN_BTC
	case 60:
		return utility.COIN_ETH
	case 714:
		return utility.COIN_BNB
	default:
		return ""
	}
}

func getAssetSweepFee(coinType int64) int64 {
	switch coinType {
	case 714:
		return 37500
	default:
		return 0
	}
}
