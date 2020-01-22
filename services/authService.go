package services

import (
	"fmt"
	"time"
	Config "wallet-adapter/config"
	"wallet-adapter/model"
	"wallet-adapter/utility"
)

// UpdateAuthToken ...
func UpdateAuthToken(logger *utility.Logger, config Config.Data) (model.UpdateAuthTokenResponse, error) {

	authorization := map[string]string{
		"username": config.ServiceID,
		"password": config.ServiceKey,
	}
	authToken := model.UpdateAuthTokenResponse{}
	metaData := utility.GetRequestMetaData("generateToken", config)

	APIClient := NewClient(nil, logger, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", nil)
	if err != nil {
		return model.UpdateAuthTokenResponse{}, err
	}
	APIClient.AddBasicAuth(APIRequest, authorization["username"], authorization["password"])
	_, err = APIClient.Do(APIRequest, &authToken)
	if err != nil {
		return model.UpdateAuthTokenResponse{}, err
	}
	logger.Info("Response from authentication service to UpdateAuthToken : %+v", authToken)

	purgeInterval := config.PurgeCacheInterval * time.Second
	createdAt, _ := time.Parse(time.RFC3339, authToken.CreatedAt)
	expiresAt, _ := time.Parse(time.RFC3339, authToken.ExpiresAt)
	cacheDuration := expiresAt.Sub(createdAt)
	memorycache := utility.InitializeCache(cacheDuration, purgeInterval)
	memorycache.Set("serviceAuth", &authToken, true)

	return authToken, nil
}

// GetAuthToken ...
func GetAuthToken(logger *utility.Logger, config Config.Data) (string, error) {

	memorycache := utility.InitializeCache(0, 0)
	cachedResult := memorycache.Get("serviceAuth")

	if cachedResult == nil {
		authTokenResponse, err := UpdateAuthToken(logger, config)
		if err != nil {
			logger.Error("Service auth token could not be retrieved, error : %s", err)
			return authTokenResponse.Token, err
		}
		return authTokenResponse.Token, err

	}
	authTokenResponse := cachedResult.(*model.UpdateAuthTokenResponse)
	authToken := authTokenResponse.Token

	if authToken == "" {
		authTokenResponse, err := UpdateAuthToken(logger, config)
		if err != nil {
			logger.Error("Service auth token could not be retrieved, error : %s", err)
			return authTokenResponse.Token, err
		}
		return authTokenResponse.Token, err
	}

	return authToken, nil
}
