package services

import (
	"encoding/json"
	"fmt"
	"errors"
	uuid "github.com/satori/go.uuid"
	"github.com/trustwallet/blockatlas/pkg/logger"
	"wallet-adapter/dto"
	"wallet-adapter/utility"
)

func (service BaseService) GenerateUserAddressOnBBS(userId uuid.UUID, assetSymbol string, network string) (dto.GetUserAddressResponse, error) {
	var serviceErr dto.ServicesRequestErr
	authToken, err := GetAuthToken(service.Cache, service.Logger, service.Config)
	if err != nil {
		return dto.GetUserAddressResponse{}, err
	}

	responseData := dto.GetUserAddressResponse{}
	metaData := utility.GetRequestMetaData("getUserAddressBBS", service.Config)

	APIClient := NewClient(nil, service.Logger, service.Config, fmt.Sprintf("%s%s/%s/assets/deposit-address?coin=%s&network=%s",
		metaData.Endpoint, metaData.Action,userId, assetSymbol, network))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", nil)
	if err != nil {
		return dto.GetUserAddressResponse{}, err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})

	_, err = APIClient.Do(APIRequest, &responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), &serviceErr); errUnmarshal != nil {
			logger.Error("An error occurred while calling binance brokerage service %+v %+v ", err, err.Error())
			return dto.GetUserAddressResponse{}, err
		}
		logger.Error("An error occurred while calling binance brokerage service %+v %+v ", err, err.Error())
		return dto.GetUserAddressResponse{}, errors.New(serviceErr.Message)
	}

	logger.Info("Response from GenerateUserAddressOnBBS : %+v", responseData)
	return responseData, nil
}