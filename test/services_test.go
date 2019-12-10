package test

import (
	"testing"
	"wallet-adapter/services"
)

func TestGetTokenReturnsEmptyAtInitialization(t *testing.T) {

	authToken, _ := services.GetAuthToken(logger, config)

	if authToken != "" {
		t.Errorf("Expected item fetched to not be empty, got %s\n", authToken)
	}

	if authToken != "" {
		t.Errorf("Expected item fetched to not be empty, got %s\n", authToken)
	}

}
