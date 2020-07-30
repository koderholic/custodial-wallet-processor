package services

import (
	"encoding/json"
	"fmt"
	Config "wallet-adapter/config"
	"wallet-adapter/dto"
	"wallet-adapter/utility"
)

func SendEmailNotification(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, requestData dto.SendEmailRequest, responseData *dto.SendEmailResponse, serviceErr interface{}) error {

	authToken, err := GetAuthToken(cache, logger, config)
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("sendEmail", config)

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
			logger.Error("An error occured while calling notifications service %+v %+v ", err, err.Error())
			return err
		}
		logger.Error("An error occured while calling notifications service %+v %+v ", err, err.Error())
		return err
	}

	return nil
}

func SendSmsNotification(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, requestData dto.SendSmsRequest, responseData *dto.SendSmsResponse, serviceErr interface{}) error {
	authToken, err := GetAuthToken(cache, logger, config)
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("sendSms", config)
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
			logger.Error("An error occured while calling notifications service %+v %+v ", err, err.Error())
			return err
		}
		logger.Error("An error occured while calling notifications service %+v %+v ", err, err.Error())
		return err
	}

	return nil
}

func BuildAndSendSms(assetSymbol string, cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, serviceErr interface{}) {
	sendSmsRequest := dto.SendSmsRequest{
		Message:     "Please fund Bundle hot wallet address for " + assetSymbol,
		PhoneNumber: config.ColdWalletSmsNumber,
		SmsType:     utility.NOTIFICATION_SMS_TYPE,
		Country:     utility.NOTIFICATION_SMS_COUNTRY,
	}
	sendSmsResponse := dto.SendSmsResponse{}
	SendSmsNotification(cache, logger, config, sendSmsRequest, &sendSmsResponse, serviceErr)
}
