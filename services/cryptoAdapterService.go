package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	Config "wallet-adapter/config"
	"wallet-adapter/utility/apiClient"
	"wallet-adapter/utility/logger"

	"wallet-adapter/dto"
	"wallet-adapter/utility"
)

//HotWalletService object
type CryptoAdapterService struct {
	Cache  *utility.MemoryCache
	Config Config.Data
	Error  *dto.ExternalServicesRequestErr
}

func NewCryptoAdapterService(cache *utility.MemoryCache, config Config.Data) *CryptoAdapterService {
	baseService := CryptoAdapterService{
		Cache:  cache,
		Config: config,
	}
	return &baseService
}

// broadcastToChain ... Calls crypto adapter with signed transaction to be broadcast to chain
func (service *CryptoAdapterService) BroadcastToChain(cache *utility.MemoryCache, config Config.Data, requestData dto.BroadcastToChainRequest, responseData *dto.SignAndBroadcastResponse, serviceErr interface{}) error {
	AuthService := NewAuthService(service.Cache, service.Config)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("broadcastTransaction", config)

	APIClient := apiClient.New(nil, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	APIResponse, err := APIClient.Do(APIRequest, responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%+v", err)), serviceErr); errUnmarshal != nil {
			return err
		}
		errWithStatus := serviceErr.(*dto.ExternalServicesRequestErr)
		errWithStatus.StatusCode = APIResponse.StatusCode
		serviceErr = errWithStatus
		return err
	}

	return nil
}

func (service *CryptoAdapterService) SubscribeAddressV1(cache *utility.MemoryCache, config Config.Data, requestData dto.SubscriptionRequestV1, responseData *dto.SubscriptionResponse, serviceErr interface{}) error {
	metaData := utility.GetRequestMetaData("subscribeAddressV1", config)
	APIClient := apiClient.New(nil, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
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
func (service *CryptoAdapterService) TransactionStatus(cache *utility.MemoryCache, config Config.Data, requestData dto.TransactionStatusRequest, responseData *dto.TransactionStatusResponse, serviceErr interface{}) error {
	AuthService := NewAuthService(service.Cache, service.Config)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("transactionStatus", config)
	var APIClient *apiClient.Client
	if requestData.TransactionHash != "" && requestData.Reference == "" {
		APIClient = apiClient.New(nil, config, fmt.Sprintf("%s%s?transactionHash=%s&assetSymbol=%s", metaData.Endpoint, metaData.Action, requestData.TransactionHash, requestData.AssetSymbol))
	} else if requestData.Reference != "" && requestData.TransactionHash == "" {
		APIClient = apiClient.New(nil, config, fmt.Sprintf("%s%s?reference=%s&assetSymbol=%s", metaData.Endpoint, metaData.Action, requestData.Reference, requestData.AssetSymbol))
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
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%+v", err)), serviceErr); errUnmarshal != nil {
			return err
		}
		errWithStatus := serviceErr.(*dto.ExternalServicesRequestErr)
		errWithStatus.StatusCode = APIResponse.StatusCode
		serviceErr = errWithStatus
		return err
	}

	return nil
}

// GetOnchainBalance ... Calls crypto adapter with asset symbol and address to return balance of asset on-chain
func (service *CryptoAdapterService) GetOnchainBalance(cache *utility.MemoryCache, config Config.Data, requestData dto.OnchainBalanceRequest, responseData *dto.OnchainBalanceResponse, serviceErr interface{}) error {
	AuthService := NewAuthService(service.Cache, service.Config)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("getOnchainBalance", config)

	APIClient := apiClient.New(nil, config, fmt.Sprintf("%s%s?address=%s&assetSymbol=%s", metaData.Endpoint, metaData.Action, requestData.Address, requestData.AssetSymbol))
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
func (service *CryptoAdapterService) GetBroadcastedTXNStatusByRef(transactionRef, assetSymbol string, cache *utility.MemoryCache, config Config.Data) bool {
	serviceErr := dto.ExternalServicesRequestErr{}

	transactionStatusRequest := dto.TransactionStatusRequest{
		Reference:   transactionRef,
		AssetSymbol: assetSymbol,
	}
	transactionStatusResponse := dto.TransactionStatusResponse{}
	if err := service.TransactionStatus(cache, config, transactionStatusRequest, &transactionStatusResponse, &serviceErr); err != nil {
		logger.Error("Error getting broadcasted transaction status : %+v", err)
		if serviceErr.StatusCode != http.StatusNotFound {
			return true
		}
		return false
	}
	return true
}

// GetBroadcastedTXNStatusByRef ...
func (service *CryptoAdapterService) GetBroadcastedTXNDetailsByRef(transactionRef, assetSymbol string, cache *utility.MemoryCache, config Config.Data) (bool, dto.TransactionStatusResponse, error) {
	serviceErr := dto.ExternalServicesRequestErr{}

	transactionStatusRequest := dto.TransactionStatusRequest{
		Reference:   transactionRef,
		AssetSymbol: assetSymbol,
	}
	transactionStatusResponse := dto.TransactionStatusResponse{}
	if err := service.TransactionStatus(cache, config, transactionStatusRequest, &transactionStatusResponse, &serviceErr); err != nil {
		logger.Error("Error getting broadcasted transaction status : %+v", err)
		if serviceErr.StatusCode != http.StatusNotFound {
			return false, dto.TransactionStatusResponse{}, err
		}
		return false, dto.TransactionStatusResponse{}, nil
	}
	return true, transactionStatusResponse, nil
}
