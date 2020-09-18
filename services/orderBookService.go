package services

import (
	"encoding/json"
	"fmt"
	Config "wallet-adapter/config"

	"wallet-adapter/dto"
	"wallet-adapter/utility"
	"wallet-adapter/utility/apiClient"
)

//OrderBookService object
type OrderBookService struct {
	Cache  *utility.MemoryCache
	Config Config.Data
	Error  *dto.ExternalServicesRequestErr
}

func NewOrderBookService(cache *utility.MemoryCache, config Config.Data) *OrderBookService {
	baseService := OrderBookService{
		Cache:  cache,
		Config: config,
	}
	return &baseService
}

// withdrawToHotWallet ... Calls order-book service to withdraw to specified hot wallet address
func (service *OrderBookService) WithdrawToHotWallet(cache *utility.MemoryCache, config Config.Data, requestData dto.WitdrawToHotWalletRequest, responseData *dto.WitdrawToHotWalletResponse, serviceErr interface{}) error {
	AuthService := NewAuthService(service.Cache, service.Config)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("withdrawToHotWallet", config)

	APIClient := apiClient.New(nil, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
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
func (service *OrderBookService) GetOnChainBinanceAssetBalances(cache *utility.MemoryCache, config Config.Data, responseData *dto.BinanceAssetBalances, serviceErr interface{}) error {
	AuthService := NewAuthService(service.Cache, service.Config)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("getAssetBalances", config)

	APIClient := apiClient.New(nil, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", nil)
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
func (service *OrderBookService) GetDepositAddress(cache *utility.MemoryCache, config Config.Data, coin string, network string, responseData *dto.DepositAddressResponse, serviceErr interface{}) error {
	AuthService := NewAuthService(service.Cache, service.Config)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("getDepositAddress", config)

	APIClient := apiClient.New(nil, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
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
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), serviceErr); errUnmarshal != nil {
			return err
		}
		return err
	}

	return nil
}
