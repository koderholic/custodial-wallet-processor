package services

import (
	"errors"
	"fmt"
	Config "wallet-adapter/config"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	uuid "github.com/satori/go.uuid"
)

// GenerateAddress ...
func GenerateAddress(logger *utility.Logger, config Config.Data, userID uuid.UUID, symbol string) (string, error) {

	authToken, err := GetAuthToken(logger, config)
	requestData := model.GenerateAddressRequest{}
	responseData := model.GenerateAddressResponse{}
	metaData := utility.GetRequestMetaData("createAddress", config)

	requestData.UserID = userID
	requestData.Symbol = symbol

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
		fmt.Printf("Error response from GenerateAddress : %+v", err)
		return "", errors.New("An error occured while calling create address endpoint of key-management service")
	}

	logger.Info("Response from GenerateAddress : %+v", responseData)
	if !responseData.Success {
		return "", errors.New(responseData.Message)
	}

	return responseData.Data["address"], nil
}
