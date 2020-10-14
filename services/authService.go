package services

import (
	"encoding/json"
	"errors"
	"fmt"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/utility/apiClient"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/logger"
)

//AuthService object
type AuthService struct {
	Cache      *cache.Memory
	Config     Config.Data
	Error      *dto.ExternalServicesRequestErr
	Repository database.IRepository
}

func NewAuthService(cache *cache.Memory, config Config.Data, repository database.IRepository) *AuthService {
	baseService := AuthService{
		Cache:      cache,
		Config:     config,
		Repository: repository,
	}
	return &baseService
}

// UpdateAuthToken ...
func (service *AuthService) UpdateAuthToken() (dto.UpdateAuthTokenResponse, error) {

	authorization := map[string]string{
		"username": service.Config.ServiceID,
		"password": service.Config.ServiceKey,
	}
	authToken := dto.UpdateAuthTokenResponse{}
	metaData := GetRequestMetaData("generateToken", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", nil)
	if err != nil {
		return dto.UpdateAuthTokenResponse{}, err
	}
	APIClient.AddBasicAuth(APIRequest, authorization["username"], authorization["password"])
	if err := APIClient.Do(APIRequest, &authToken); err != nil {
		appErr := err.(appError.Err)
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%s", err.Error())), service.Error); errUnmarshal != nil {
			return dto.UpdateAuthTokenResponse{}, err
		}
		return dto.UpdateAuthTokenResponse{}, serviceError(appErr.ErrCode, service.Error.Code, errors.New(service.Error.Message))
	}

	service.Cache.Set("serviceAuth", &authToken, true)

	return authToken, nil
}

// GetAuthToken ...
func (service *AuthService) GetAuthToken() (string, error) {

	cachedResult := service.Cache.Get("serviceAuth")

	if cachedResult == nil {
		authTokenResponse, err := service.UpdateAuthToken()
		if err != nil {
			logger.Error("Service auth token could not be retrieved, error : %s", err)
			return authTokenResponse.Token, err
		}
		return authTokenResponse.Token, err

	}
	authTokenResponse := cachedResult.(*dto.UpdateAuthTokenResponse)
	authToken := authTokenResponse.Token

	if authToken == "" {
		authTokenResponse, err := service.UpdateAuthToken()
		if err != nil {
			logger.Error("Service auth token could not be retrieved, error : %s", err)
			return authTokenResponse.Token, err
		}
		return authTokenResponse.Token, nil
	}

	return authToken, nil
}
