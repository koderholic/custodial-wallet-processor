package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	Config "wallet-adapter/config"

	"wallet-adapter/utility/apiClient"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/constants"
	"wallet-adapter/utility/logger"

	"wallet-adapter/database"
	"wallet-adapter/dto"
)

//NotificationService object
type NotificationService struct {
	Cache      *cache.Memory
	Config     Config.Data
	Error      *dto.ExternalServicesRequestErr
	Repository database.IRepository
}

func NewNotificationService(cache *cache.Memory, config Config.Data, repository database.IRepository) *NotificationService {
	baseService := NotificationService{
		Cache:      cache,
		Config:     config,
		Repository: repository,
		Error:      &dto.ExternalServicesRequestErr{},
	}
	return &baseService
}

func (service *NotificationService) SendEmailNotification(requestData dto.SendEmailRequest, responseData *dto.SendEmailResponse) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("sendEmail", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	if err := APIClient.Do(APIRequest, responseData); err != nil {
		appErr := err.(appError.Err)
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%s", err.Error())), service.Error); errUnmarshal != nil {
			return err
		}
		return serviceError(appErr.ErrCode, service.Error.Code, errors.New(service.Error.Message))
	}

	return nil
}

func (service *NotificationService) SendSmsNotification(requestData dto.SendSmsRequest, responseData *dto.SendSmsResponse) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("sendSms", service.Config)
	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	if err := APIClient.Do(APIRequest, responseData); err != nil {
		appErr := err.(appError.Err)
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%s", err.Error())), service.Error); errUnmarshal != nil {
			return err
		}
		return serviceError(appErr.ErrCode, service.Error.Code, errors.New(service.Error.Message))
	}

	return nil
}

func (service *NotificationService) BuildAndSendSms(assetSymbol string, amount *big.Float) {
	logger.Info("Sending sms notification for asset ", assetSymbol)
	formattedPhoneNumber := service.Config.ColdWalletSmsNumber
	if !strings.HasPrefix(service.Config.ColdWalletSmsNumber, "+") {
		formattedPhoneNumber = "+" + service.Config.ColdWalletSmsNumber
	}

	notificationEnv := ""
	if service.Config.SENTRY_ENVIRONMENT == constants.ENV_PRODUCTION {
		notificationEnv = "LIVE"
	} else {
		notificationEnv = "TEST"
	}

	sendSmsRequest := dto.SendSmsRequest{
		Message:     fmt.Sprintf("%s - Please fund Bundle hot wallet address for %s with at least %f %s", notificationEnv, assetSymbol, amount, assetSymbol),
		PhoneNumber: formattedPhoneNumber,
		SmsType:     constants.NOTIFICATION_SMS_TYPE,
		Country:     constants.NOTIFICATION_SMS_COUNTRY,
	}
	sendSmsResponse := dto.SendSmsResponse{}
	if err := service.SendSmsNotification(sendSmsRequest, &sendSmsResponse); err != nil {
		logger.Error(fmt.Sprintf("error with sending sms notification for asset %s : %s", assetSymbol, err))
	}
}
