package services

import (
	"encoding/json"
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
	generatedAddress, err := service.GenerateAddressWithoutSub(service.Cache, service.Config, userID, symbol, service.Error)
	if err != nil {
		return "", err
	}

	//call subscribe
	if err := service.subscribeAddress(service.Error, []string{generatedAddress}, userID, coinType); err != nil {
		return "", err
	}

	return generatedAddress, nil
}

func (service *KeyManagementService) GenerateAddressWithoutSub(cache *cache.Memory, config Config.Data, userID uuid.UUID, symbol string, serviceErr interface{}) (string, error) {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return "", err
	}
	requestData := dto.GenerateAddressRequest{}
	responseData := dto.GenerateAddressResponse{}
	metaData := GetRequestMetaData("createAddress", config)

	requestData.UserID = userID
	requestData.AssetSymbol = symbol

	APIClient := apiClient.New(nil, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return "", err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	_, err = APIClient.Do(APIRequest, &responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), serviceErr); errUnmarshal != nil {
			return "", err
		}
		return "", err
	}

	logger.Info("Response from GenerateAddress : %+v", responseData)
	return responseData.Address, nil
}

// GenerateAllAddresses ...
func (service *KeyManagementService) GenerateAllAddresses(userID uuid.UUID, symbol string, coinType int64, addressType string, serviceErr interface{}) ([]dto.AllAddressResponse, error) {
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
	_, err = APIClient.Do(APIRequest, &responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), serviceErr); errUnmarshal != nil {
			return []dto.AllAddressResponse{}, err
		}
		return []dto.AllAddressResponse{}, err
	}
	addressArray := []string{}
	for _, item := range responseData.Addresses {
		addressArray = append(addressArray, item.Data)
	}

	//call subscribe
	if err := service.subscribeAddress(serviceErr, addressArray, userID, coinType); err != nil {
		return []dto.AllAddressResponse{}, err
	}

	return responseData.Addresses, nil
}

// SignTransaction ... Calls key-management service with a transaction object to sign
func (service *KeyManagementService) SignTransaction(cache *cache.Memory, config Config.Data, requestData dto.SignTransactionRequest, responseData *dto.SignTransactionResponse, serviceErr interface{}) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("signTransaction", config)

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
	APIResponse, err := APIClient.Do(APIRequest, responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%+v", err)), service.Error); errUnmarshal != nil {
			return err
		}
		service.Error.StatusCode = APIResponse.StatusCode
		return err
	}

	return nil
}

func (service *KeyManagementService) SignBatchTransaction(HttpClient *http.Client, cache *cache.Memory, config Config.Data, requestData dto.BatchBTCRequest, responseData *dto.SignTransactionResponse, serviceErr interface{}) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("signBatchTransaction", config)

	APIClient := apiClient.New(nil, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
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

func (service *KeyManagementService) SignBatchTransactionAndBroadcast(HttpClient *http.Client, cache *cache.Memory, config Config.Data, requestData dto.BatchBTCRequest, responseData *dto.SignAndBroadcastResponse, serviceErr interface{}) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("signBatchTransactionAndbroadcast", config)

	APIClient := apiClient.New(nil, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
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

//does v1 and v2 address subscriptions
func (service *KeyManagementService) subscribeAddress(serviceErr interface{}, addressArray []string, userID uuid.UUID, coinType int64) error {

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
