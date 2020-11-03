package services

import (
	"fmt"
	"strings"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
)

var (
	batchable    = true
	notBatchable = false
	isBatchable  = map[int64]*bool{
		0:   &batchable,
		145: &batchable,
		2:   &batchable,
	}
	addressProvider  = map[string]string{
		"LINK":   model.AddressProvider.BINANCE,
		"USDT":  model.AddressProvider.BINANCE,
		"TRX":    model.AddressProvider.BINANCE,
	}
	sweepFee = map[int64]int64{
		714: 37500,
	}
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

	assets := normalizeAsset(config, assetDenominations.Denominations, TWDenominations)

	for _, asset := range assets {
		if err := DB.Where(model.Denomination{AssetSymbol: asset.AssetSymbol}).Assign(asset).FirstOrCreate(&asset).Error; err != nil {
			logger.Error("Error with creating asset record %s : %s", asset.AssetSymbol, err)
		}
	}
	logger.Info("Supported assets seeded successfully")
}

func normalizeAsset(config Config.Data, denominations []dto.AssetDenomination, TWDenominations []dto.TWDenomination) []model.Denomination {
	normalizedAssets := []model.Denomination{}

	for _, denom := range denominations {
		isToken := false

		if !strings.EqualFold(denom.TokenType, "NATIVE") {
			isToken = true
		}

		normalizedAsset := model.Denomination{
			Name:                denom.Name,
			AssetSymbol:         denom.Symbol,
			CoinType:            denom.CoinType,
			RequiresMemo:        denom.RequiresMemo,
			Decimal:             denom.NativeDecimals,
			IsEnabled:           denom.Enabled,
			IsToken:             &isToken,
			MainCoinAssetSymbol: getMainCoinAssetSymbol(denom.CoinType, TWDenominations),
			SweepFee:            sweepFee[denom.CoinType],
			TradeActivity:       denom.TradeActivity,
			DepositActivity:     denom.DepositActivity,
			WithdrawActivity:    denom.WithdrawActivity,
			TransferActivity:    denom.TransferActivity,
			MinimumSweepable:    viper.GetFloat64(fmt.Sprintf("MINIMUMSWEEP.%s", denom.Symbol)),
			IsBatchable:         isBatchable[denom.CoinType],
			AddressProvider: addressProvider[denom.Symbol],
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

func (service BaseService) IsDepositActive(assetSymbol string, repository database.IUserAssetRepository) (bool, error) {
	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: assetSymbol, IsEnabled: true}, &denomination); err != nil {
		return false, err
	}

	if !strings.EqualFold(denomination.DepositActivity, utility.ACTIVE) {
		return false, nil
	}

	return true, nil
}

func (service BaseService) IsBatchable(assetSymbol string, repository database.IUserAssetRepository) (bool, error) {
	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: assetSymbol, IsEnabled: true}, &denomination); err != nil {
		return false, err
	}

	return *denomination.IsBatchable, nil
}
