package services

import (
	"fmt"
	"strings"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/utility"
	"wallet-adapter/utility/constants"

	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
)

var (
	yes    = true
	no = false
	isBatchable  = map[int64]*bool{
		0:   &yes,
		145: &yes,
		2:   &yes,
	}
	IsMultiAddresses  = map[int64]*bool{
		0:   &yes,
		145: &yes,
	}
	sweepFee = map[int64]int64{
		714: 37500,
	}
)

// Create additional network table


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
		for _, network := range asset.Networks {
			if err := DB.Where(model.Network{AssetSymbol: asset.AssetSymbol, Network: network.Network}).Assign(network).FirstOrCreate(&network).Error; err != nil {
				logger.Error("Error with creating asset record %s : %s", asset.AssetSymbol, err)
			}
		}
	}
	logger.Info("Supported assets seeded successfully")

}

func normalizeAsset(denominations []dto.AssetDenomination, TWDenominations []dto.TWDenomination) []model.Denomination {
	normalizedAssets := []model.Denomination{}
	normalizedNetworks := []model.Network{}

	for _, denom := range denominations {
		for _, network := range denom.AdditionalNetworks {
			if network.Network != "" {
				normalizedNetwork := normalizeNetwork(denom.Symbol, network)
				normalizedNetworks = append(normalizedNetworks, normalizedNetwork)
			}
		}
		// Add default network to network array
		nativeSymbol := getMainCoinAssetSymbol(denom.CoinType, TWDenominations)
		defaultNetwork := normalizeDefaultNetwork(denom, nativeSymbol)
		normalizedNetworks = append(normalizedNetworks, defaultNetwork)

		normalizedAsset := model.Denomination{
			Name:             denom.Name,
			AssetSymbol:      denom.Symbol,
			TradeActivity:    denom.TradeActivity,
			TransferActivity: denom.TransferActivity,
			DefaultNetwork:   denom.Network,
			Networks: normalizedNetworks,
		}
		normalizedAssets = append(normalizedAssets, normalizedAsset)
	}
	return normalizedAssets
}

func normalizeDefaultNetwork(denom dto.AssetDenomination, nativeSymbol string) model.Network {
	isToken, addressProvider := GetDynamicDenominationValues(denom.TokenType, denom.CoinType)
	defaultNetwork := model.Network{
		AssetSymbol:      denom.Symbol,
		CoinType:         denom.CoinType,
		RequiresMemo:     denom.RequiresMemo,
		NativeDecimals:   denom.NativeDecimal,
		NativeAsset :		nativeSymbol,
		IsToken:          &isToken,
		SweepFee:         sweepFee[denom.CoinType],
		DepositActivity:  denom.DepositActivity,
		WithdrawActivity: denom.WithdrawActivity,
		MinimumSweepable: viper.GetFloat64(fmt.Sprintf("MINIMUMSWEEP.%s_%s", denom.Symbol, denom.Network)),
		IsBatchable:      isBatchable[denom.CoinType],
		IsMultiAddresses: IsMultiAddresses[denom.CoinType],
		AddressProvider:  addressProvider,
		Network:          denom.Network,
	}
	return defaultNetwork
}

func normalizeNetwork(assetSymbol string, network dto.AdditionalNetwork) model.Network {
	isToken := false
	_, addressProvider := GetDynamicDenominationValues("", network.CoinType)
	if network.NativeAsset != assetSymbol {
		isToken = true
	}
	additionalNetwork := model.Network{
		AssetSymbol:         assetSymbol,
		CoinType:            network.CoinType,
		RequiresMemo:        network.RequiresMemo,
		NativeDecimals :     network.NativeDecimal,
		NativeAsset :		network.NativeAsset,
		IsToken:             &isToken,
		SweepFee:            sweepFee[network.CoinType],
		DepositActivity:     network.DepositActivity,
		WithdrawActivity:    network.WithdrawActivity,
		MinimumSweepable:    viper.GetFloat64(fmt.Sprintf("MINIMUMSWEEP.%s_%s", network.NativeAsset, network.Network)),
		IsBatchable:         isBatchable[network.CoinType],
		IsMultiAddresses:    IsMultiAddresses[network.CoinType],
		AddressProvider:     addressProvider,
		Network:            network.Network,
	}
	return additionalNetwork
}

func GetDynamicDenominationValues(tokenType string, coinType int64) (bool, string) {
	isToken := false
	addressProvider := model.AddressProvider.BUNDLE

	if tokenType != "" && !strings.EqualFold(tokenType, "NATIVE") {
		isToken = true
		if coinType == constants.ETH_COINTYPE {
			addressProvider = model.AddressProvider.BINANCE
		}
	}
	return isToken, addressProvider
}

func getMainCoinAssetSymbol(coinType int64, TWDenominations []dto.TWDenomination) string {

	for _, denom := range TWDenominations {
		if denom.CoinId == coinType {
			return denom.Symbol
		}
	}
	return ""
}

func (service BaseService) IsWithdrawalActive(assetSymbol, network string, repository database.IUserAssetRepository) (bool, error) {

	// Check if withdrawal is allowed on the network
	networkAsset, err := GetNetworkByAssetAndNetwork(repository, network, assetSymbol)
	if err != nil {
		if err.Error() == errorcode.SQL_404 {
			return false, nil
		}
		return false, err
	}

	if !strings.EqualFold(networkAsset.WithdrawActivity, utility.ACTIVE) {
		return false, nil
	}

	return true, nil
}

func (service BaseService) IsDepositActive(assetSymbol, network string, repository database.IUserAssetRepository) (bool, error) {

	// Check if deposit is allowed on the network
	networkAsset, err := GetNetworkByAssetAndNetwork(repository, network, assetSymbol)
	if err != nil {
		if err.Error() == errorcode.SQL_404 {
			return false, nil
		}
		return false, err
	}

	if !strings.EqualFold(networkAsset.DepositActivity, utility.ACTIVE) {
		return false, nil
	}

	return true, nil
}

func (service BaseService) IsBatchable(assetSymbol, network string, repository database.IUserAssetRepository) (bool, error) {

	networkAsset, err := GetNetworkByAssetAndNetwork(repository, network, assetSymbol)
	if err != nil {
		if err.Error() == errorcode.SQL_404 {
			return false, nil
		}
		return false, err
	}

	if !*networkAsset.IsBatchable  {
		return false, nil
	}

	return true, nil
}

func (service BaseService) IsMultipleAddresses(assetSymbol, network string, repository database.IUserAssetRepository) (bool, error) {

	networkAsset, err := GetNetworkByAssetAndNetwork(repository, network, assetSymbol);
	if err != nil {
		if err.Error() == errorcode.SQL_404 {
			return false, nil
		}
		return false, err
	}

	return *networkAsset.IsMultiAddresses, nil
}