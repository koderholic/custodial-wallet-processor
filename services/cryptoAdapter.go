package services

import (
	"encoding/json"
	"fmt"
	Config "wallet-adapter/config"
	"wallet-adapter/model"
	"wallet-adapter/utility"
)

// broadcastToChain ... Calls crypto adapter with signed transaction to be broadcast to chain
func BroadcastToChain(logger *utility.Logger, config Config.Data, requestData model.BroadcastToChainRequest, responseData *model.BroadcastToChainResponse, serviceErr interface{}) error {

	authToken, err := GetAuthToken(logger, config)
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
	_, err = APIClient.Do(APIRequest, responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), serviceErr); errUnmarshal != nil {
			return err
		}
		return err
	}

	return nil
}
