package services

import (
	"encoding/json"
	"fmt"
	Config "wallet-adapter/config"
	"wallet-adapter/dto"
	"wallet-adapter/utility"
)

// withdrawToHotWallet ... Calls order-book service to withdraw to specified hot wallet address
func WithdrawToHotWallet(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, requestData dto.WitdrawToHotWalletRequest, responseData *dto.WitdrawToHotWalletResponse, serviceErr interface{}) error {

	authToken, err := GetAuthToken(cache, logger, config)
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("withdrawToHotWallet", config)

	APIClient := NewClient(nil, logger, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
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
func GetOnChainBinanceAssetBalances(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, responseData *dto.BinanceAssetBalances, serviceErr interface{}) error {

	authToken, err := GetAuthToken(cache, logger, config)
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("getAssetBalances", config)

	APIClient := NewClient(nil, logger, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
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
func GetDepositAddress(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, coin string, network string, responseData *dto.DepositAddressResponse, serviceErr interface{}) error {

	authToken, err := GetAuthToken(cache, logger, config)
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("getDepositAddress", config)

	APIClient := NewClient(nil, logger, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
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
