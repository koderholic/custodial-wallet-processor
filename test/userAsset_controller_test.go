package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
	Config "wallet-adapter/config"
	config "wallet-adapter/config"
	"wallet-adapter/controllers"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/middlewares"
	"wallet-adapter/utility"

	"github.com/stretchr/testify/require"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/suite"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	httpSwagger "github.com/swaggo/http-swagger"
	validation "gopkg.in/go-playground/validator.v9"
)

type Test struct {
	pingEndpoint           string
	CreateAssetEndpoint    string
	GetAssetEndpoint       string
	CreditAssetEndpoint    string
	DebitAssetEndpoint     string
	GetAddressEndpoint     string
	GetTransactionByRef    string
	GetTransactionsByAsset string
	GetAssetByIdEndpoint   string
}

var test = Test{
	pingEndpoint:           "/ping",
	CreateAssetEndpoint:    "/users/assets",
	GetAssetEndpoint:       "/users/a10fce7b-7844-43af-9ed1-e130723a1ea3/assets",
	GetAssetByIdEndpoint:   "/users/dbd77a9f-0dd9-4ff0-b17b-430e3895b82f/assets",
	CreditAssetEndpoint:    "/assets/credit",
	DebitAssetEndpoint:     "/assets/debit",
	GetAddressEndpoint:     "/assets/a10fce7b-7844-43af-9ed1-e130723a1ea3/address",
	GetTransactionByRef:    "/assets/transactions/9b7227pba3d915ef756a",
	GetTransactionsByAsset: "/assets/a10fce7b-7844-43af-9ed1-e130723a1ea3/transactions",
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
	Database database.Database
	Logger   *utility.Logger
	Config   config.Data
	Router   *mux.Router
}

var (
	once          sync.Once
	purgeInterval = 5 * time.Second
	cacheDuration = 60 * time.Second
	authCache     = utility.InitializeCache(cacheDuration, purgeInterval)
	authToken     = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJTVkNTL0FVVEgiLCJwZXJtaXNzaW9ucyI6WyJzdmNzLmNyeXB0by13YWxsZXQtYWRhcHRlci5jcmVkaXQtYXNzZXQiLCJzdmNzLmNyeXB0by13YWxsZXQtYWRhcHRlci5nZXQtYXNzZXRzIiwic3Zjcy5jcnlwdG8td2FsbGV0LWFkYXB0ZXIuY3JlYXRlLWFzc2V0cyIsInN2Y3MuY3J5cHRvLXdhbGxldC1hZGFwdGVyLmNyZWRpdC1hc3NldCIsInN2Y3MuY3J5cHRvLXdhbGxldC1hZGFwdGVyLmRlYml0LWFzc2V0Iiwic3Zjcy5jcnlwdG8td2FsbGV0LWFkYXB0ZXIuZG8taW50ZXJuYWwtdHJhbnNmZXIiLCJzdmNzLmNyeXB0by13YWxsZXQtYWRhcHRlci5nZXQtYWRkcmVzcyIsInN2Y3MuY3J5cHRvLXdhbGxldC1hZGFwdGVyLmdldC10cmFuc2FjdGlvbnMiLCJzdmNzLmNyeXB0by13YWxsZXQtYWRhcHRlci5vbi1jaGFpbi1kZXBvc2l0Iiwic3Zjcy5jcnlwdG8td2FsbGV0LWFkYXB0ZXIuY29uZmlybS10cmFuc2FjdGlvbiIsInN2Y3MuY3J5cHRvLXdhbGxldC1hZGFwdGVyLmRvLWV4dGVybmFsLXRyYW5zZmVyIiwic3Zjcy5jcnlwdG8td2FsbGV0LWFkYXB0ZXIucHJvY2Vzcy10cmFuc2FjdGlvbnMiXSwic2VydmljZUlkIjoiNzZhYTcyZjctYjAwZS00OWRhLTgwN2ItNzVjZGUyZjEwZTI3IiwidG9rZW5UeXBlIjoiU0VSVklDRSJ9.ImOiJYkjwGG5_-E4FDUO3LRKZFDLxv3WLpgDt__Ih42B4jUlJ7pl4YJPfSJBc0vM1A57fjuPdJ8NhCd0wcIkxOuDDXJuon5xE1NIr0muIbPWQjNtpkgcVy9gSYBgHAERAFNkSIV_GWvki06uIT0DoQviWTWZmwuG112jquRpfyYV8M5l2pE-xtpf75quQBQQU08EEA-dS17iR4VaaTiCD584o9ujO-Wql9PBs8NK5g1kBpqpOWj2jIpa0NQSYlwijOw2cKL91KpTS0xxG1AXMzvyOyQK-QVpTX09tJrqsmzYHH49Zg5AlaTmiHbsSDhxacdiIE7O_Ge0T1B6PC_SLA"
)

