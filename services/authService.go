package services

import (
	"encoding/json"
	Config "wallet-adapter/config"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/go-redis/redis/v7"
)

// UpdateAuthToken ...
func UpdateAuthToken(logger *utility.Logger, config Config.Data) (model.UpdateAuthTokenResponse, error) {

	requestData := model.UpdateAuthTokenRequest{Body: model.AuthTokenRequestBody{
		ServiceID: config.ServiceID,
		Payload:   "",
	}}
	authToken := model.UpdateAuthTokenResponse{}

	marshaledRequest, _ := json.Marshal(requestData)

	if err := ExternalAPICall(marshaledRequest, "generateToken", authToken, config, logger); err != nil {
		return err
	}

	memorycache := utility.InitializeCache()
	if err := memorycache.Set("serviceAuth", &authToken, 0).Err(); err != nil {
		return err
	}

	return authToken, nil
}

// GetAuthToken ...
func GetAuthToken(logger *utility.Logger, config Config.Data) (model.UpdateAuthTokenResponse, error) {

	authToken, err := client.Get("serviceAuth").Result()
	if err != nil {
		logger.Error("Authentication token validation error : %s", err)
		return authToken, err
	}

	if authToken == "" {
		authToken, err := UpdateAuthToken(logger, config, client)
		if err != nil {
			logger.Error("Service auth token could not be retrieved, error : %s", err)
			return authToken, err
		}
		return authToken.Token, nil
	}

	tokenClaims := model.TokenClaims{}
	if err := utility.VerifyJWT(authToken, config, &tokenClaims); err != nil {

		logger.Error("Service auth error : %s", err)

		authToken, err := UpdateAuthToken(logger, config, client)
		if err != nil {
			logger.Error("Service auth token could not be retrieved, error : %s", err)
			return authToken, err
		}
		return authToken.Token, nil
	}
}
