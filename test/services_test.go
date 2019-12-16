package test

import (
	// "testing"
	"wallet-adapter/services"
)

func (s *Suite) Test_GetTokenReturnsEmptyAtInitialization() {

	authToken, _ := services.GetAuthToken(s.Logger, s.Config)

	if authToken != "" {
		s.T().Errorf("Expected item fetched to not be empty, got %s\n", authToken)
	}

	if authToken != "" {
		s.T().Errorf("Expected item fetched to not be empty, got %s\n", authToken)
	}

}