func TestInit(t *testing.T) {
	suite.Run(t, new(Suite))
}

// SetupSuite ...
func (s *Suite) SetupSuite() {

	db, err := gorm.Open("sqlite3", "./walletAdapter.db")
	s.DB = db
	require.NoError(s.T(), err)
	s.DB.LogMode(true)

	logger := utility.NewLogger()
	router := mux.NewRouter()
	validator := validation.New()
	Config := config.Data{
		AppPort:               "9000",
		ServiceName:           "crypto-wallet-adapter",
		AuthenticatorKey:      "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE0ZjV3ZzVsMmhLc1RlTmVtL1Y0MQpmR25KbTZnT2Ryajh5bTNyRmtFVS93VDhSRHRuU2dGRVpPUXBIRWdRN0pMMzh4VWZVMFkzZzZhWXc5UVQwaEo3Cm1DcHo5RXI1cUxhTVhKd1p4ekh6QWFobGZBMGljcWFidkpPTXZRdHpENnVRdjZ3UEV5WnREVFdpUWk5QVh3QnAKSHNzUG5wWUdJbjIwWlp1TmxYMkJyQ2xjaUhoQ1BVSUlaT1FuL01tcVREMzFqU3lqb1FvVjdNaGhNVEFUS0p4MgpYckhoUisxRGNLSnpRQlNUQUducFlWYXFwc0FSYXArbndSaXByM25VVHV4eUdvaEJUU21qSjJ1c1NlUVhISTNiCk9ESVJlMUF1VHlIY2VBYmV3bjhiNDYyeUVXS0FSZHBkOUFqUVc1U0lWUGZkc3o1QjZHbFlRNUxkWUt0em5UdXkKN3dJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0t",
		PurgeCacheInterval:    60,
		ServiceID:             "4b0bde7a-9201-4cf9-859f-e61d976e376d",
		ServiceKey:            "32e1f6396de342e879ca07ec68d4d907",
		AuthenticationService: "https://internal.dev.bundlewallet.com/authentication",
		KeyManagementService:  "https://internal.dev.bundlewallet.com/key-management",
		CryptoAdapterService:  "https://internal.dev.bundlewallet.com/crypto-adapter",
		LockerService:         "https://internal.dev.bundlewallet.com/locker",
		ExpireCacheDuration:   400,
		RequestTimeout:        60,
		MaxIdleConns:          25,
		MaxOpenConns:          50,
		ConnMaxLifetime:       300,
		LockerPrefix:          "Wallet-Adapter-Lock-",
	}

	Database := database.Database{
		Logger: logger,
		Config: Config,
		DB:     s.DB,
	}

	s.Database = Database
	s.Logger = logger
	s.Config = Config
	s.Router = router

	s.RunMigration()
	s.DBSeeder()
	s.RegisterRoutes(logger, Config, router, validator)
}

func (s *Suite) TearDownTestSuite() {
	err := os.Remove("./walletAdapter.db")
	if err != nil {
		s.Logger.Error("Error with deleting the test database : ", err)
	}
}

