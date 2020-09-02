package services

import (
	"strings"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"
)

func SeedSupportedAssets(DB *gorm.DB, logger *utility.Logger, config Config.Data, cache *utility.MemoryCache) {

	// Get assets from rate service
	denominationService := NewService(cache, logger, config)
	assetDenominations, err := denominationService.GetAssetDenominations()
	if err != nil {
		logger.Fatal("Supported assets could not be seeded, err : %s", err)
	}

	TWDenominations, err := denominationService.GetTWDenominations()
	if err != nil {
		logger.Fatal("Supported assets could not be seeded, err : %s", err)
	}

	assets := normalizeAsset(assetDenominations.Denominations, TWDenominations)

	for _, asset := range assets {
		if err := DB.Where(model.Denomination{AssetSymbol: asset.AssetSymbol}).Assign(asset).FirstOrCreate(&asset).Error; err != nil {
			logger.Error("Error with creating asset record %s : %s", asset.AssetSymbol, err)
		}
	}
	logger.Info("Supported assets seeded successfully")
}

func normalizeAsset(denominations []dto.AssetDenomination, TWDenominations []dto.TWDenomination) []model.Denomination {

	normalizedAssets := []model.Denomination{}

	for _, denom := range denominations {
		var isToken bool

		if strings.EqualFold(denom.TokenType, "NATIVE") {
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
			MainCoinAssetSymbol: getMainCoinAssetSymbol(denom.CoinType, TWDenominations),
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

func getMainCoinAssetSymbol(coinType int64, TWDenominations []dto.TWDenomination) string {

	for _, denom := range TWDenominations {
		if denom.CoinId == coinType {
			return denom.Symbol
		}
	}
	return ""
}

func getAssetSweepFee(coinType int64) int64 {
	switch coinType {
	case 714:
		return 37500
	default:
		return 0
	}
}

func (service BaseService) IsWithdrawalActive(assetSymbol string, repository database.IUserAssetRepository) (bool, error) {
	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: assetSymbol, IsEnabled: true}, &denomination); err != nil {
		return false, err
	}

	if !strings.EqualFold(denomination.WithdrawActivity, utility.ACTIVE) {
		return false, nil
	}

	return true, nil
}
