package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	Config "wallet-adapter/config"
	"wallet-adapter/dto"
	"wallet-adapter/utility"
)

// broadcastToChain ... Calls crypto adapter with signed transaction to be broadcast to chain
func BroadcastToChain(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, requestData dto.BroadcastToChainRequest, responseData *dto.BroadcastToChainResponse, serviceErr interface{}) error {

	authToken, err := GetAuthToken(cache, logger, config)
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("broadcastTransaction", config)

	APIClient := NewClient(nil, logger, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	APIResponse, err := APIClient.Do(APIRequest, responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), serviceErr); errUnmarshal != nil {
			return err
		}
		status := serviceErr.(*dto.ServicesRequestErr)
		status.StatusCode = APIResponse.StatusCode
		return err
	}

	return nil
}

func SubscribeAddress(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, requestData dto.SubscriptionRequest, responseData *dto.SubscriptionResponse, serviceErr interface{}) error {
	metaData := utility.GetRequestMetaData("subscribeAddress", config)
	APIClient := NewClient(nil, logger, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	_, err = APIClient.Do(APIRequest, responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), serviceErr); errUnmarshal != nil {
			return err
		}
		return err
	}

	if responseData.Status == false {
		err := utility.AppError{
			ErrType: "Could not subscribe address",
			Err:     nil,
		}
		return err
	}
	return nil
}

// TransactionStatus ... Calls crypto adapter with transaction hash to confirm transaction status on-chain
func TransactionStatus(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, requestData dto.TransactionStatusRequest, responseData *dto.TransactionStatusResponse, serviceErr interface{}) error {

	authToken, err := GetAuthToken(cache, logger, config)
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("transactionStatus", config)
	var APIClient *Client
	if requestData.TransactionHash != "" && requestData.Reference == "" {
		APIClient = NewClient(nil, logger, config, fmt.Sprintf("%s%s?transactionHash=%s&assetSymbol=%s", metaData.Endpoint, metaData.Action, requestData.TransactionHash, requestData.AssetSymbol))
	} else if requestData.Reference != "" && requestData.TransactionHash == "" {
		APIClient = NewClient(nil, logger, config, fmt.Sprintf("%s%s?reference=%s&assetSymbol=%s", metaData.Endpoint, metaData.Action, requestData.Reference, requestData.AssetSymbol))
	}

	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	APIResponse, err := APIClient.Do(APIRequest, responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), serviceErr); errUnmarshal != nil {
			return err
		}
		status := serviceErr.(*dto.ServicesRequestErr)
		status.StatusCode = APIResponse.StatusCode
		return err
	}

	return nil
}

// GetOnchainBalance ... Calls crypto adapter with asset symbol and address to return balance of asset on-chain
func GetOnchainBalance(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, requestData dto.OnchainBalanceRequest, responseData *dto.OnchainBalanceResponse, serviceErr interface{}) error {

	authToken, err := GetAuthToken(cache, logger, config)
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("getOnchainBalance", config)

	APIClient := NewClient(nil, logger, config, fmt.Sprintf("%s%s?address=%s&assetSymbol=%s", metaData.Endpoint, metaData.Action, requestData.Address, requestData.AssetSymbol))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", nil)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	_, err = APIClient.Do(APIRequest, responseData)
	if err != nil {
		logger.Error("An error occured when trying to get onChain Balance: ", err)
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), serviceErr); errUnmarshal != nil {
			return err
		}
		return err
	}

	return nil
}

// GetBroadcastedTXNStatusByRef ...
func GetBroadcastedTXNStatusByRef(transactionRef, assetSymbol string, cache *utility.MemoryCache, logger *utility.Logger, config Config.Data) bool {
	serviceErr := dto.ServicesRequestErr{}

	transactionStatusRequest := dto.TransactionStatusRequest{
		Reference:   transactionRef,
		AssetSymbol: assetSymbol,
	}
	transactionStatusResponse := dto.TransactionStatusResponse{}
	if err := TransactionStatus(cache, logger, config, transactionStatusRequest, &transactionStatusResponse, &serviceErr); err != nil {
		logger.Error("Error getting broadcasted transaction status : %+v", err)
		if serviceErr.StatusCode != http.StatusNotFound {
			return true
		}
		return false
	}
	return true
}

// GetBroadcastedTXNStatusByRef ...
func GetBroadcastedTXNDetailsByRef(transactionRef, assetSymbol string, cache *utility.MemoryCache, logger *utility.Logger, config Config.Data) (bool, dto.TransactionStatusResponse, error) {
	serviceErr := dto.ServicesRequestErr{}

	transactionStatusRequest := dto.TransactionStatusRequest{
		Reference:   transactionRef,
		AssetSymbol: assetSymbol,
	}
	transactionStatusResponse := dto.TransactionStatusResponse{}
	if err := TransactionStatus(cache, logger, config, transactionStatusRequest, &transactionStatusResponse, &serviceErr); err != nil {
		logger.Error("Error getting broadcasted transaction status : %+v", err)
		if serviceErr.StatusCode != http.StatusNotFound {
			return false, dto.TransactionStatusResponse{}, err
		}
		return false, dto.TransactionStatusResponse{}, nil
	}
	return true, transactionStatusResponse, nil
}
