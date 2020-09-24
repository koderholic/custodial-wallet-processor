package services

import (
	"encoding/json"
	"fmt"
	Config "wallet-adapter/config"

	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/utility/apiClient"
	"wallet-adapter/utility/cache"
)

//OrderBookService object
type OrderBookService struct {
	Cache      *cache.Memory
	Config     Config.Data
	Error      *dto.ExternalServicesRequestErr
	Repository database.IRepository
}

func NewOrderBookService(cache *cache.Memory, config Config.Data, repository database.IRepository, serviceErr *dto.ExternalServicesRequestErr) *OrderBookService {
	baseService := OrderBookService{
		Cache:      cache,
		Config:     config,
		Repository: repository,
		Error:      serviceErr,
	}
	return &baseService
}

// withdrawToHotWallet ... Calls order-book service to withdraw to specified hot wallet address
func (service *OrderBookService) WithdrawToHotWallet(requestData dto.WitdrawToHotWalletRequest, responseData *dto.WitdrawToHotWalletResponse, serviceErr interface{}) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("withdrawToHotWallet", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	_, err = APIClient.Do(APIRequest, responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), serviceErr); errUnmarshal != nil {
			return err
		}
		return err
	}

	return nil
}

// withdrawToHotWallet ... Calls order-book service to get asset details
func (service *OrderBookService) GetOnChainBinanceAssetBalances(responseData *dto.BinanceAssetBalances) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("getAssetBalances", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", nil)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	_, err = APIClient.Do(APIRequest, responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), service.Error); errUnmarshal != nil {
			return err
		}
		return err
	}

	return nil
}

// withdrawToHotWallet ... Calls order-book service to get asset details
func (service *OrderBookService) GetDepositAddress(coin string, network string, responseData *dto.DepositAddressResponse) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("getDepositAddress", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", nil)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	params := APIRequest.URL.Query()
	params.Add("coin", coin) // Add a new value to the set.
	if network != "" {
		params.Add("network", network)
	}
	APIRequest.URL.RawQuery = params.Encode() // Encode and assign back to the original query.

	_, err = APIClient.Do(APIRequest, responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), service.Error); errUnmarshal != nil {
			return err
		}
		return err
	}

	return nil
}
