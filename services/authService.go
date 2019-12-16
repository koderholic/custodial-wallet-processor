package services

import (
	"time"
	Config "wallet-adapter/config"
	"wallet-adapter/model"
	"wallet-adapter/utility"
)

// UpdateAuthToken ...
func UpdateAuthToken(logger *utility.Logger, config Config.Data) (model.UpdateAuthTokenResponse, error) {

	requestData := model.UpdateAuthTokenRequest{Body: model.AuthTokenRequestBody{
		ServiceID: config.ServiceID,
		Payload:   "",
	}}
	authToken := model.UpdateAuthTokenResponse{}
	metaData := utility.GetRequestMetaData("generateToken", config)

	APIClient := NewClient(nil, logger, config, metaData.Endpoint)
	APIRequest, err := APIClient.NewRequest(metaData.Type, metaData.Action, requestData)
	if err != nil {
		return model.UpdateAuthTokenResponse{}, err
	}
	_, err = APIClient.Do(APIRequest, &authToken)
	if err != nil {
		return model.UpdateAuthTokenResponse{}, err
	}

	purgeInterval := config.PurgeCacheInterval * time.Second
	memorycache := utility.InitializeCache(authToken.ExpiresAt, purgeInterval)
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
