package services

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	if err := service.subscribeAddress(serviceErr, []string{generatedAddress}, userID, coinType); err != nil {
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
func (service BaseService) GenerateAllAddresses(userID uuid.UUID, symbol string, coinType int64, addressType string, serviceErr interface{}) ([]dto.AllAddressResponse, error) {
	var APIClient *Client

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
func SignTransaction(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, requestData dto.SignTransactionRequest, responseData *dto.SignTransactionResponse, serviceErr interface{}) error {

	authToken, err := GetAuthToken(cache, logger, config)
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("signTransaction", config)

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

func SignBatchBTCTransaction(httpClient *http.Client, cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, requestData dto.BatchBTCRequest, responseData *dto.SignTransactionResponse, serviceErr interface{}) error {
	authToken, err := GetAuthToken(cache, logger, config)
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("signBatchTransaction", config)

	APIClient := NewClient(nil, logger, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	if httpClient != nil {
		APIClient.httpClient = httpClient
	}
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

//does v1 and v2 address subscriptions
func (service BaseService) subscribeAddress(serviceErr interface{}, addressArray []string, userID uuid.UUID, coinType int64) error {

	subscriptionRequestData := dto.SubscriptionRequestV1{}
	subscriptionRequestDataV2 := dto.SubscriptionRequestV2{}
	subscriptionRequestData.Subscriptions = make(map[string][]string)
	subscriptionRequestDataV2.Subscriptions = make(map[string][]string)
	switch coinType {
	case 0:
		subscriptionRequestData.Subscriptions[service.Config.BtcSlipValue] = addressArray
		subscriptionRequestDataV2.Subscriptions[service.Config.BtcSlipValue] = addressArray
		break
	case 60:
		subscriptionRequestData.Subscriptions[service.Config.EthSlipValue] = addressArray
		subscriptionRequestDataV2.Subscriptions[service.Config.EthSlipValue] = addressArray
		break
	case 714:
		subscriptionRequestData.Subscriptions[service.Config.BnbSlipValue] = addressArray
		subscriptionRequestDataV2.Subscriptions[service.Config.BnbSlipValue] = addressArray
		break
	}
	subscriptionRequestData.Webhook = service.Config.DepositWebhookURL
	subscriptionRequestDataV2.UserId = userID

	subscriptionResponseData := dto.SubscriptionResponse{}
	if err := SubscribeAddressV1(service.Cache, service.Logger, service.Config, subscriptionRequestData, &subscriptionResponseData, serviceErr); err != nil {
		service.Logger.Error("Failing to subscribe to addresses %+v with err %s\n", addressArray, err)
		return err
	}

	return nil
}
