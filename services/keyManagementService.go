package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	Config "wallet-adapter/config"
	"wallet-adapter/database"

	"wallet-adapter/dto"
	"wallet-adapter/utility/apiClient"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/logger"

	uuid "github.com/satori/go.uuid"
)

//KeyManagementService object
type KeyManagementService struct {
	Cache      *cache.Memory
	Config     Config.Data
	Error      *dto.ExternalServicesRequestErr
	Repository database.IRepository
}

func NewKeyManagementService(cache *cache.Memory, config Config.Data, repository database.IRepository, serviceErr *dto.ExternalServicesRequestErr) *KeyManagementService {
	baseService := KeyManagementService{
		Cache:      cache,
		Config:     config,
		Repository: repository,
		Error:      serviceErr,
	}
	return &baseService
}

// GenerateAddress ...
func (service *KeyManagementService) GenerateAddress(userID uuid.UUID, symbol string, coinType int64) (string, error) {
	generatedAddress, err := service.GenerateAddressWithoutSub(userID, symbol)
	if err != nil {
		return "", err
	}

	//call subscribe
	if err := service.subscribeAddress([]string{generatedAddress}, userID, coinType); err != nil {
		return "", err
	}

	return generatedAddress, nil
}

func (service *KeyManagementService) GenerateAddressWithoutSub(userID uuid.UUID, symbol string) (string, error) {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return "", err
	}
	requestData := dto.GenerateAddressRequest{}
	responseData := dto.GenerateAddressResponse{}
	metaData := GetRequestMetaData("createAddress", service.Config)

	requestData.UserID = userID
	requestData.AssetSymbol = symbol

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return "", err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	if err := APIClient.Do(APIRequest, &responseData); err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), service.Error); errUnmarshal != nil {
			return "", err
		}
		return "", serviceError(service.Error.StatusCode, service.Error.Code, errors.New(service.Error.Message))
	}

	logger.Info("Response from GenerateAddress : %+v", responseData)
	return responseData.Address, nil
}

// GenerateAllAddresses ...
func (service *KeyManagementService) GenerateAllAddresses(userID uuid.UUID, symbol string, coinType int64, addressType string) ([]dto.AllAddressResponse, error) {
	var APIClient *apiClient.Client
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return []dto.AllAddressResponse{}, err
	}
	requestData := dto.GenerateAddressRequest{}
	responseData := dto.GenerateAllAddressesResponse{}
	metaData := GetRequestMetaData("createAllAddresses", service.Config)

	requestData.UserID = userID
	requestData.AssetSymbol = symbol
	if addressType == "" {
		APIClient = apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	} else {
		APIClient = apiClient.New(nil, service.Config, fmt.Sprintf("%s%s?addressType=%s", metaData.Endpoint, metaData.Action, addressType))
	}
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return []dto.AllAddressResponse{}, err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	if err := APIClient.Do(APIRequest, &responseData); err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), service.Error); errUnmarshal != nil {
			return []dto.AllAddressResponse{}, err
		}
		return []dto.AllAddressResponse{}, serviceError(service.Error.StatusCode, service.Error.Code, errors.New(service.Error.Message))
	}

	addressArray := []string{}
	for _, item := range responseData.Addresses {
		addressArray = append(addressArray, item.Data)
	}

	//call subscribe
	if err := service.subscribeAddress(addressArray, userID, coinType); err != nil {
		return []dto.AllAddressResponse{}, err
	}

	return responseData.Addresses, nil
}

// SignTransaction ... Calls key-management service with a transaction object to sign
func (service *KeyManagementService) SignTransaction(requestData dto.SignTransactionRequest, responseData *dto.SignTransactionResponse) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("signTransaction", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	if err := APIClient.Do(APIRequest, &responseData); err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), service.Error); errUnmarshal != nil {
			return err
		}
		return serviceError(service.Error.StatusCode, service.Error.Code, errors.New(service.Error.Message))
	}

	return nil
}

// SignTransaction ... Calls key-management service with a transaction object to sign
func (service *KeyManagementService) SignTransactionAndBroadcast(requestData dto.SignTransactionRequest, responseData *dto.SignAndBroadcastResponse) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("signAndBroadcastTransaction", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	if err := APIClient.Do(APIRequest, &responseData); err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), service.Error); errUnmarshal != nil {
			return err
		}
		return serviceError(service.Error.StatusCode, service.Error.Code, errors.New(service.Error.Message))
	}

	return nil
}

func (service *KeyManagementService) SignBatchTransaction(HttpClient *http.Client, requestData dto.BatchBTCRequest, responseData *dto.SignTransactionResponse) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("signBatchTransaction", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	if HttpClient != nil {
		APIClient.HttpClient = HttpClient
	}
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	if err := APIClient.Do(APIRequest, &responseData); err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), service.Error); errUnmarshal != nil {
			return err
		}
		return serviceError(service.Error.StatusCode, service.Error.Code, errors.New(service.Error.Message))
	}
	return nil

}

func (service *KeyManagementService) SignBatchTransactionAndBroadcast(HttpClient *http.Client, requestData dto.BatchBTCRequest, responseData *dto.SignAndBroadcastResponse) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("signBatchTransactionAndbroadcast", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	if HttpClient != nil {
		APIClient.HttpClient = HttpClient
	}
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})

	if err := APIClient.Do(APIRequest, &responseData); err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), service.Error); errUnmarshal != nil {
			return err
		}
		return serviceError(service.Error.StatusCode, service.Error.Code, errors.New(service.Error.Message))
	}

	return nil

}

//does v1 and v2 address subscriptions
func (service *KeyManagementService) subscribeAddress(addressArray []string, userID uuid.UUID, coinType int64) error {

	subscriptionRequestDataV2 := dto.SubscriptionRequestV2{}
	subscriptionRequestDataV2.Subscriptions = make(map[string][]string)

	subscriptionRequestDataV2.Subscriptions[strconv.Itoa(int(coinType))] = addressArray

	subscriptionResponseData := dto.SubscriptionResponse{}
	CryptoAdapterService := NewCryptoAdapterService(service.Cache, service.Config, service.Repository, service.Error)
	if err := CryptoAdapterService.SubscribeAddressV2(subscriptionRequestDataV2, &subscriptionResponseData); err != nil {
		logger.Error("Failing to subscribe to addresses %+v with err %s\n", addressArray, err)
		return err
	}

	return nil
}
