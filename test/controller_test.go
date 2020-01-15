package test

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
	"time"
	Config "wallet-adapter/config"
	config "wallet-adapter/config"
	"wallet-adapter/controllers"
	"wallet-adapter/database"
	"wallet-adapter/middlewares"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/DATA-DOG/go-sqlmock"

	httpSwagger "github.com/swaggo/http-swagger"
	validation "gopkg.in/go-playground/validator.v9"
)

type Test struct {
	pingEndpoint        string
	CreateAssetEndpoint string
	GetAssetEndpoint    string
	CreditAssetEndpoint string
}

var test = Test{
	pingEndpoint:        "/ping",
	CreateAssetEndpoint: "/crypto/users/assets",
	GetAssetEndpoint:    "/crypto/users/a10fce7b-7844-43af-9ed1-e130723a1ea3/assets",
	CreditAssetEndpoint: "/crypto/assets/credit",
}

//BaseController : Base controller struct
type Controller struct {
	Logger     *utility.Logger
	Config     Config.Data
	Validator  *validation.Validate
	Repository database.IRepository
}

//Suite ...
type Suite struct {
	suite.Suite
	DB         *gorm.DB
	Mock       sqlmock.Sqlmock
	Database   database.Database
	Logger     *utility.Logger
	Config     config.Data
	Middleware http.Handler
}

var (
	once sync.Once
)

func TestInit(t *testing.T) {
	suite.Run(t, new(Suite))
}

// SetupSuite ...
func (s *Suite) SetupSuite() {

	var (
		db  *sql.DB
		err error
	)

	db, s.Mock, err = sqlmock.New()
	require.NoError(s.T(), err)
	s.DB, err = gorm.Open("mysql", db)
	require.NoError(s.T(), err)
	s.DB.LogMode(true)

	logger := utility.NewLogger()
	router := mux.NewRouter()
	validator := validation.New()
	Config := config.Data{
		AppPort:            "9000",
		ServiceName:        "crypto-wallet-adapter",
		AuthenticatorKey:   "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE0ZjV3ZzVsMmhLc1RlTmVtL1Y0MQpmR25KbTZnT2Ryajh5bTNyRmtFVS93VDhSRHRuU2dGRVpPUXBIRWdRN0pMMzh4VWZVMFkzZzZhWXc5UVQwaEo3Cm1DcHo5RXI1cUxhTVhKd1p4ekh6QWFobGZBMGljcWFidkpPTXZRdHpENnVRdjZ3UEV5WnREVFdpUWk5QVh3QnAKSHNzUG5wWUdJbjIwWlp1TmxYMkJyQ2xjaUhoQ1BVSUlaT1FuL01tcVREMzFqU3lqb1FvVjdNaGhNVEFUS0p4MgpYckhoUisxRGNLSnpRQlNUQUducFlWYXFwc0FSYXArbndSaXByM25VVHV4eUdvaEJUU21qSjJ1c1NlUVhISTNiCk9ESVJlMUF1VHlIY2VBYmV3bjhiNDYyeUVXS0FSZHBkOUFqUVc1U0lWUGZkc3o1QjZHbFlRNUxkWUt0em5UdXkKN3dJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0t",
		PurgeCacheInterval: 5,
	}

	Database := database.Database{
		Logger: logger,
		Config: Config,
		DB:     s.DB,
	}
	middleware := middlewares.NewMiddleware(logger, Config, router).ValidateAuthToken().LogAPIRequests().Build()

	s.Database = Database
	s.Logger = logger
	s.Config = Config
	s.Middleware = middleware

	s.RegisterRoutes(router, validator)
}

// RegisterRoutes ...
func (s *Suite) RegisterRoutes(router *mux.Router, validator *validation.Validate) {

	once.Do(func() {

		baseRepository := database.BaseRepository{Database: s.Database}
		userAssetRepository := database.UserAssetRepository{BaseRepository: baseRepository}

		// controller := controllers.NewController(s.Logger, s.Config, validator, &baseRepository)
		userAssetController := controllers.NewUserAssetController(s.Logger, s.Config, validator, &userAssetRepository)

		apiRouter := router.PathPrefix("").Subrouter()
		router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		// User Asset Routes
		apiRouter.HandleFunc("/crypto/users/assets", userAssetController.CreateUserAssets).Methods(http.MethodPost)
		apiRouter.HandleFunc("/crypto/users/{userId}/assets", userAssetController.GetUserAssets).Methods(http.MethodGet)
		apiRouter.HandleFunc("/crypto/assets/credit", userAssetController.CreditUserAssets).Methods(http.MethodPost)

	})
}

