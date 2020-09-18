package services

import (
	"fmt"
	Config "wallet-adapter/config"
	"wallet-adapter/dto"
	"wallet-adapter/utility"
	"wallet-adapter/utility/apiClient"
	"wallet-adapter/utility/logger"
)

//AuthService object
type AuthService struct {
	Cache  *utility.MemoryCache
	Config Config.Data
	Error  *dto.ExternalServicesRequestErr
}

func NewAuthService(cache *utility.MemoryCache, config Config.Data) *AuthService {
	baseService := AuthService{
		Cache:  cache,
		Config: config,
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
	metaData := utility.GetRequestMetaData("generateToken", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", nil)
	if err != nil {
		return dto.UpdateAuthTokenResponse{}, err
	}
	APIClient.AddBasicAuth(APIRequest, authorization["username"], authorization["password"])
	_, err = APIClient.Do(APIRequest, &authToken)
	if err != nil {
		return dto.UpdateAuthTokenResponse{}, err
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
		return authTokenResponse.Token, err
	}

	return authToken, nil
}
