package services

import (
	"encoding/json"
	"fmt"
	Config "wallet-adapter/config"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	uuid "github.com/satori/go.uuid"
)

// GenerateAddress ...
func GenerateAddress(logger *utility.Logger, config Config.Data, userID uuid.UUID, symbol string, serviceErr interface{}) (string, error) {

	authToken, err := GetAuthToken(logger, config)
	if err != nil {
		return "", err
	}
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
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), serviceErr); errUnmarshal != nil {
			return "", err
		}
		return "", err
	}

	logger.Info("Response from GenerateAddress : %+v", responseData)
	return responseData.Address, nil
}
