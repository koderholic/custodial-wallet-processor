package test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/utility"

	"github.com/stretchr/testify/suite"

	"github.com/DATA-DOG/go-sqlmock"

	validation "gopkg.in/go-playground/validator.v9"
)

type Test struct {
	pingEndpoint        string
	CreateAssetEndpoint string
	GetAssetEndpoint    string
}

var test = Test{
	pingEndpoint:        "/api/v1ping",
	CreateAssetEndpoint: "/api/v1/crypto/users/assets",
	GetAssetEndpoint:    "/api/v1/crypto/users/a10fce7b-7844-43af-9ed1-e130723a1ea3/assets",
}

//BaseController : Base controller struct
type Controller struct {
	Logger     *utility.Logger
	Config     Config.Data
	Validator  *validation.Validate
	Repository database.IRepository
}

func TestInit(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) Test_GetUserAsset() {
	s.Mock.ExpectQuery(regexp.QuoteMeta(
		fmt.Sprintf("SELECT assets.symbol,user_balances.* FROM `user_balances`"))).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "asset_id", "available_balance", "reserved_balance", "symbol"}).
			AddRow("60ed6eb5-41f9-482c-82e5-78abce7c142e", time.Now(), time.Now(), nil, "a10fce7b-7844-43af-9ed1-e130723a1ea3", "0c9f0ffe-169d-463e-b77f-bc36a8704db4", 0, 0, "BTC"),
		)
	getAssetRequest, _ := http.NewRequest("GET", test.GetAssetEndpoint, bytes.NewBuffer([]byte("")))
	getAssetRequest.Header.Set("x-auth-token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJtaXNzaW9ucyI6WyJzcnZjLndhbGxldC1hZGFwdGVyLmdldC1hc3NldHMiXSwidG9rZW5UeXBlIjoiU0VSVklDRSJ9.Dk0_p1YmYkxXunVD4AZIeTyozzxwcVJ9eUvp7tsVh3kZGCMNMtTrgNA28zSL1cPQ_e5B7J_VcgS47twS-A0Gl5vJmlsebtMbea5CO3RzukEU99vMZnL5aGXNivsh1OHfnBFi3ZNDxIu0tLIcjlVQEGrWMZoBxvUBfSr_ffi59XclkyyoIbUyqsISZaYMJE1XDDgYQ33hg4y-jFBvau5R2KnKwXho7yFt7RvLaKhW1cGEuJYDZj1_grDNp8hR1Sb0xOjDHAlppO8T0p2bcYNf1W9K0W09zFoudInpqpJTmoPjjjZCQ7miPp6NPA342bqlMuR3pZndQhxL0ZGcNL8Sgg")

	getAssetResponse := httptest.NewRecorder()
	s.Middleware.ServeHTTP(getAssetResponse, getAssetRequest)

	if getAssetResponse.Code != http.StatusOK {
		s.T().Errorf("Expected response code to not be %d. Got %d\n", http.StatusOK, getAssetResponse.Code)
	}
}

func (s *Suite) Test_CreateUserAsset() {

	s.Mock.ExpectQuery(regexp.QuoteMeta(
		fmt.Sprintf("SELECT assets.symbol,user_balances.* FROM `user_balances`"))).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "asset_id", "available_balance", "reserved_balance", "symbol"}).
			AddRow("60ed6eb5-41f9-482c-82e5-78abce7c142e", time.Now(), time.Now(), nil, "a10fce7b-7844-43af-9ed1-e130723a1ea3", "0c9f0ffe-169d-463e-b77f-bc36a8704db4", 0, 0, "BTC"),
		)
	s.Mock.ExpectQuery(regexp.QuoteMeta("INSERT  INTO `user_balances`")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "asset_id", "available_balance", "reserved_balance", "symbol"}).
			AddRow("60ed6eb5-41f9-482c-82e5-78abce7c142e", time.Now(), time.Now(), nil, "a10fce7b-7844-43af-9ed1-e130723a1ea3", "0c9f0ffe-169d-463e-b77f-bc36a8704db4", 0, 0, "BTC"),
		)

	createAssetInputData := []byte(`{"assets" : ["BTC"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJtaXNzaW9ucyI6WyJzcnZjLndhbGxldC1hZGFwdGVyLnBvc3QtYXNzZXRzIl0sInRva2VuVHlwZSI6IlNFUlZJQ0UifQ.yIx-wr2HNzn8z9mxuiXJ3oZpVyLRRzZWm7IlcmKEVoic9p7qsoy9kvNUnmZqfz1gNLRJYUEd5FkypLEUMzaF3rURG2OBjKx1T341DnsnBWwf89qX8ENKam3WXqZVGpXRqcpgJLfCKnmyQJm-cRTJaiI-MCFkvojqzT0njumfhgHSpdA2ZeGOFu6djeOpFUqi1KzGkwWS2cnU07zRnfSU0CWXokDVabOZ-xlhzhdqZVlUOC-YnFXfGURQ0fTGz4YwHmWcQTJ1f770zVKOb-LyVzx_rg3akkhn6150bbLr17_JaG2F6aXyr12P70TGy1Xw-dzO5Rl-IfQs0BBvecwKXg")

	createAssetResponse := httptest.NewRecorder()
	s.Middleware.ServeHTTP(createAssetResponse, createAssetRequest)

	if createAssetResponse.Code != http.StatusCreated {
		s.T().Errorf("Expected response code to not be %d. Got %d\n", http.StatusCreated, createAssetResponse.Code)
	}
}
