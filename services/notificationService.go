package services

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	Config "wallet-adapter/config"

	"wallet-adapter/utility/apiClient"
	"wallet-adapter/utility/logger"

	"wallet-adapter/dto"
	"wallet-adapter/utility"
)

//NotificationService object
type NotificationService struct {
	Cache  *utility.MemoryCache
	Config Config.Data
	Error  *dto.ExternalServicesRequestErr
}

func NewNotificationService(cache *utility.MemoryCache, config Config.Data) *NotificationService {
	baseService := NotificationService{
		Cache:  cache,
		Config: config,
	}
	return &baseService
}

func (service *NotificationService) SendEmailNotification(cache *utility.MemoryCache, config Config.Data, requestData dto.SendEmailRequest, responseData *dto.SendEmailResponse, serviceErr interface{}) error {
	AuthService := NewAuthService(service.Cache, service.Config)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("sendEmail", config)

	APIClient := apiClient.New(nil, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
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

func (service *NotificationService) SendSmsNotification(cache *utility.MemoryCache, config Config.Data, requestData dto.SendSmsRequest, responseData *dto.SendSmsResponse, serviceErr interface{}) error {
	AuthService := NewAuthService(service.Cache, service.Config)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("sendSms", config)
	APIClient := apiClient.New(nil, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
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

func (service *NotificationService) BuildAndSendSms(assetSymbol string, amount *big.Float, cache *utility.MemoryCache, config Config.Data, serviceErr interface{}) {
	logger.Info("Sending sms notification for asset ", assetSymbol)
	formattedPhoneNumber := config.ColdWalletSmsNumber
	if !strings.HasPrefix(config.ColdWalletSmsNumber, "+") {
		formattedPhoneNumber = "+" + config.ColdWalletSmsNumber
	}
	sendSmsRequest := dto.SendSmsRequest{
		Message:     fmt.Sprintf("Please fund Bundle hot wallet address for %s with at least %f %s", assetSymbol, amount, assetSymbol),
		PhoneNumber: formattedPhoneNumber,
		SmsType:     utility.NOTIFICATION_SMS_TYPE,
		Country:     utility.NOTIFICATION_SMS_COUNTRY,
	}
	sendSmsResponse := dto.SendSmsResponse{}
	if err := service.SendSmsNotification(cache, config, sendSmsRequest, &sendSmsResponse, serviceErr); err != nil {
		logger.Error(fmt.Sprintf("error with sending sms notification for asset %s : %s", assetSymbol, err))
	}
}