// RegisterRoutes ...
func (s *Suite) RegisterRoutes(logger *utility.Logger, Config config.Data, router *mux.Router, validator *validation.Validate) {

	once.Do(func() {

		baseRepository := database.BaseRepository{Database: s.Database}
		userAssetRepository := database.UserAssetRepository{BaseRepository: baseRepository}

		controller := controllers.NewController(authCache, s.Logger, s.Config, validator, &baseRepository)
		userAssetController := controllers.NewUserAssetController(authCache, s.Logger, s.Config, validator, &userAssetRepository)

		apiRouter := router.PathPrefix("").Subrouter()
		router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		// User Asset Routes
		apiRouter.HandleFunc("/users/assets", middlewares.NewMiddleware(logger, Config, userAssetController.CreateUserAssets).ValidateAuthToken(utility.Permissions["CreateUserAssets"]).LogAPIRequests().Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/users/{userId}/assets", middlewares.NewMiddleware(logger, Config, userAssetController.GetUserAssets).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/credit", middlewares.NewMiddleware(logger, Config, userAssetController.CreditUserAsset).ValidateAuthToken(utility.Permissions["CreditUserAsset"]).LogAPIRequests().Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/debit", middlewares.NewMiddleware(logger, Config, userAssetController.DebitUserAsset).ValidateAuthToken(utility.Permissions["DebitUserAsset"]).LogAPIRequests().Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/transfer-internal", middlewares.NewMiddleware(logger, Config, userAssetController.InternalTransfer).ValidateAuthToken(utility.Permissions["InternalTransfer"]).LogAPIRequests().Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/by-id/{assetId}", middlewares.NewMiddleware(logger, Config, userAssetController.GetUserAssetById).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/by-address/{address}", middlewares.NewMiddleware(logger, Config, userAssetController.GetUserAssetByAddress).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/{assetId}/address", middlewares.NewMiddleware(logger, Config, userAssetController.GetAssetAddress).ValidateAuthToken(utility.Permissions["GetAssetAddress"]).LogAPIRequests().Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/transactions/{reference}", middlewares.NewMiddleware(logger, Config, controller.GetTransaction).ValidateAuthToken(utility.Permissions["GetTransaction"]).LogAPIRequests().Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/{assetId}/transactions", middlewares.NewMiddleware(logger, Config, controller.GetTransactionsByAssetId).ValidateAuthToken(utility.Permissions["GetTransaction"]).LogAPIRequests().Build()).Methods(http.MethodGet)

	})
}

// RunDbMigrations ... This creates corresponding tables for dtos on the db for testing
func (s *Suite) RunMigration() {
	s.DB.AutoMigrate(&dto.Denomination{}, &dto.BatchRequest{}, &dto.ChainTransaction{}, &dto.Transaction{}, &dto.UserAddress{}, &dto.UserAsset{}, &dto.HotWalletAsset{}, &dto.TransactionQueue{})
	s.DB.Model(&dto.UserAsset{}).AddForeignKey("denomination_id", "denominations(id)", "CASCADE", "CASCADE")
	s.DB.Model(&dto.UserAddress{}).AddForeignKey("asset_id", "user_assets(id)", "CASCADE", "CASCADE")
	s.DB.Model(&dto.Transaction{}).AddForeignKey("recipient_id", "user_assets(id)", "CASCADE", "CASCADE")
	s.DB.Model(&dto.TransactionQueue{}).AddForeignKey("transaction_id", "transactions(id)", "CASCADE", "CASCADE")
	s.DB.Model(&dto.TransactionQueue{}).AddForeignKey("debit_reference", "transactions(transaction_reference)", "NO ACTION", "NO ACTION")
}

// DBSeeder .. This seeds supported assets into the database for testing
func (s *Suite) DBSeeder() {

	assets := []dto.Denomination{
		dto.Denomination{Name: "Binance Coin", AssetSymbol: "BNB", CoinType: 714, Decimal: 8},
		dto.Denomination{Name: "Ethereum Coin", AssetSymbol: "ETH", CoinType: 60, Decimal: 18},
		dto.Denomination{Name: "Bitcoin", AssetSymbol: "BTC", CoinType: 0, Decimal: 8},
	}

	for _, asset := range assets {
		if err := s.DB.FirstOrCreate(&asset, dto.Denomination{AssetSymbol: asset.AssetSymbol}).Error; err != nil {
			s.Logger.Error("Error with creating asset record %s : %s", asset.AssetSymbol, err)
		}
	}
	s.Logger.Info("Supported assets seeded successfully")
}

func (s *Suite) Test_CreateUserAsset() {

	createAssetInputData := []byte(`{"assets" : ["BTC","ETH","BNB"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", authToken)

	response := httptest.NewRecorder()
	s.Router.ServeHTTP(response, createAssetRequest)

	resBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse := map[string][]interface{}{}
	err = json.Unmarshal(resBody, &response)

	if response.Code != http.StatusCreated && len(createAssetResponse["assets"]) != 3 {
		s.T().Errorf("Expected response code to be %d and length of assets returned to be %d. Got responseCode of %d and assets length of %d\n", 201, 3, response.Code, len(createAssetResponse["assets"]))
	}
}
func (s *Suite) Test_GetUserAsset() {
	createAssetInputData := []byte(`{"assets" : ["BTC","ETH","BNB"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", authToken)
	createAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(createAssetResponse, createAssetRequest)

	getAssetRequest, _ := http.NewRequest("GET", test.GetAssetEndpoint, bytes.NewBuffer([]byte("")))
	getAssetRequest.Header.Set("x-auth-token", authToken)

	response := httptest.NewRecorder()
	s.Router.ServeHTTP(response, getAssetRequest)
	resBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAssetResponse := map[string][]interface{}{}
	err = json.Unmarshal(resBody, &response)
	fmt.Printf("getAssetResponse > %+v", getAssetResponse)

	if response.Code != http.StatusOK && len(getAssetResponse) != 3 {
		s.T().Errorf("Expected response code to be %d and length of assets returned to be %d. Got responseCode of %d and assets length of %d\n", 200, 3, response.Code, len(getAssetResponse))
	}
}
