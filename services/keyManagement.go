package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	Config "wallet-adapter/config"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	uuid "github.com/satori/go.uuid"
)

// GenerateAddress ...
func GenerateAddress(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, userID uuid.UUID, symbol string, serviceErr interface{}) (string, error) {

	authToken, err := GetAuthToken(cache, logger, config)
	if err != nil {
		return "", err
	}
	requestData := model.GenerateAddressRequest{}
	responseData := model.GenerateAddressResponse{}
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
	//call subscribe
	subscriptionRequestData := model.SubscriptionRequest{}
	subscriptionRequestData.Subscriptions = make(map[string][]string)
	addressArray := []string{responseData.Address}
	switch symbol {
	case "BTC":
		subscriptionRequestData.Subscriptions[config.BtcSlipValue] = addressArray
		break
	case "ETH":
		subscriptionRequestData.Subscriptions[config.EthSlipValue] = addressArray
		break
	case "BNB":
		subscriptionRequestData.Subscriptions[config.BnbSlipValue] = addressArray
		break
	}
	subscriptionRequestData.Webhook = config.DepositWebhookURL

	subscriptionResponseData := model.SubscriptionResponse{}

	if err := SubscribeAddress(cache, logger, config, subscriptionRequestData, &subscriptionResponseData, serviceErr); err != nil {
		logger.Error("Failing to subscribe to address %s with err %s\n", responseData.Address, err)
		return "", err
	}

	logger.Info("Response from GenerateAddress : %+v", responseData)
	return responseData.Address, nil
}

// SignTransaction ... Calls key-management service with a transaction object to sign
func SignTransaction(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, requestData model.SignTransactionRequest, responseData *model.SignTransactionResponse, serviceErr interface{}) error {

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

func SignBatchBTCTransaction(httpClient *http.Client, cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, requestData model.BatchBTCRequest, responseData *model.SignTransactionResponse, serviceErr interface{}) error {
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
