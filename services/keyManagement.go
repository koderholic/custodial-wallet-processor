package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	Config "wallet-adapter/config"
	"wallet-adapter/dto"
	"wallet-adapter/utility"

	uuid "github.com/satori/go.uuid"
)

// GenerateAddress ...
func (service BaseService) GenerateAddress(userID uuid.UUID, symbol string, coinType int64, serviceErr interface{}) (string, error) {

	generatedAddress, err := GenerateAddressWithoutSub(service.Cache, service.Logger, service.Config, userID, symbol, serviceErr)
	if err != nil {
		return "", err
	}

	//call subscribe
	if err := service.subscribeAddress(serviceErr, []string{generatedAddress}, coinType); err != nil {
		return "", err
	}

	return generatedAddress, nil
}

func GenerateAddressWithoutSub(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, userID uuid.UUID, symbol string, serviceErr interface{}) (string, error) {

	authToken, err := GetAuthToken(cache, logger, config)
	if err != nil {
		return "", err
	}
	requestData := dto.GenerateAddressRequest{}
	responseData := dto.GenerateAddressResponse{}
	metaData := utility.GetRequestMetaData("createAddress", config)

	requestData.UserID = userID
	requestData.AssetSymbol = symbol

	APIClient := NewClient(nil, logger, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
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
func (service BaseService) GenerateAllAddresses(userID uuid.UUID, symbol string, coinType int64, addressType string) ([]dto.AllAddressResponse, error) {
	var APIClient *Client
	var serviceErr dto.ServicesRequestErr
	authToken, err := GetAuthToken(service.Cache, service.Logger, service.Config)
	if err != nil {
		return []dto.AllAddressResponse{}, err
	}
	requestData := dto.GenerateAddressRequest{}
	responseData := dto.GenerateAllAddressesResponse{}
	metaData := utility.GetRequestMetaData("createAllAddresses", service.Config)

	requestData.UserID = userID
	requestData.AssetSymbol = symbol
	if addressType == "" {
		APIClient = NewClient(nil, service.Logger, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	} else {
		APIClient = NewClient(nil, service.Logger, service.Config, fmt.Sprintf("%s%s?addressType=%s", metaData.Endpoint, metaData.Action, addressType))
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
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), &serviceErr); errUnmarshal != nil {
			return []dto.AllAddressResponse{}, err
		}
		return []dto.AllAddressResponse{}, errors.New(serviceErr.Message)
	}
	addressArray := []string{}
	for _, item := range responseData.Addresses {
		addressArray = append(addressArray, item.Data)
	}

	//call subscribe
	if err := service.subscribeAddress(serviceErr, addressArray, coinType); err != nil {
		return []dto.AllAddressResponse{}, err
	}

	return responseData.Addresses, nil
}

//does v1 and v2 address subscriptions
func (service BaseService) subscribeAddress(serviceErr interface{}, addressArray []string, coinType int64) error {

	subscriptionRequestDataV2 := dto.SubscriptionRequestV2{}
	subscriptionRequestDataV2.Subscriptions = make(map[string][]string)

	subscriptionRequestDataV2.Subscriptions[strconv.Itoa(int(coinType))] = addressArray

	subscriptionResponseData := dto.SubscriptionResponse{}
	if err := SubscribeAddressV2(service.Cache, service.Logger, service.Config, subscriptionRequestDataV2, &subscriptionResponseData, serviceErr); err != nil {
		service.Logger.Error("Failing to subscribe to addresses %+v with err %s\n", addressArray, err)
		return err
	}

	return nil
}
