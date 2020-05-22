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
func GenerateAddress(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, userID uuid.UUID, symbol string, coinType int64, serviceErr interface{}) (string, error) {

	generatedAddress, err := GenerateAddressWithoutSub(cache, logger, config, userID, symbol, serviceErr)
	if err != nil {
		return "", err
	}

	//call subscribe
	subscriptionRequestData := dto.SubscriptionRequest{}
	subscriptionRequestData.Subscriptions = make(map[string][]string)
	addressArray := []string{generatedAddress}
	switch coinType {
	case 0:
		subscriptionRequestData.Subscriptions[config.BtcSlipValue] = addressArray
		break
	case 60:
		subscriptionRequestData.Subscriptions[config.EthSlipValue] = addressArray
		break
	case 714:
		subscriptionRequestData.Subscriptions[config.BnbSlipValue] = addressArray
		break
	}
	subscriptionRequestData.Webhook = config.DepositWebhookURL

	subscriptionResponseData := dto.SubscriptionResponse{}

	if err := SubscribeAddress(cache, logger, config, subscriptionRequestData, &subscriptionResponseData, serviceErr); err != nil {
		logger.Error("Failing to subscribe to address %s with err %s\n", generatedAddress, err)
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
