package test

import (
	"fmt"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"wallet-adapter/utility"
	Config "wallet-adapter/config"
	validation "gopkg.in/go-playground/validator.v9"
	"wallet-adapter/database"
)

type Test struct {
	pingEndpoint string
	CreateAssetEndpoint string
}

var test = Test{
	pingEndpoint: "/api/v1ping",
	CreateAssetEndpoint: "/api/v1/crypto/users/assets",
}

//BaseController : Base controller struct
type Controller struct {
	Logger    *utility.Logger
	Config    Config.Data
	Validator *validation.Validate
	Repository database.IRepository
}

func TestRegisterRoutes(t *testing.T) {

	pingRequest, _ := http.NewRequest("GET", test.pingEndpoint, bytes.NewBuffer([]byte("")))

	pingResponse := fireRequest(pingRequest)
	if pingResponse.Code == http.StatusNotFound {
		t.Errorf("Expected response code to not be %d. Got %d\n", http.StatusNotFound, pingResponse.Code)
	}
}

func TestCreateUserAsset(t *testing.T) {
	createAssetInputData := []byte(`{"assets" : ["BTC","BND"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJtaXNzaW9ucyI6WyJzcnZjLndhbGxldC1hZGFwdGVyLnBvc3QtYXNzZXRzIl0sInRva2VuVHlwZSI6IlNFUlZJQ0UifQ.yIx-wr2HNzn8z9mxuiXJ3oZpVyLRRzZWm7IlcmKEVoic9p7qsoy9kvNUnmZqfz1gNLRJYUEd5FkypLEUMzaF3rURG2OBjKx1T341DnsnBWwf89qX8ENKam3WXqZVGpXRqcpgJLfCKnmyQJm-cRTJaiI-MCFkvojqzT0njumfhgHSpdA2ZeGOFu6djeOpFUqi1KzGkwWS2cnU07zRnfSU0CWXokDVabOZ-xlhzhdqZVlUOC-YnFXfGURQ0fTGz4YwHmWcQTJ1f770zVKOb-LyVzx_rg3akkhn6150bbLr17_JaG2F6aXyr12P70TGy1Xw-dzO5Rl-IfQs0BBvecwKXg")

	createAssetResponse := fireRequest(createAssetRequest)
	fmt.Printf("createAssetResponse >> %s", createAssetResponse.Body.String())

	if createAssetResponse.Code != http.StatusOK {
		t.Errorf("Expected response code to not be %d. Got %d\n", http.StatusOK, createAssetResponse.Code)
	}


}

func fireRequest(request *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, request)

	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}