func (s *Suite) Test_CreditUserAsset() {

	s.Mock.ExpectQuery(regexp.QuoteMeta(
		fmt.Sprintf("SELECT denominations.symbol, denominations.decimal,user_balances.* FROM `user_balances`"))).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "asset_id", "available_balance", "reserved_balance", "symbol", "decimal"}).
			AddRow("60ed6eb5-41f9-482c-82e5-78abce7c142e", time.Now(), time.Now(), nil, "a10fce7b-7844-43af-9ed1-e130723a1ea3", "0c9f0ffe-169d-463e-b77f-bc36a8704db4", 0, 0, "BTC", 8),
		)
	s.Mock.ExpectBegin()
	// s.Mock.ExpectExec(regexp.QuoteMeta("UPDATE `user_balances` SET `available_balance` = '7.2e+08', `reserved_balance` = '7.2e+08', `updated_at` = '2020-01-14 22:06:05'  WHERE `user_balances`.`deleted_at` IS NULL AND `user_balances`.`id` = '1f6c2d1e-31ec-4b68-93c5-626ef9bee3d0'"))
	s.Mock.ExpectExec("UPDATE user_balances").WillReturnResult(sqlmock.NewResult(1, 1))
	s.Mock.ExpectCommit()

	creditAssetInputData := []byte(`{"assetId" : "1f6c2d1e-31ec-4b68-93c5-626ef9bee3d0","value" : "0.9","transactionReference" : "75622fab2dd51bec0779","memo" : "Test credit transaction"}`)
	creditAssetRequest, _ := http.NewRequest("POST", test.CreditAssetEndpoint, bytes.NewBuffer(creditAssetInputData))
	creditAssetRequest.Header.Set("x-auth-token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJtaXNzaW9ucyI6WyJzdmNzLmNyeXB0by13YWxsZXQtYWRhcHRlci5wb3N0LWNyZWRpdCJdLCJ0b2tlblR5cGUiOiJTRVJWSUNFIn0.a-Hh5C9yb4BN6qfLQJyar5GnG3qmCn2z2t2VwAovEFx3atelrjVZv70hR-sgicjWrQ4YJgr5GCIglWpsh3dlISljoB2OAwKqicTo5HPD_97Z3EmD0jGyCoWbr0kgc22llrg9ihGI9F3wLnhf9LRLDLeuVRQNj3FQQM1uVhtECbzchVaJLWb-AUtDJQMYT1C1nTZfFv5-0Uq0yyAK9fAPJaPjV2eTagWlkbEyXVVbdxcGSHqcuvQTKJs3NrxS60k6glWhw5S9HMX2HgZPMhcLDCziElrFt3Xqx3y0jGeEY0ldxCMRP4aH0Kp6krso6Jt6vYx3Ky-RXKGJniPp_6fXiw")

	creditAssetResponse := httptest.NewRecorder()
	s.Middleware.ServeHTTP(creditAssetResponse, creditAssetRequest)

	if creditAssetResponse.Code != http.StatusCreated {
		s.T().Errorf("Expected response code to not be %d. Got %d\n", http.StatusCreated, creditAssetResponse.Code)
	}
}
func (s *Suite) Test_GetUserAsset() {
	// s.Mock.ExpectQuery(regexp.QuoteMeta(
	// 	fmt.Sprintf("SELECT denominations.symbol, denominations.decimal,user_balances.* FROM `user_balances`"))).
	// 	WithArgs(sqlmock.AnyArg()).
	// 	WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "asset_id", "available_balance", "reserved_balance", "symbol", "decimal"}).
	// 		AddRow("60ed6eb5-41f9-482c-82e5-78abce7c142e", time.Now(), time.Now(), nil, "a10fce7b-7844-43af-9ed1-e130723a1ea3", "0c9f0ffe-169d-463e-b77f-bc36a8704db4", 0, 0, "BTC", 8),
	// 	)
	s.Mock.ExpectQuery(regexp.QuoteMeta("INSERT  INTO `user_balances`")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "denomination_id", "available_balance", "reserved_balance", "symbol"}).
			AddRow("60ed6eb5-41f9-482c-82e5-78abce7c142e", time.Now(), time.Now(), nil, "a10fce7b-7844-43af-9ed1-e130723a1ea3", "0c9f0ffe-169d-463e-b77f-bc36a8704db4", 0, 0, "BTC"),
		)
	getAssetRequest, _ := http.NewRequest("GET", test.GetAssetEndpoint, bytes.NewBuffer([]byte("")))
	getAssetRequest.Header.Set("x-auth-token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJtaXNzaW9ucyI6WyJzdmNzLmNyeXB0by13YWxsZXQtYWRhcHRlci5nZXQtYXNzZXRzIl0sInRva2VuVHlwZSI6IlNFUlZJQ0UifQ.vMOgtyLiKeYN7oLSi8FVOi87ydHMwqVhUtPGxV16HIbdUnRd1fUS0KlEjHvfGZ6EVMXqGJsgOLv_05fVtrtBAR54QgXKejItR_zNhSah3lxhN4S4ZCAjlmw_J6trByBY5H1dSSLvZHNjSJD2NXx5_8SDXWoBZBauIq0_jAuVF171PEDJVdoYB7ZFkeiQfF4WguwOcGPPRnW0qtpHBS7apx9jXF9eFzm8kpDe-h4hjd-BcMX0FCdaR00K1YZ-fSLtyOdj55JKEUoop4xevJnWEZE-3sMWi2GzAl1advFha84hbE0eHEKkky9Dal_H5Awpwpv7kqHqj0Melf-zvW1HEg")

	getAssetResponse := httptest.NewRecorder()
	s.Middleware.ServeHTTP(getAssetResponse, getAssetRequest)

	if getAssetResponse.Code != http.StatusOK {
		s.T().Errorf("Expected response code to not be %d. Got %d\n", http.StatusOK, getAssetResponse.Code)
	}
}

