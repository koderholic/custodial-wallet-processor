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
	DebitAssetEndpoint  string
}

var test = Test{
	pingEndpoint:        "/ping",
	CreateAssetEndpoint: "/crypto/users/assets",
	GetAssetEndpoint:    "/crypto/users/a10fce7b-7844-43af-9ed1-e130723a1ea3/assets",
	CreditAssetEndpoint: "/crypto/assets/credit",
	DebitAssetEndpoint:  "/crypto/assets/debit",
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
	DB       *gorm.DB
	Mock     sqlmock.Sqlmock
	Database database.Database
	Logger   *utility.Logger
	Config   config.Data
	Router   *mux.Router
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
	// middleware := middlewares.NewMiddleware(logger, Config, router).ValidateAuthToken().LogAPIRequests().Build()

	s.Database = Database
	s.Logger = logger
	s.Config = Config
	s.Router = router

	s.RegisterRoutes(logger, Config, router, validator)
}

// RegisterRoutes ...
func (s *Suite) RegisterRoutes(logger *utility.Logger, Config config.Data, router *mux.Router, validator *validation.Validate) {

	once.Do(func() {

		baseRepository := database.BaseRepository{Database: s.Database}
		userAssetRepository := database.UserAssetRepository{BaseRepository: baseRepository}

		// controller := controllers.NewController(s.Logger, s.Config, validator, &baseRepository)
		userAssetController := controllers.NewUserAssetController(s.Logger, s.Config, validator, &userAssetRepository)

		apiRouter := router.PathPrefix("/crypto").Subrouter()
		router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		// User Asset Routes
		apiRouter.HandleFunc("/users/assets", middlewares.NewMiddleware(logger, Config, userAssetController.CreateUserAssets).ValidateAuthToken(utility.Permissions["CreateUserAssets"]).LogAPIRequests().Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/users/{userId}/assets", middlewares.NewMiddleware(logger, Config, userAssetController.GetUserAssets).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/credit", middlewares.NewMiddleware(logger, Config, userAssetController.CreditUserAsset).ValidateAuthToken(utility.Permissions["CreditUserAsset"]).LogAPIRequests().Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/debit", middlewares.NewMiddleware(logger, Config, userAssetController.DebitUserAsset).ValidateAuthToken(utility.Permissions["DebitUserAsset"]).LogAPIRequests().Build()).Methods(http.MethodPost)

	})
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
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "denomination_id", "available_balance", "symbol"}).
			AddRow("60ed6eb5-41f9-482c-82e5-78abce7c142e", time.Now(), time.Now(), nil, "a10fce7b-7844-43af-9ed1-e130723a1ea3", "0c9f0ffe-169d-463e-b77f-bc36a8704db4", 0, "BTC"),
		)
	getAssetRequest, _ := http.NewRequest("GET", test.GetAssetEndpoint, bytes.NewBuffer([]byte("")))
	getAssetRequest.Header.Set("x-auth-token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOiIxNTc5MTI0NzA2IiwiaWF0IjoiMTU3OTEyMTEwNiIsImlzcyI6IlNWQ1MvQVVUSCIsInBlcm1pc3Npb25zIjpbInN2Y3MuY3J5cHRvLXdhbGxldC1hZGFwdGVyLmdldC1hc3NldHMiXSwic2VydmljZUlkIjoiNzZhYTcyZjctYjAwZS00OWRhLTgwN2ItNzVjZGUyZjEwZTI3IiwidG9rZW5UeXBlIjoiU0VSVklDRSJ9.jxs7G6kVWkCJineS8snanbYJtJXnlcMGU84AsZWAjLTBP_zlpNQSbxyVCmwGHBHdR0Yd0_URuyiJMSaLGNoMNDniHnxTisNVjj_BV7RgWAFKgO8_dnZEkScirZLCE-l8LwBfdQa4vja4_2yzt9gcIVrK5kQGNPvWLpDX2F0KAMYYD8GNeSBe2AtWXnaOY-OFAnJO8qfio3qrNP5EmpXyJ7hCIHXcSXzyTW6kp7D1T81AcRAF9269_ZUQGbbfVvXKzoixnrHZa7pOKjOeTqCYBwrwptKba0ExfP7GA3qP3PYgH4CwRphcluk6gGDz0otuBo3rA_bK8XdipknbmW3MOg")

	getAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(getAssetResponse, getAssetRequest)

	if getAssetResponse.Code != http.StatusOK {
		s.T().Errorf("Expected response code to not be %d. Got %d\n", http.StatusOK, getAssetResponse.Code)
	}
}

