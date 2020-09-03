package test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
	"wallet-adapter/config"
	"wallet-adapter/controllers"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/middlewares"
	"wallet-adapter/model"
	"wallet-adapter/services"
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
	pingEndpoint               string
	CreateAssetEndpoint        string
	GetAssetEndpoint           string
	CreditAssetEndpoint        string
	DebitAssetEndpoint         string
	GetTransactionByRef        string
	OnchainDepositEndpoint     string
	InternalTransferEndpoint   string
	TransferExternalEndpoint   string
	ProcessTransactionEndpoint string
}

var test = Test{
	pingEndpoint:               "/ping",
	CreateAssetEndpoint:        "/users/assets",
	GetAssetEndpoint:           "/users/a10fce7b-7844-43af-9ed1-e130723a1ea3/assets",
	CreditAssetEndpoint:        "/assets/credit",
	OnchainDepositEndpoint:     "/assets/onchain-deposit",
	DebitAssetEndpoint:         "/assets/debit",
	InternalTransferEndpoint:   "/assets/transfer-internal",
	GetTransactionByRef:        "/assets/transactions/",
	TransferExternalEndpoint:   "/assets/transfer-external",
	ProcessTransactionEndpoint: "/assets/process-transaction",
}

//BaseController : Base controller struct
type Controller struct {
	Logger     *utility.Logger
	Config     config.Data
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
	dir, err := os.Getwd()
	if err != nil {
		require.NoError(s.T(), err)
	}
	db, err := gorm.Open("sqlite3", dir+"/walletAdapter.db")
	db.DB().SetMaxOpenConns(1)

	s.DB = db
	require.NoError(s.T(), err)

	if err = os.Chmod(dir+"/walletAdapter.db", 0777); err != nil {
		require.NoError(s.T(), err)
	}

	logger := utility.NewLogger()
	router := mux.NewRouter()
	validator := validation.New()
	Config := config.Data{
		AppPort:                "9000",
		ServiceName:            "crypto-wallet-adapter",
		AuthenticatorKey:       "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE0ZjV3ZzVsMmhLc1RlTmVtL1Y0MQpmR25KbTZnT2Ryajh5bTNyRmtFVS93VDhSRHRuU2dGRVpPUXBIRWdRN0pMMzh4VWZVMFkzZzZhWXc5UVQwaEo3Cm1DcHo5RXI1cUxhTVhKd1p4ekh6QWFobGZBMGljcWFidkpPTXZRdHpENnVRdjZ3UEV5WnREVFdpUWk5QVh3QnAKSHNzUG5wWUdJbjIwWlp1TmxYMkJyQ2xjaUhoQ1BVSUlaT1FuL01tcVREMzFqU3lqb1FvVjdNaGhNVEFUS0p4MgpYckhoUisxRGNLSnpRQlNUQUducFlWYXFwc0FSYXArbndSaXByM25VVHV4eUdvaEJUU21qSjJ1c1NlUVhISTNiCk9ESVJlMUF1VHlIY2VBYmV3bjhiNDYyeUVXS0FSZHBkOUFqUVc1U0lWUGZkc3o1QjZHbFlRNUxkWUt0em5UdXkKN3dJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0t",
		PurgeCacheInterval:     60,
		ServiceID:              "4b0bde7a-9201-4cf9-859f-e61d976e376d",
		ServiceKey:             "32e1f6396de342e879ca07ec68d4d907",
		AuthenticationService:  "https://internal.dev.bundlewallet.com/authentication",
		KeyManagementService:   "https://internal.dev.bundlewallet.com/key-management",
		CryptoAdapterService:   "https://internal.dev.bundlewallet.com/crypto-adapter",
		LockerService:          "https://internal.dev.bundlewallet.com/locker",
		DepositWebhookURL:      "http://internal.dev.bundlewallet.com/crypto-adapter/incoming-deposit",
		WithdrawToHotWalletUrl: "http://order-book",
		NotificationServiceUrl: "http://internal.dev.bundlewallet.com/notifications",
		ExpireCacheDuration:    400,
		RequestTimeout:         60,
		MaxIdleConns:           25,
		MaxOpenConns:           50,
		ConnMaxLifetime:        300,
		LockerPrefix:           "Wallet-Adapter-Lock-",
		ETH_minimumSweep:       0.9,
		BnbSlipValue:           "714",
		BtcSlipValue:           "0",
		EthSlipValue:           "60",
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

	s.RegisterRoutes(logger, Config, router, validator)
}

func (s *Suite) SetupTest() {
	s.RunMigration()
	s.DBSeeder()
}

func (s *Suite) TearDownTest() {
	s.DB.DropTableIfExists(&model.Denomination{}, &model.BatchRequest{}, &model.ChainTransaction{}, &model.Transaction{}, &model.UserAddress{}, &model.UserAsset{}, &model.HotWalletAsset{}, &model.TransactionQueue{})
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
		var requestTimeout = time.Duration(s.Config.RequestTimeout) * time.Second
		apiRouter.HandleFunc("/users/assets", middlewares.NewMiddleware(logger, s.Config, userAssetController.CreateUserAssets).ValidateAuthToken(utility.Permissions["CreateUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/users/{userId}/assets", middlewares.NewMiddleware(logger, s.Config, userAssetController.GetUserAssets).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/credit", middlewares.NewMiddleware(logger, s.Config, userAssetController.CreditUserAsset).ValidateAuthToken(utility.Permissions["CreditUserAsset"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/onchain-deposit", middlewares.NewMiddleware(logger, s.Config, userAssetController.OnChainCreditUserAsset).ValidateAuthToken(utility.Permissions["OnChainDeposit"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/debit", middlewares.NewMiddleware(logger, s.Config, userAssetController.DebitUserAsset).ValidateAuthToken(utility.Permissions["DebitUserAsset"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/transfer-internal", middlewares.NewMiddleware(logger, s.Config, userAssetController.InternalTransfer).ValidateAuthToken(utility.Permissions["InternalTransfer"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/by-id/{assetId}", middlewares.NewMiddleware(logger, s.Config, userAssetController.GetUserAssetById).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/by-address/{address}", middlewares.NewMiddleware(logger, s.Config, userAssetController.GetUserAssetByAddress).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/{assetId}/address", middlewares.NewMiddleware(logger, s.Config, userAssetController.GetAssetAddress).ValidateAuthToken(utility.Permissions["GetAssetAddress"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/transactions/{reference}", middlewares.NewMiddleware(logger, s.Config, controller.GetTransaction).ValidateAuthToken(utility.Permissions["GetTransaction"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/{assetId}/transactions", middlewares.NewMiddleware(logger, s.Config, controller.GetTransactionsByAssetId).ValidateAuthToken(utility.Permissions["GetTransaction"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/transfer-external", middlewares.NewMiddleware(logger, s.Config, userAssetController.ExternalTransfer).ValidateAuthToken(utility.Permissions["ExternalTransfer"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/confirm-transaction", middlewares.NewMiddleware(logger, s.Config, userAssetController.ConfirmTransaction).ValidateAuthToken(utility.Permissions["ConfirmTransaction"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/process-transaction", middlewares.NewMiddleware(logger, s.Config, userAssetController.ProcessTransactions).LogAPIRequests().Build()).Methods(http.MethodPost)

	})
}

// RunDbMigrations ... This creates corresponding tables for dtos on the db for testing
func (s *Suite) RunMigration() {
	s.DB.AutoMigrate(&model.Denomination{}, &model.BatchRequest{}, &model.SharedAddress{}, &model.ChainTransaction{}, &model.Transaction{}, &model.UserAddress{}, &model.UserAsset{}, &model.HotWalletAsset{}, &model.TransactionQueue{})
}

// DBSeeder .. This seeds supported assets into the database for testing
func (s *Suite) DBSeeder() {

	assets := []model.Denomination{
		model.Denomination{Name: "Binance Coin", AssetSymbol: "BNB", CoinType: 714, Decimal: 8},
		model.Denomination{Name: "Binance USD", AssetSymbol: "BUSD", CoinType: 714, Decimal: 8},
		model.Denomination{Name: "Ethereum Coin", AssetSymbol: "ETH", CoinType: 60, Decimal: 18},
		model.Denomination{Name: "Bitcoin", AssetSymbol: "BTC", CoinType: 0, Decimal: 8},
	}

	for _, asset := range assets {
		if err := s.DB.FirstOrCreate(&asset, model.Denomination{AssetSymbol: asset.AssetSymbol}).Error; err != nil {
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
	createAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse)

	if response.Code != http.StatusCreated || len(createAssetResponse.Assets) != 3 {
		s.T().Errorf("Expected response code to be %d and length of assets returned to be %d. Got responseCode of %d and assets length of %d\n", 201, 3, response.Code, len(createAssetResponse.Assets))
	}
}

func (s *Suite) Test_CreateUserBUSDAsset() {

	createAssetInputData := []byte(`{"assets" : ["BUSD"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", authToken)

	response := httptest.NewRecorder()
	s.Router.ServeHTTP(response, createAssetRequest)

	resBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse)

	if response.Code != http.StatusCreated || len(createAssetResponse.Assets) != 1 {
		s.T().Errorf("Expected response code to be %d and length of assets returned to be %d. Got responseCode of %d and assets length of %d\n", 201, 1, response.Code, len(createAssetResponse.Assets))
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
	getAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &getAssetResponse)

	if response.Code != http.StatusOK || len(getAssetResponse.Assets) != 3 {
		s.T().Errorf("Expected response code to be %d and length of assets returned to be %d. Got responseCode of %d and assets length of %d\n", 200, 3, response.Code, len(getAssetResponse.Assets))
	}
}
func (s *Suite) Test_GetUserAssetById() {
	createAssetInputData := []byte(`{"assets" : ["BTC","ETH","BNB"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", authToken)
	createResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(createResponse, createAssetRequest)
	resBody, err := ioutil.ReadAll(createResponse.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse)
	if createResponse.Code != http.StatusCreated || len(createAssetResponse.Assets) < 1 {
		require.NoError(s.T(), errors.New("Expected asset creation to not error"))
	}

	getAssetRequest, _ := http.NewRequest("GET", fmt.Sprintf("/assets/by-id/%s", createAssetResponse.Assets[0].ID), bytes.NewBuffer([]byte("")))
	getAssetRequest.Header.Set("x-auth-token", authToken)

	response := httptest.NewRecorder()
	s.Router.ServeHTTP(response, getAssetRequest)
	resBody, err = ioutil.ReadAll(response.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAssetResponse := dto.Asset{}
	err = json.Unmarshal(resBody, &getAssetResponse)

	fmt.Printf("getAssetResponse >> %+v", getAssetResponse.UserID.String())

	if response.Code != http.StatusOK || getAssetResponse.UserID.String() == "00000000-0000-0000-0000-000000000000" {
		s.T().Errorf("Expected response code to be %d and userId not empty. Got responseCode of %d and %s\n", http.StatusOK, response.Code, getAssetResponse.UserID.String())
	}
}
func (s *Suite) Test_GetUserAssetAddress() {
	createAssetInputData := []byte(`{"assets" : ["BTC","ETH","BNB"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", authToken)
	createResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(createResponse, createAssetRequest)
	resBody, err := ioutil.ReadAll(createResponse.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse)
	if createResponse.Code != http.StatusCreated || len(createAssetResponse.Assets) < 1 {
		require.NoError(s.T(), errors.New("Expected asset creation to not error"))
	}
	// First time call to get address
	getNewAssetAddressRequest, _ := http.NewRequest("GET", fmt.Sprintf("/assets/%s/address", createAssetResponse.Assets[0].ID), bytes.NewBuffer([]byte("")))
	getNewAssetAddressRequest.Header.Set("x-auth-token", authToken)

	response := httptest.NewRecorder()
	s.Router.ServeHTTP(response, getNewAssetAddressRequest)
	resBody, err = ioutil.ReadAll(response.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getNewAssetAddressResponse := map[string]string{}
	err = json.Unmarshal(resBody, &getNewAssetAddressResponse)
	//  Second time call to get address
	getOldAssetAddressRequest, _ := http.NewRequest("GET", fmt.Sprintf("/assets/%s/address", createAssetResponse.Assets[0].ID), bytes.NewBuffer([]byte("")))
	getOldAssetAddressRequest.Header.Set("x-auth-token", authToken)
	response2 := httptest.NewRecorder()
	s.Router.ServeHTTP(response2, getOldAssetAddressRequest)
	resBody2, err := ioutil.ReadAll(response2.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getOldAssetAddressResponse := map[string]string{}
	err = json.Unmarshal(resBody2, &getOldAssetAddressResponse)

	if response.Code != http.StatusOK || response2.Code != http.StatusOK || getNewAssetAddressResponse["address"] == "" || getNewAssetAddressResponse["address"] != getOldAssetAddressResponse["address"] {
		s.T().Errorf("Expected response code to be %d and asset address to not be empty and the two calls to get address to return same address. Got responseCode of %d and address of %s and the equality of both address to be %t\n", http.StatusOK, response.Code, getNewAssetAddressResponse["address"], getNewAssetAddressResponse["address"] == getOldAssetAddressResponse["address"])
	}
}

func (s *Suite) Test_GetUserAssetAddrReturnsSameAddrForSameCoinType() {
	if err := services.InitHotWallet(authCache, s.DB, s.Logger, s.Config); err != nil {
		require.NoError(s.T(), err)
	}

	createAssetInputData := []byte(`{"assets" : ["BNB","BUSD"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", authToken)
	createResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(createResponse, createAssetRequest)
	resBody, err := ioutil.ReadAll(createResponse.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse)
	if createResponse.Code != http.StatusCreated || len(createAssetResponse.Assets) < 1 {
		require.NoError(s.T(), errors.New("Expected asset creation to not error"))
	}
	// call to get address for asset 1
	getAsset1AddressRequest, _ := http.NewRequest("GET", fmt.Sprintf("/assets/%s/address", createAssetResponse.Assets[0].ID), bytes.NewBuffer([]byte("")))
	getAsset1AddressRequest.Header.Set("x-auth-token", authToken)

	response := httptest.NewRecorder()
	s.Router.ServeHTTP(response, getAsset1AddressRequest)
	resBody, err = ioutil.ReadAll(response.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAsset1AddressResponse := map[string]string{}
	err = json.Unmarshal(resBody, &getAsset1AddressResponse)
	fmt.Printf("getAsset1AddressResponse >> %+v", getAsset1AddressResponse)
	//  call to get address for asset 2
	getAsset2AddressRequest, _ := http.NewRequest("GET", fmt.Sprintf("/assets/%s/address", createAssetResponse.Assets[1].ID), bytes.NewBuffer([]byte("")))
	getAsset2AddressRequest.Header.Set("x-auth-token", authToken)
	response2 := httptest.NewRecorder()
	s.Router.ServeHTTP(response2, getAsset2AddressRequest)
	resBody2, err := ioutil.ReadAll(response2.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAsset2AddressResponse := map[string]string{}
	err = json.Unmarshal(resBody2, &getAsset2AddressResponse)
	fmt.Printf("getAsset2AddressResponse >> %+v", getAsset2AddressResponse)

	if response.Code != http.StatusOK || response2.Code != http.StatusOK || getAsset1AddressResponse["address"] == "" || getAsset1AddressResponse["address"] != getAsset2AddressResponse["address"] {
		s.T().Errorf("Expected response code to be %d and asset address to not be empty and the address of same coinType to be the same. Got responseCode of %d and address of %s and the equality of both address to be %t\n", http.StatusOK, response.Code, getAsset1AddressResponse["address"], getAsset1AddressResponse["address"] == getAsset2AddressResponse["address"])
	}
}

func (s *Suite) Test_GetUserAssetByAddress() {
	createAssetInputData := []byte(`{"assets" : ["BNB"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", authToken)
	createResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(createResponse, createAssetRequest)
	resBody, err := ioutil.ReadAll(createResponse.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse)
	if createResponse.Code != http.StatusCreated || len(createAssetResponse.Assets) < 1 {
		require.NoError(s.T(), errors.New("Expected asset creation to not error"))
	}
	// First time call to get address
	getAssetAddressRequest, _ := http.NewRequest("GET", fmt.Sprintf("/assets/%s/address?assetSymbol=BNB", createAssetResponse.Assets[0].ID), bytes.NewBuffer([]byte("")))
	getAssetAddressRequest.Header.Set("x-auth-token", authToken)
	responseAddress := httptest.NewRecorder()
	s.Router.ServeHTTP(responseAddress, getAssetAddressRequest)
	resBody, err = ioutil.ReadAll(responseAddress.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAssetAddressResponse := map[string]string{}
	err = json.Unmarshal(resBody, &getAssetAddressResponse)

	getAssetRequest, _ := http.NewRequest("GET", fmt.Sprintf("/assets/by-address/%s?assetSymbol=BNB", getAssetAddressResponse["address"]), bytes.NewBuffer([]byte("")))
	getAssetRequest.Header.Set("x-auth-token", authToken)
	response := httptest.NewRecorder()
	s.Router.ServeHTTP(response, getAssetRequest)
	resBody, err = ioutil.ReadAll(response.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAssetResponse := dto.Asset{}
	err = json.Unmarshal(resBody, &getAssetResponse)

	if response.Code != http.StatusOK {
		s.T().Errorf("Expected response code to be %d. Got responseCode of %d\n", http.StatusOK, response.Code)
	}
}
func (s *Suite) Test_CreditUserAsset() {
	createAssetInputData := []byte(`{"assets" : ["BTC","ETH","BNB"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", authToken)
	createResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(createResponse, createAssetRequest)
	resBody, err := ioutil.ReadAll(createResponse.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse)
	if createResponse.Code != http.StatusCreated || len(createAssetResponse.Assets) < 1 {
		require.NoError(s.T(), errors.New("Expected asset creation to not error"))
	}

	creditAssetInputData := []byte(fmt.Sprintf(`{"assetId" : "%s","value" : 200.30,"transactionReference" : "ra29bv7y111p945e17514","memo" :"Test credit transaction"}`, createAssetResponse.Assets[0].ID))
	creditAssetRequest, _ := http.NewRequest("POST", test.CreditAssetEndpoint, bytes.NewBuffer(creditAssetInputData))
	creditAssetRequest.Header.Set("x-auth-token", authToken)
	creditAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(creditAssetResponse, creditAssetRequest)

	getAssetRequest, _ := http.NewRequest("GET", test.GetAssetEndpoint, bytes.NewBuffer([]byte("")))
	getAssetRequest.Header.Set("x-auth-token", authToken)

	response := httptest.NewRecorder()
	s.Router.ServeHTTP(response, getAssetRequest)
	resBody, err = ioutil.ReadAll(response.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &getAssetResponse)
	if len(getAssetResponse.Assets) < 1 {
		require.NoError(s.T(), errors.New("No assests returned"))
	}

	if response.Code != http.StatusOK || len(getAssetResponse.Assets) < 1 || getAssetResponse.Assets[0].AvailableBalance == "200.30" {
		s.T().Errorf("Expected statusCode to be %d and asset balance to be %s. Got %d and %s\n", http.StatusOK, "200.30", response.Code, createAssetResponse.Assets[0].AvailableBalance)
	}
}

func (s *Suite) Test_DebitUserAsset() {
	createAssetInputData := []byte(`{"assets" : ["BTC","ETH","BNB"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", authToken)
	createResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(createResponse, createAssetRequest)
	resBody, err := ioutil.ReadAll(createResponse.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse)
	if createResponse.Code != http.StatusCreated || len(createAssetResponse.Assets) < 1 {
		require.NoError(s.T(), errors.New("Expected asset creation to not error"))
	}

	creditAssetInputData := []byte(fmt.Sprintf(`{"assetId" : "%s","value" : 200.30,"transactionReference" : "ra29bv7y111p945e17514","memo" :"Test credit transaction"}`, createAssetResponse.Assets[0].ID))
	creditAssetRequest, _ := http.NewRequest("POST", test.CreditAssetEndpoint, bytes.NewBuffer(creditAssetInputData))
	creditAssetRequest.Header.Set("x-auth-token", authToken)
	creditAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(creditAssetResponse, creditAssetRequest)

	debitAssetInputData := []byte(fmt.Sprintf(`{"assetId" : "%s","value" : 10.30,"transactionReference" : "ra29bv7y111p945e17515","memo" :"Test credit transaction"}`, createAssetResponse.Assets[0].ID))
	debitAssetRequest, _ := http.NewRequest("POST", test.DebitAssetEndpoint, bytes.NewBuffer(debitAssetInputData))
	debitAssetRequest.Header.Set("x-auth-token", authToken)
	debitAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(debitAssetResponse, debitAssetRequest)

	getAssetRequest, _ := http.NewRequest("GET", test.GetAssetEndpoint, bytes.NewBuffer([]byte("")))
	getAssetRequest.Header.Set("x-auth-token", authToken)

	response := httptest.NewRecorder()
	s.Router.ServeHTTP(response, getAssetRequest)
	resBody, err = ioutil.ReadAll(response.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &getAssetResponse)
	if len(getAssetResponse.Assets) < 1 {
		require.NoError(s.T(), errors.New("No assests returned"))
	}

	if response.Code != http.StatusOK || len(getAssetResponse.Assets) < 1 || getAssetResponse.Assets[0].AvailableBalance != "190" {
		s.T().Errorf("Expected statusCode to be %d and asset balance to be %s. Got %d and %s\n", http.StatusOK, "190", response.Code, createAssetResponse.Assets[0].AvailableBalance)
	}
}
func (s *Suite) Test_ExternalTransfer() {
	createAssetInputData := []byte(`{"assets" : ["BTC","ETH","BNB"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", authToken)
	createResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(createResponse, createAssetRequest)
	resBody, err := ioutil.ReadAll(createResponse.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse)
	if createResponse.Code != http.StatusCreated || len(createAssetResponse.Assets) < 1 {
		require.NoError(s.T(), errors.New("Expected asset creation to not error"))
	}

	creditAssetInputData := []byte(fmt.Sprintf(`{"assetId" : "%s","value" : 200.30,"transactionReference" : "ra29bv7y111p945e17514","memo" :"Test credit transaction"}`, createAssetResponse.Assets[0].ID))
	creditAssetRequest, _ := http.NewRequest("POST", test.CreditAssetEndpoint, bytes.NewBuffer(creditAssetInputData))
	creditAssetRequest.Header.Set("x-auth-token", authToken)
	creditAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(creditAssetResponse, creditAssetRequest)
	if creditAssetResponse.Code != http.StatusOK {
		require.NoError(s.T(), errors.New("Expected credit asset to not error"))
	}

	debitAssetInputData := []byte(fmt.Sprintf(`{"assetId" : "%s","value" : 10.30,"transactionReference" : "ra29bv7y111p945e17515","memo" :"Test credit transaction"}`, createAssetResponse.Assets[0].ID))
	debitAssetRequest, _ := http.NewRequest("POST", test.DebitAssetEndpoint, bytes.NewBuffer(debitAssetInputData))
	debitAssetRequest.Header.Set("x-auth-token", authToken)
	debitAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(debitAssetResponse, debitAssetRequest)
	if debitAssetResponse.Code != http.StatusOK {
		require.NoError(s.T(), errors.New("Expected debit asset to not error"))
	}
	println("!!!!!!!!!!!!!!!!!!!!!!")
	externalTransferInputData := []byte(`{"recipientAddress" : "bnb1k05t5h6h7t4mq9tvafz2mx8c29jz2w4r0l0hda","value" : 10.00,"debitReference" : "ra29bv7y111p945e17515","transactionReference" : "ra29bv7y111p945e17516"}`)
	externalTransferRequest, _ := http.NewRequest("POST", test.TransferExternalEndpoint, bytes.NewBuffer(externalTransferInputData))
	externalTransferRequest.Header.Set("x-auth-token", authToken)
	externalTransferResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(externalTransferResponse, externalTransferRequest)
	if externalTransferResponse.Code != http.StatusOK {
		require.NoError(s.T(), errors.New(fmt.Sprintf("Expected external transfer asset to not error >> %+v", externalTransferResponse)))
	}

	getAssetTransactionRequest, _ := http.NewRequest("GET", test.GetTransactionByRef+"ra29bv7y111p945e17516", bytes.NewBuffer([]byte("")))
	getAssetTransactionRequest.Header.Set("x-auth-token", authToken)
	response := httptest.NewRecorder()
	s.Router.ServeHTTP(response, getAssetTransactionRequest)
	resBody, err = ioutil.ReadAll(response.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAssetTransactionResponse := dto.TransactionResponse{}
	err = json.Unmarshal(resBody, &getAssetTransactionResponse)

	queuedTransaction := model.TransactionQueue{}
	s.DB.Raw("SELECT * from transaction_queues where transaction_id = ?", getAssetTransactionResponse.ID).Scan(&queuedTransaction)

	if response.Code != http.StatusOK || getAssetTransactionResponse.RecipientID.String() != createAssetResponse.Assets[0].ID.String() || queuedTransaction.Recipient != "bnb1k05t5h6h7t4mq9tvafz2mx8c29jz2w4r0l0hda" {
		s.T().Errorf("Expected statusCode to be %d, external transaction recipientId to be %s and external recipient to be %s. Got %d, %s and %s\n", http.StatusOK, createAssetResponse.Assets[0].ID, "bnb1k05t5h6h7t4mq9tvafz2mx8c29jz2w4r0l0hda", response.Code, getAssetTransactionResponse.RecipientID, queuedTransaction.Recipient)
	}
}
func (s *Suite) Test_InternalAssetTransfer() {
	// Asset 1
	createAssetInputData1 := []byte(`{"assets" : ["BTC","ETH","BNB"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest1, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData1))
	createAssetRequest1.Header.Set("x-auth-token", authToken)
	createResponse1 := httptest.NewRecorder()
	s.Router.ServeHTTP(createResponse1, createAssetRequest1)
	resBody, err := ioutil.ReadAll(createResponse1.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse1 := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse1)
	if createResponse1.Code != http.StatusCreated || len(createAssetResponse1.Assets) < 1 {
		require.NoError(s.T(), errors.New("Expected asset creation to not error"))
	}
	// Asset 2
	createAssetInputData2 := []byte(`{"assets" : ["BTC","ETH","BNB"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1e44"}`)
	createAssetRequest2, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData2))
	createAssetRequest2.Header.Set("x-auth-token", authToken)
	createResponse2 := httptest.NewRecorder()
	s.Router.ServeHTTP(createResponse2, createAssetRequest2)
	resBody, err = ioutil.ReadAll(createResponse2.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse2 := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse2)
	if createResponse1.Code != http.StatusCreated || len(createAssetResponse2.Assets) < 1 {
		require.NoError(s.T(), errors.New("Expected asset creation to not error"))
	}

	creditAssetInputData := []byte(fmt.Sprintf(`{"assetId" : "%s","value" : 200.30,"transactionReference" : "ra29bv7y111p945e17514","memo" :"Test credit transaction"}`, createAssetResponse1.Assets[0].ID))
	creditAssetRequest, _ := http.NewRequest("POST", test.CreditAssetEndpoint, bytes.NewBuffer(creditAssetInputData))
	creditAssetRequest.Header.Set("x-auth-token", authToken)
	creditAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(creditAssetResponse, creditAssetRequest)

	initiator := createAssetResponse1.Assets[0]
	recipient := createAssetResponse2.Assets[0]

	transferAssetInputData := []byte(fmt.Sprintf(`{"initiatorAssetId" : "%s", "recipientAssetId" : "%s","value" : 50.09,"transactionReference" : "ra29bv7y111p945e17515","memo" :"Test credit transaction"}`, initiator.ID, recipient.ID))
	transferAssetRequest, _ := http.NewRequest("POST", test.InternalTransferEndpoint, bytes.NewBuffer(transferAssetInputData))
	transferAssetRequest.Header.Set("x-auth-token", authToken)
	transferAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(transferAssetResponse, transferAssetRequest)

	// Get asset 1
	getAssetRequest1, _ := http.NewRequest("GET", fmt.Sprintf("/assets/by-id/%s", initiator.ID), bytes.NewBuffer([]byte("")))
	getAssetRequest1.Header.Set("x-auth-token", authToken)
	response1 := httptest.NewRecorder()
	s.Router.ServeHTTP(response1, getAssetRequest1)
	resBody, err = ioutil.ReadAll(response1.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAssetResponse1 := dto.Asset{}
	err = json.Unmarshal(resBody, &getAssetResponse1)
	// Get asset 2
	getAssetRequest2, _ := http.NewRequest("GET", fmt.Sprintf("/assets/by-id/%s", recipient.ID), bytes.NewBuffer([]byte("")))
	getAssetRequest2.Header.Set("x-auth-token", authToken)
	response2 := httptest.NewRecorder()
	s.Router.ServeHTTP(response2, getAssetRequest2)
	resBody, err = ioutil.ReadAll(response2.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAssetResponse2 := dto.Asset{}
	err = json.Unmarshal(resBody, &getAssetResponse2)

	if response1.Code != http.StatusOK || response2.Code != http.StatusOK || getAssetResponse1.AvailableBalance != "150.21" || getAssetResponse2.AvailableBalance != "50.09" {
		s.T().Errorf("Expected statusCode to be %d,  sender asset balance to be %s and the recipient asset balance to be %s. Got %d, %s and %s \n", http.StatusOK, "150.21", "50.09", response1.Code, getAssetResponse1.AvailableBalance, getAssetResponse2.AvailableBalance)
	}
}

func (s *Suite) Test_OnchainCreditUserAsset() {
	createAssetInputData := []byte(`{"assets" : ["BTC","ETH","BNB"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", authToken)
	createResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(createResponse, createAssetRequest)
	resBody, err := ioutil.ReadAll(createResponse.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse)
	if createResponse.Code != http.StatusCreated || len(createAssetResponse.Assets) < 1 {
		require.NoError(s.T(), errors.New("Expected asset creation to not error"))
	}

	creditAssetInputData := []byte(fmt.Sprintf(`{"assetId" : "%s","value" : 200.30,"transactionReference" : "ra29bv7y111p945e17515","memo" :"Test credit transaction"}`, createAssetResponse.Assets[0].ID))
	creditAssetRequest, _ := http.NewRequest("POST", test.CreditAssetEndpoint, bytes.NewBuffer(creditAssetInputData))
	creditAssetRequest.Header.Set("x-auth-token", authToken)
	creditAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(creditAssetResponse, creditAssetRequest)

	onchainCreditAssetInputData := []byte(fmt.Sprintf(`{"assetId" : "%s","value" : 3.441122091,"transactionReference" : "ra29bv7y111p945e17516","memo" :"Test credit transaction","chainData": {"status": true,"transactionHash": "string","transactionFee": "string","blockHeight": 0, "recipientAddress": "1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2"}}`, createAssetResponse.Assets[0].ID))
	onchainCreditAssetRequest, _ := http.NewRequest("POST", test.OnchainDepositEndpoint, bytes.NewBuffer(onchainCreditAssetInputData))
	onchainCreditAssetRequest.Header.Set("x-auth-token", authToken)
	onchainCreditAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(onchainCreditAssetResponse, onchainCreditAssetRequest)

	getAssetRequest, _ := http.NewRequest("GET", test.GetAssetEndpoint, bytes.NewBuffer([]byte("")))
	getAssetRequest.Header.Set("x-auth-token", authToken)

	response := httptest.NewRecorder()
	s.Router.ServeHTTP(response, getAssetRequest)
	resBody, err = ioutil.ReadAll(response.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &getAssetResponse)
	fmt.Printf("getAssetResponse >>> %+v", getAssetResponse)

	if len(getAssetResponse.Assets) < 1 {
		require.NoError(s.T(), errors.New("No assests returned"))
	}

	if response.Code != http.StatusOK || getAssetResponse.Assets[0].AvailableBalance != "203.741122091" {
		s.T().Errorf("Expected statusCode to be %d and asset balance to be %s. Got %d and %+v\n", http.StatusOK, "203.741122091", response.Code, getAssetResponse.Assets[0].AvailableBalance)
	}
}

func (s *Suite) Test_ProcessTransfer() {
	createAssetInputData := []byte(`{"assets" : ["BTC","ETH","BNB"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", authToken)
	createResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(createResponse, createAssetRequest)
	resBody, err := ioutil.ReadAll(createResponse.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse)
	if createResponse.Code != http.StatusCreated || len(createAssetResponse.Assets) < 1 {
		require.NoError(s.T(), errors.New("Expected asset creation to not error"))
	}

	creditAssetInputData := []byte(fmt.Sprintf(`{"assetId" : "%s","value" : 200.30,"transactionReference" : "ra29bv7y111p945e17514","memo" :"Test credit transaction"}`, createAssetResponse.Assets[0].ID))
	creditAssetRequest, _ := http.NewRequest("POST", test.CreditAssetEndpoint, bytes.NewBuffer(creditAssetInputData))
	creditAssetRequest.Header.Set("x-auth-token", authToken)
	creditAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(creditAssetResponse, creditAssetRequest)
	if creditAssetResponse.Code != http.StatusOK {
		require.NoError(s.T(), errors.New("Expected credit asset to not error"))
	}

	debitAssetInputData := []byte(fmt.Sprintf(`{"assetId" : "%s","value" : 10.30,"transactionReference" : "ra29bv7y111p945e17515","memo" :"Test credit transaction"}`, createAssetResponse.Assets[0].ID))
	debitAssetRequest, _ := http.NewRequest("POST", test.DebitAssetEndpoint, bytes.NewBuffer(debitAssetInputData))
	debitAssetRequest.Header.Set("x-auth-token", authToken)
	debitAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(debitAssetResponse, debitAssetRequest)
	if debitAssetResponse.Code != http.StatusOK {
		require.NoError(s.T(), errors.New("Expected debit asset to not error"))
	}

	externalTransferInputData := []byte(`{"recipientAddress" : "bnb1k05t5h6h7t4mq9tvafz2mx8c29jz2w4r0l0hda","value" : 10.00,"debitReference" : "ra29bv7y111p945e17515","transactionReference" : "ra29bv7y111p945e17516"}`)
	externalTransferRequest, _ := http.NewRequest("POST", test.TransferExternalEndpoint, bytes.NewBuffer(externalTransferInputData))
	externalTransferRequest.Header.Set("x-auth-token", authToken)
	externalTransferResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(externalTransferResponse, externalTransferRequest)
	if externalTransferResponse.Code != http.StatusOK {
		require.NoError(s.T(), errors.New(fmt.Sprintf("Expected external transfer asset to not error >> %+v", externalTransferResponse)))
	}

	processTransactionRequest, _ := http.NewRequest("POST", test.ProcessTransactionEndpoint, bytes.NewBuffer([]byte("")))
	processTransactionRequest.Header.Set("x-auth-token", authToken)
	response := httptest.NewRecorder()
	s.Router.ServeHTTP(response, processTransactionRequest)

	if response.Code != http.StatusOK {
		s.T().Errorf("Expected statusCode to be %d. Got %d \n", http.StatusOK, response.Code)
	}
}

func (s *Suite) Test_GetTransactionByRef() {
	createAssetInputData := []byte(`{"assets" : ["BTC","ETH","BNB"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", authToken)
	createResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(createResponse, createAssetRequest)
	resBody, err := ioutil.ReadAll(createResponse.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse)
	if createResponse.Code != http.StatusCreated || len(createAssetResponse.Assets) < 1 {
		require.NoError(s.T(), errors.New("Expected asset creation to not error"))
	}

	creditAssetInputData := []byte(fmt.Sprintf(`{"assetId" : "%s","value" : 200.30,"transactionReference" : "ra29bv7y111p945e17514","memo" :"Test credit transaction"}`, createAssetResponse.Assets[0].ID))
	creditAssetRequest, _ := http.NewRequest("POST", test.CreditAssetEndpoint, bytes.NewBuffer(creditAssetInputData))
	creditAssetRequest.Header.Set("x-auth-token", authToken)
	creditAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(creditAssetResponse, creditAssetRequest)
	if creditAssetResponse.Code != http.StatusOK {
		require.NoError(s.T(), errors.New("Expected credit asset to not error"))
	}

	getAssetTransactionRequest, _ := http.NewRequest("GET", test.GetTransactionByRef+"ra29bv7y111p945e17514", bytes.NewBuffer([]byte("")))
	getAssetTransactionRequest.Header.Set("x-auth-token", authToken)

	response := httptest.NewRecorder()
	s.Router.ServeHTTP(response, getAssetTransactionRequest)
	resBody, err = ioutil.ReadAll(response.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAssetTransactionResponse := dto.TransactionResponse{}
	err = json.Unmarshal(resBody, &getAssetTransactionResponse)

	if response.Code != http.StatusOK || getAssetTransactionResponse.Value != "200.3" || getAssetTransactionResponse.TransactionStatus != "COMPLETED" {
		s.T().Errorf("Expected statusCode to be %d and transaction value to be %s. Got %d and %s\n", http.StatusOK, "200.3", response.Code, getAssetTransactionResponse.Value)
	}
}
func (s *Suite) Test_GetTransactionsByUserId() {
	createAssetInputData := []byte(`{"assets" : ["BTC","ETH","BNB"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", authToken)
	createResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(createResponse, createAssetRequest)
	resBody, err := ioutil.ReadAll(createResponse.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse)
	if createResponse.Code != http.StatusCreated || len(createAssetResponse.Assets) < 1 {
		require.NoError(s.T(), errors.New("Expected asset creation to not error"))
	}

	creditAssetInputData := []byte(fmt.Sprintf(`{"assetId" : "%s","value" : 200.30,"transactionReference" : "ra29bv7y111p945e17514","memo" :"Test credit transaction"}`, createAssetResponse.Assets[0].ID))
	creditAssetRequest, _ := http.NewRequest("POST", test.CreditAssetEndpoint, bytes.NewBuffer(creditAssetInputData))
	creditAssetRequest.Header.Set("x-auth-token", authToken)
	creditAssetResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(creditAssetResponse, creditAssetRequest)
	if creditAssetResponse.Code != http.StatusOK {
		require.NoError(s.T(), errors.New("Expected credit asset to not error"))
	}

	getAssetTransactionsRequest, _ := http.NewRequest("GET", fmt.Sprintf("/assets/%s/transactions", createAssetResponse.Assets[0].ID), bytes.NewBuffer([]byte("")))
	getAssetTransactionsRequest.Header.Set("x-auth-token", authToken)

	response := httptest.NewRecorder()
	s.Router.ServeHTTP(response, getAssetTransactionsRequest)
	resBody, err = ioutil.ReadAll(response.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAssetTransactionsResponse := dto.TransactionListResponse{}
	err = json.Unmarshal(resBody, &getAssetTransactionsResponse)

	if response.Code != http.StatusOK || len(getAssetTransactionsResponse.Transactions) < 1 {
		s.T().Errorf("Expected statusCode to be %d and transaction length to be %d. Got %d and %d\n", http.StatusOK, 1, response.Code, len(getAssetTransactionsResponse.Transactions))
	}
}

func (s *Suite) Test_GetUserAssetByV2Address() {
	createAssetInputData := []byte(`{"assets" : ["BNB"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
	createAssetRequest, _ := http.NewRequest("POST", test.CreateAssetEndpoint, bytes.NewBuffer(createAssetInputData))
	createAssetRequest.Header.Set("x-auth-token", authToken)
	createResponse := httptest.NewRecorder()
	s.Router.ServeHTTP(createResponse, createAssetRequest)
	resBody, err := ioutil.ReadAll(createResponse.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	createAssetResponse := dto.UserAssetResponse{}
	err = json.Unmarshal(resBody, &createAssetResponse)
	if createResponse.Code != http.StatusCreated || len(createAssetResponse.Assets) < 1 {
		require.NoError(s.T(), errors.New("Expected asset creation to not error"))
	}
	// First time call to get address
	getAssetAddressRequest, _ := http.NewRequest("GET", fmt.Sprintf("/assets/%s/address?addressVersion=VERSION_2", createAssetResponse.Assets[0].ID), bytes.NewBuffer([]byte("")))
	getAssetAddressRequest.Header.Set("x-auth-token", authToken)
	responseAddress := httptest.NewRecorder()
	s.Router.ServeHTTP(responseAddress, getAssetAddressRequest)
	resBody, err = ioutil.ReadAll(responseAddress.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAssetAddressResponse := map[string]string{}
	err = json.Unmarshal(resBody, &getAssetAddressResponse)

	getAssetRequest, _ := http.NewRequest("GET", fmt.Sprintf("/assets/by-address/%s?assetSymbol=BNB&userAssetMemo=%s", getAssetAddressResponse["address"], getAssetAddressResponse["memo"]), bytes.NewBuffer([]byte("")))
	getAssetRequest.Header.Set("x-auth-token", authToken)
	response := httptest.NewRecorder()
	s.Router.ServeHTTP(response, getAssetRequest)
	resBody, err = ioutil.ReadAll(response.Body)
	if err != nil {
		require.NoError(s.T(), err)
	}
	getAssetResponse := dto.Asset{}
	err = json.Unmarshal(resBody, &getAssetResponse)

	if response.Code != http.StatusBadRequest && getAssetResponse.ID != createAssetResponse.Assets[0].ID {
		s.T().Errorf("Expected response code to be %d and asset to match. Got responseCode of %d and asset matching is %+v\n", http.StatusOK, response.Code, getAssetResponse.ID == createAssetResponse.Assets[0].ID)
	}
}
