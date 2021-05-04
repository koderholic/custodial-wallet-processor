package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/trustwallet/blockatlas/pkg/logger"
)

// GetAssetDenominations Fetch all supported asset denominations from rate service
func (service BaseService) GetAssetDenominations() (dto.AssetDenominations, error) {

	responseData := dto.AssetDenominations{}
	metaData := utility.GetRequestMetaData("getAssetDenominations", service.Config)

	APIClient := NewClient(nil, service.Logger, service.Config, fmt.Sprintf("%s%s?assetType=CRYPTO", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", nil)
	if err != nil {
		return responseData, err
	}
	_, err = APIClient.Do(APIRequest, &responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), service.Error); errUnmarshal != nil {
			return responseData, err
		}
		return responseData, errors.New(service.Error.Message)
	}

	logger.Info("Response from GetAssetDenominations : %+v", responseData)
	return responseData, nil

}

// GetTWDenominations, returns all coins and their info from TW
func (service BaseService) GetTWDenominations() ([]dto.TWDenomination, error) {

	responseData := []dto.TWDenomination{}
	metaData := utility.GetRequestMetaData("getTWDenominations", service.Config)

	APIClient := NewClient(nil, service.Logger, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", nil)
	if err != nil {
		return responseData, err
	}
	_, err = APIClient.Do(APIRequest, &responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), service.Error); errUnmarshal != nil {
			return responseData, err
		}
		return responseData, errors.New(service.Error.Message)
	}

	logger.Info("Response from GetTWDenominations : %+v", responseData)
	return responseData, nil

}

func (service BaseService) GetNetworksByDenom(repository database.IUserAssetRepository, denom string) ([]model.Network, error)  {
	additionalNetworks := []model.Network{}
	if err := repository.GetByFieldName(&model.Network{AssetSymbol: denom}, &additionalNetworks); err != nil {
		return additionalNetworks, err
	}
	return additionalNetworks, nil
}

func GetNetworkByAssetAndNetwork(repository database.IUserAssetRepository, network, assetSymbol string) (model.Network, error)  {
	networkAsset := model.Network{}
	if err := repository.GetByFieldName(&model.Network{Network: network, AssetSymbol: assetSymbol}, &networkAsset); err != nil {
		return networkAsset, err
	}
	return networkAsset, nil
}

func GetDefaultNetworkByAssetSymbol(repository database.IUserAssetRepository, assetSymbol string) (string, error)  {
	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: assetSymbol}, &denomination); err != nil {
		return "", err
	}
	return denomination.DefaultNetwork, nil
}