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

func SubscribeAddress(logger *utility.Logger, config Config.Data, requestData model.SubscriptionRequest, responseData *model.SubscriptionResponse, serviceErr interface{}) error {
	metaData := utility.GetRequestMetaData("subscribeAddress", config)
	APIClient := NewClient(nil, logger, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
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
func TransactionStatus(logger *utility.Logger, config Config.Data, requestData model.TransactionStatusRequest, responseData *model.TransactionStatusResponse, serviceErr interface{}) error {

	authToken, err := GetAuthToken(logger, config)
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("transactionStatus", config)

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

// GetOnchainBalance ... Calls crypto adapter with asset symbol and address to return balance of asset on-chain
func GetOnchainBalance(logger *utility.Logger, config Config.Data, requestData model.OnchainBalanceRequest, responseData *model.OnchainBalanceResponse, serviceErr interface{}) error {

	authToken, err := GetAuthToken(logger, config)
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("getOnchainBalance", config)

	APIClient := NewClient(nil, logger, config, fmt.Sprintf("%s%s?address=%s&assetSymbol=%s", metaData.Endpoint, metaData.Action, requestData.Address, requestData.AssetSymbol ))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", nil)
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
