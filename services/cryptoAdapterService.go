package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/utility/apiClient"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/logger"

	"wallet-adapter/dto"
)

//HotWalletService object
type CryptoAdapterService struct {
	Cache      *cache.Memory
	Config     Config.Data
	Error      *dto.ExternalServicesRequestErr
	Repository database.IRepository
}

func NewCryptoAdapterService(cache *cache.Memory, config Config.Data, repository database.IRepository, serviceErr *dto.ExternalServicesRequestErr) *CryptoAdapterService {
	baseService := CryptoAdapterService{
		Cache:      cache,
		Config:     config,
		Repository: repository,
		Error:      serviceErr,
	}
	return &baseService
}

// broadcastToChain ... Calls crypto adapter with signed transaction to be broadcast to chain
func (service *CryptoAdapterService) BroadcastToChain(requestData dto.BroadcastToChainRequest, responseData *dto.SignAndBroadcastResponse) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("broadcastTransaction", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	if err := APIClient.Do(APIRequest, responseData); err != nil {
		appErr := err.(appError.Err)
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%s", err.Error())), service.Error); errUnmarshal != nil {
			return err
		}
		return serviceError(appErr.ErrCode, service.Error.Code, errors.New(service.Error.Message))
	}

	return nil
}

func (service *CryptoAdapterService) SubscribeAddressV2(requestData dto.SubscriptionRequestV2, responseData *dto.SubscriptionResponse) error {
	metaData := GetRequestMetaData("subscribeAddressV2", service.Config)
	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	if err := APIClient.Do(APIRequest, responseData); err != nil {
		appErr := err.(appError.Err)
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%s", err.Error())), service.Error); errUnmarshal != nil {
			return err
		}
		return serviceError(appErr.ErrCode, service.Error.Code, errors.New(service.Error.Message))
	}
	return nil
}

// TransactionStatus ... Calls crypto adapter with transaction hash to confirm transaction status on-chain
func (service *CryptoAdapterService) TransactionStatus(requestData dto.TransactionStatusRequest, responseData *dto.TransactionStatusResponse) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("transactionStatus", service.Config)
	var APIClient *apiClient.Client
	if requestData.TransactionHash != "" && requestData.Reference == "" {
		APIClient = apiClient.New(nil, service.Config, fmt.Sprintf("%s%s?transactionHash=%s&assetSymbol=%s",
			metaData.Endpoint, metaData.Action, requestData.TransactionHash, requestData.AssetSymbol))
	} else if requestData.Reference != "" && requestData.TransactionHash == "" {
		APIClient = apiClient.New(nil, service.Config, fmt.Sprintf("%s%s?reference=%s&assetSymbol=%s",
			metaData.Endpoint, metaData.Action, requestData.Reference, requestData.AssetSymbol))
	}

	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	if err := APIClient.Do(APIRequest, responseData); err != nil {
		appErr := err.(appError.Err)
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%s", err.Error())), service.Error); errUnmarshal != nil {
			return err
		}
		return serviceError(appErr.ErrCode, service.Error.Code, errors.New(service.Error.Message))
	}

	return nil
}

// GetOnchainBalance ... Calls crypto adapter with asset symbol and address to return balance of asset on-chain
func (service *CryptoAdapterService) GetOnchainBalance(requestData dto.OnchainBalanceRequest, responseData *dto.OnchainBalanceResponse) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("getOnchainBalance", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s?address=%s&assetSymbol=%s", metaData.Endpoint, metaData.Action, requestData.Address, requestData.AssetSymbol))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", nil)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	if err := APIClient.Do(APIRequest, responseData); err != nil {
		appErr := err.(appError.Err)
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%s", err.Error())), service.Error); errUnmarshal != nil {
			return err
		}
		return serviceError(appErr.ErrCode, service.Error.Code, errors.New(service.Error.Message))
	}

	return nil
}

// GetBroadcastedTXNStatusByRef ...
func (service *CryptoAdapterService) GetBroadcastedTXNStatusByRef(transactionRef, assetSymbol string) bool {
	transactionStatusRequest := dto.TransactionStatusRequest{
		Reference:   transactionRef,
		AssetSymbol: assetSymbol,
	}
	transactionStatusResponse := dto.TransactionStatusResponse{}
	if err := service.TransactionStatus(transactionStatusRequest, &transactionStatusResponse); err != nil {
		logger.Error("Error getting broadcasted transaction status : %+v", err)
		if service.Error.StatusCode != http.StatusNotFound {
			return true
		}
		return false
	}
	return true
}

// GetBroadcastedTXNStatusByRef ...
func (service *CryptoAdapterService) GetBroadcastedTXNDetailsByRefAndSymbol(transactionRef, assetSymbol string) (bool, dto.TransactionStatusResponse, error) {
	transactionStatusRequest := dto.TransactionStatusRequest{
		Reference:   transactionRef,
		AssetSymbol: assetSymbol,
	}
	transactionStatusResponse := dto.TransactionStatusResponse{}
	if err := service.TransactionStatus(transactionStatusRequest, &transactionStatusResponse); err != nil {
		logger.Error("Error getting broadcasted transaction status : %+v", err)
		if service.Error.StatusCode != http.StatusNotFound {
			return false, dto.TransactionStatusResponse{}, err
		}
		return false, dto.TransactionStatusResponse{}, nil
	}
	return true, transactionStatusResponse, nil
}
