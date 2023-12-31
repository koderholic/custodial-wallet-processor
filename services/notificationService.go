package services

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
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

func BuildAndSendSms(assetSymbol, network string, amount *big.Float, cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, serviceErr interface{}) {
	logger.Info("Sending sms notification for asset ", assetSymbol)
	formattedPhoneNumber := config.ColdWalletSmsNumber
	if !strings.HasPrefix(config.ColdWalletSmsNumber, "+") {
		formattedPhoneNumber = "+" + config.ColdWalletSmsNumber
	}

	notificationEnv := ""
	if config.SENTRY_ENVIRONMENT == utility.ENV_PRODUCTION {
		notificationEnv = "LIVE"
	} else {
		notificationEnv = "TEST"
	}

	sendSmsRequest := dto.SendSmsRequest{
		Message:     fmt.Sprintf("%s - Please fund Bundle hot wallet address for %s - %s with at least %f %s", notificationEnv, assetSymbol, network, amount, assetSymbol),
		PhoneNumber: formattedPhoneNumber,
		SmsType:     utility.NOTIFICATION_SMS_TYPE,
		Country:     utility.NOTIFICATION_SMS_COUNTRY,
	}
	sendSmsResponse := dto.SendSmsResponse{}
	if err := SendSmsNotification(cache, logger, config, sendSmsRequest, &sendSmsResponse, serviceErr); err != nil {
		logger.Error(fmt.Sprintf("error with sending sms notification for asset %s - %s : %s", assetSymbol, network, err))
	}
}
