package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"

	"wallet-adapter/utility/apiClient"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/constants"
	"wallet-adapter/utility/logger"

	"github.com/jinzhu/gorm"
)

//HotWalletService object
type DenominationServices struct {
	Cache      *cache.Memory
	Config     Config.Data
	Error      *dto.ExternalServicesRequestErr
	Repository database.IRepository
}

func NewDenominationServices(cache *cache.Memory, config Config.Data, repository database.IRepository) *DenominationServices {
	baseService := DenominationServices{
		Cache:      cache,
		Config:     config,
		Repository: repository,
		Error:      &dto.ExternalServicesRequestErr{},
	}
	return &baseService
}

// GetAssetDenominations Fetch all supported asset denominations from rate service
func (service *DenominationServices) GetAssetDenominations() (dto.AssetDenominations, error) {

	responseData := dto.AssetDenominations{}
	metaData := GetRequestMetaData("getAssetDenominations", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s?assetType=CRYPTO", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", nil)
	if err != nil {
		return responseData, err
	}
	if err := APIClient.Do(APIRequest, &responseData); err != nil {
		appErr := err.(appError.Err)
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%s", err.Error())), service.Error); errUnmarshal != nil {
			return responseData, err
		}
		return responseData, serviceError(appErr.ErrCode, service.Error.Code, errors.New(service.Error.Message))
	}

	logger.Info("Response from GetAssetDenominations : %+v", responseData)
	return responseData, nil

}

// GetTWDenominations, returns all coins and their info from TW
func (service *DenominationServices) GetTWDenominations() ([]dto.TWDenomination, error) {

	responseData := []dto.TWDenomination{}
	metaData := GetRequestMetaData("getTWDenominations", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", nil)
	if err != nil {
		return responseData, err
	}
	if err := APIClient.Do(APIRequest, &responseData); err != nil {
		appErr := err.(appError.Err)
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%s", err.Error())), service.Error); errUnmarshal != nil {
			return responseData, err
		}
		return responseData, serviceError(appErr.ErrCode, service.Error.Code, errors.New(service.Error.Message))
	}

	logger.Info("Response from GetTWDenominations : %+v", responseData)
	return responseData, nil

}

func (service *DenominationServices) SeedSupportedAssets(DB *gorm.DB) {

	// Get assets from rate service
	assetDenominations, err := service.GetAssetDenominations()
	if err != nil {
		logger.Fatal("Supported assets could not be seeded, err : %s", err)
	}

	TWDenominations, err := service.GetTWDenominations()
	if err != nil {
		logger.Fatal("Supported assets could not be seeded, err : %s", err)
	}

	assets := service.normalizeAsset(assetDenominations.Denominations, TWDenominations)

	for _, asset := range assets {
		if err := DB.Where(model.Denomination{AssetSymbol: asset.AssetSymbol}).Assign(asset).FirstOrCreate(&asset).Error; err != nil {
			logger.Error("Error with creating asset record %s : %s", asset.AssetSymbol, err)
		}
	}
	logger.Info("Supported assets seeded successfully")
}

func (service *DenominationServices) normalizeAsset(denominations []dto.AssetDenomination, TWDenominations []dto.TWDenomination) []model.Denomination {

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
			MainCoinAssetSymbol: service.getMainCoinAssetSymbol(denom.CoinType, TWDenominations),
			SweepFee:            service.getAssetSweepFee(denom.CoinType),
			TradeActivity:       denom.TradeActivity,
			DepositActivity:     denom.DepositActivity,
			WithdrawActivity:    denom.WithdrawActivity,
			TransferActivity:    denom.TransferActivity,
		}
		normalizedAssets = append(normalizedAssets, normalizedAsset)
	}

	return normalizedAssets

}

func (service *DenominationServices) getMainCoinAssetSymbol(coinType int64, TWDenominations []dto.TWDenomination) string {

	for _, denom := range TWDenominations {
		if denom.CoinId == coinType {
			return denom.Symbol
		}
	}
	return ""
}

func (service *DenominationServices) getAssetSweepFee(coinType int64) int64 {
	switch coinType {
	case 714:
		return 37500
	default:
		return 0
	}
}

func (service *DenominationServices) IsWithdrawalActive(assetSymbol string) (bool, error) {
	repository := service.Repository.(database.IUserAssetRepository)
	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: assetSymbol, IsEnabled: true}, &denomination); err != nil {
		return false, err
	}

	if !strings.EqualFold(denomination.WithdrawActivity, constants.ACTIVE) {
		return false, nil
	}

	return true, nil
}

func (service *DenominationServices) IsDepositActive(assetSymbol string) (bool, error) {
	repository := service.Repository.(database.IUserAssetRepository)
	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: assetSymbol, IsEnabled: true}, &denomination); err != nil {
		return false, err
	}

	if !strings.EqualFold(denomination.DepositActivity, constants.ACTIVE) {
		return false, nil
	}

	return true, nil
}