func (s *Suite) Test_CreateUserAsset() {

	s.Mock.ExpectQuery(regexp.QuoteMeta(
		fmt.Sprintf("SELECT denominations.symbol, denominations.decimal,user_balances.* FROM `user_balances`"))).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "asset_id", "available_balance", "reserved_balance", "symbol", "decimal"}).
			AddRow("60ed6eb5-41f9-482c-82e5-78abce7c142e", time.Now(), time.Now(), nil, "a10fce7b-7844-43af-9ed1-e130723a1ea3", "0c9f0ffe-169d-463e-b77f-bc36a8704db4", 0, 0, "BTC", 8),
		)
	s.Mock.ExpectQuery(regexp.QuoteMeta("INSERT  INTO `user_balances`")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "denomination_id", "available_balance", "reserved_balance", "symbol"}).
			AddRow("60ed6eb5-41f9-482c-82e5-78abce7c142e", time.Now(), time.Now(), nil, "a10fce7b-7844-43af-9ed1-e130723a1ea3", "0c9f0ffe-169d-463e-b77f-bc36a8704db4", 0, 0, "BTC"),
		)

	createAssetInputData := []byte(`{"assets" : ["BTC"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJwZXJtaXNzaW9ucyI6WyJzdmNzLmNyeXB0by13YWxsZXQtYWRhcHRlci5wb3N0LWFzc2V0cyJdLCJ0b2tlblR5cGUiOiJTRVJWSUNFIn0.F4ONI4_z5YMihzXRxq2apTt242JyaAIx98XugeQsGYJ9QQ1aqf65Nv0186b76fPDHI4bF_WoC3tj9khWFcuhH8zwMFC4ohVQ1NLSMrCUBk19pCamhV-lr5znLbcNuWJ9Nhf9Z9R-S4_HOOizKJu1ydAwYdMI5Z_MPwaHooGqH5FAkwH9DTB_k08MzlMcnKWkzVC7kOKS5fnbqPCIrYYLhygTQxRcmeXFhgScsh54TzNfI5334WCCWoKBppCrvl_vyPsWciEt1wQUu_29hDrNJFf_3sqf9ooROkyQhf6G0p7Sh3nHZhKBATN7g3X-xD1KTJ99aZ0khZMPEyOHbJb3Zg")

	createAssetResponse := httptest.NewRecorder()
	s.Middleware.ServeHTTP(createAssetResponse, createAssetRequest)

	if createAssetResponse.Code != http.StatusCreated {
		s.T().Errorf("Expected response code to not be %d. Got %d\n", http.StatusCreated, createAssetResponse.Code)
	}
}
