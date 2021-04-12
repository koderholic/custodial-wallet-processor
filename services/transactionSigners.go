package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	Config "wallet-adapter/config"
	"wallet-adapter/dto"
	"wallet-adapter/utility"
)

// SendSingleTransaction ... Calls transaction-signers service with a transaction object to sign and send to chain
func SendSingleTransaction(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, requestData dto.SendSingleTransactionRequest,
	responseData *dto.SendTransactionResponse, serviceErr interface{}) error {

	authToken, err := GetAuthToken(cache, logger, config)
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("sendSingleTransaction", config)

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
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%+v", err)), serviceErr); errUnmarshal != nil {
			return err
		}
		errWithStatus := serviceErr.(*dto.ServicesRequestErr)
		errWithStatus.StatusCode = APIResponse.StatusCode
		serviceErr = errWithStatus
		return err
	}

	return nil
}

// SendBatchTransaction ... Calls transaction-signers service with a batch transaction object to sign and send to chain
func SendBatchTransaction(httpClient *http.Client, cache *utility.MemoryCache, logger *utility.Logger, config Config.Data,
	requestData dto.BatchRequest, responseData *dto.SendTransactionResponse, serviceErr interface{}) error {
	authToken, err := GetAuthToken(cache, logger, config)
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("sendBatchTransaction", config)

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
	APIResponse, err := APIClient.Do(APIRequest, responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%+v", err)), serviceErr); errUnmarshal != nil {
			return err
		}
		errWithStatus := serviceErr.(*dto.ServicesRequestErr)
		errWithStatus.StatusCode = APIResponse.StatusCode
		serviceErr = errWithStatus
		return err
	}

	return nil

}