func (s *Suite) Test_CreateUserAsset() {
	s.Mock.ExpectQuery(regexp.QuoteMeta(
		fmt.Sprintf("SELECT * FROM `denominations` WHERE (`denominations`.`symbol` = ?) AND (`denominations`.`is_enabled` = ?) ORDER BY `denominations`.`id` ASC LIMIT 1"))).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "symbol", "tokenType", "decimal", "isEnabled"}).
			AddRow("60ed6eb5-41f9-482c-82e5-78abce7c142e", time.Now(), time.Now(), nil, "BTC", "BTC", 8, true),
		)

	s.Mock.ExpectQuery(regexp.QuoteMeta(
		fmt.Sprintf("SELECT denominations.symbol, denominations.decimal,user_balances.* FROM `user_balances`"))).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "asset_id", "available_balance", "symbol", "decimal"}).
			AddRow("60ed6eb5-41f9-482c-82e5-78abce7c142e", time.Now(), time.Now(), nil, "a10fce7b-7844-43af-9ed1-e130723a1ea3", "0c9f0ffe-169d-463e-b77f-bc36a8704db4", 0, "BTC", 8),
		)
	s.Mock.ExpectQuery(regexp.QuoteMeta("INSERT  INTO `user_balances`")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "denomination_id", "available_balance", "symbol"}).
			AddRow("60ed6eb5-41f9-482c-82e5-78abce7c142e", time.Now(), time.Now(), nil, "a10fce7b-7844-43af-9ed1-e130723a1ea3", "0c9f0ffe-169d-463e-b77f-bc36a8704db4", 0, "BTC"),
		)

	createAssetInputData := []byte(`{"assets" : ["BTC"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOiIxNTc5MTI0NzA2IiwiaWF0IjoiMTU3OTEyMTEwNiIsImlzcyI6IlNWQ1MvQVVUSCIsInBlcm1pc3Npb25zIjpbInN2Y3MuY3J5cHRvLXdhbGxldC1hZGFwdGVyLmNyZWF0ZS1hc3NldHMiXSwic2VydmljZUlkIjoiNzZhYTcyZjctYjAwZS00OWRhLTgwN2ItNzVjZGUyZjEwZTI3IiwidG9rZW5UeXBlIjoiU0VSVklDRSJ9.NZUQEBgvHrOKpg-f5zgGuqD-EAdPMCNKDQre53fg5ew04tZhbe3dDrneMsBEK2GxXz3M8-0XS6KIUqUhkn1RwvZO9cSmQpJ7DsTNNW07QGilqZw5d7ZFdJGilVA7mRwrDJ-xqyO2T7D-Gp6kWwOl9E1l3X3CSbpx2H9aSPK3Pbbffu_FAZKxitl6ao1b9DRbEEmPQ4-_mCXbGPeLp_2t2noW3zdW433eYaZV3_n6Y-MeZknmHOfp8h0MOo0u8pFf2jB4exU0L8Z_Vb3bNQz-jakhRsfYzgRR5SzcQ45_7EaZYXGSq0T415kGyAnNaugl7vte85YZmlbKbQpddMY6_A")

	createAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(createAssetResponse, createAssetRequest)
	if createAssetResponse.Code != http.StatusCreated {
		s.T().Errorf("Expected response code to not be %d. Got %d\n", http.StatusCreated, createAssetResponse.Code)
	}
}
