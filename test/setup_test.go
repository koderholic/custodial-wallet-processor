package test

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
	"wallet-adapter/config"
	"wallet-adapter/controllers"
	"wallet-adapter/database"
	"wallet-adapter/middlewares"
	"wallet-adapter/model"
	"wallet-adapter/utility"
	"wallet-adapter/utility/logger"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/suite"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	httpSwagger "github.com/swaggo/http-swagger"
	validation "gopkg.in/go-playground/validator.v9"
)

//Suite ...
type Suite struct {
	suite.Suite
	DB       *gorm.DB
	Database database.Database
	Config   config.Data
	Router   *mux.Router
}

var (
	once                    sync.Once
	purgeInterval           = 5 * time.Second
	cacheDuration           = 60 * time.Second
	authCache               = utility.InitializeCache(cacheDuration, purgeInterval)
	authToken               = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJTVkNTL0FVVEgiLCJwZXJtaXNzaW9ucyI6WyJzdmNzLmNyeXB0by13YWxsZXQtYWRhcHRlci5jcmVkaXQtYXNzZXQiLCJzdmNzLmNyeXB0by13YWxsZXQtYWRhcHRlci5nZXQtYXNzZXRzIiwic3Zjcy5jcnlwdG8td2FsbGV0LWFkYXB0ZXIuY3JlYXRlLWFzc2V0cyIsInN2Y3MuY3J5cHRvLXdhbGxldC1hZGFwdGVyLmNyZWRpdC1hc3NldCIsInN2Y3MuY3J5cHRvLXdhbGxldC1hZGFwdGVyLmRlYml0LWFzc2V0Iiwic3Zjcy5jcnlwdG8td2FsbGV0LWFkYXB0ZXIuZG8taW50ZXJuYWwtdHJhbnNmZXIiLCJzdmNzLmNyeXB0by13YWxsZXQtYWRhcHRlci5nZXQtYWRkcmVzcyIsInN2Y3MuY3J5cHRvLXdhbGxldC1hZGFwdGVyLmdldC10cmFuc2FjdGlvbnMiLCJzdmNzLmNyeXB0by13YWxsZXQtYWRhcHRlci5vbi1jaGFpbi1kZXBvc2l0Iiwic3Zjcy5jcnlwdG8td2FsbGV0LWFkYXB0ZXIuY29uZmlybS10cmFuc2FjdGlvbiIsInN2Y3MuY3J5cHRvLXdhbGxldC1hZGFwdGVyLmRvLWV4dGVybmFsLXRyYW5zZmVyIiwic3Zjcy5jcnlwdG8td2FsbGV0LWFkYXB0ZXIucHJvY2Vzcy10cmFuc2FjdGlvbnMiXSwic2VydmljZUlkIjoiNzZhYTcyZjctYjAwZS00OWRhLTgwN2ItNzVjZGUyZjEwZTI3IiwidG9rZW5UeXBlIjoiU0VSVklDRSJ9.ImOiJYkjwGG5_-E4FDUO3LRKZFDLxv3WLpgDt__Ih42B4jUlJ7pl4YJPfSJBc0vM1A57fjuPdJ8NhCd0wcIkxOuDDXJuon5xE1NIr0muIbPWQjNtpkgcVy9gSYBgHAERAFNkSIV_GWvki06uIT0DoQviWTWZmwuG112jquRpfyYV8M5l2pE-xtpf75quQBQQU08EEA-dS17iR4VaaTiCD584o9ujO-Wql9PBs8NK5g1kBpqpOWj2jIpa0NQSYlwijOw2cKL91KpTS0xxG1AXMzvyOyQK-QVpTX09tJrqsmzYHH49Zg5AlaTmiHbsSDhxacdiIE7O_Ge0T1B6PC_SLA"
	testUserId1, _          = uuid.FromString("a10fce7b-7844-43af-9ed1-e130723a1ea3")
	testUserId2, _          = uuid.FromString("ff365b4d-6e56-4df7-b0ed-1c5ce325f6e2")
	testUserAssets1         []model.UserAsset
	testUserAssets2         []model.UserAsset
	testUserAssets1Ids      []uuid.UUID
	testUserAssets2Ids      []uuid.UUID
	testDenominations       = []model.Denomination{}
	testUserAssetRepository database.UserAssetRepository
	testUserAddresses       []model.UserAddress
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
	db.LogMode(true)

	s.DB = db
	require.NoError(s.T(), err)

	if err = os.Chmod(dir+"/walletAdapter.db", 0777); err != nil {
		require.NoError(s.T(), err)
	}

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
	}

	Database := database.Database{
		Config: Config,
		DB:     s.DB,
	}

	s.Database = Database
	s.Config = Config
	s.Router = router

	testUserAssetRepository = database.UserAssetRepository{
		BaseRepository: database.BaseRepository{
			Database: database.Database{
				Config: s.Config,
				DB:     s.DB,
			},
		},
	}

	s.RegisterRoutes(Config, router, validator)
}

func (s *Suite) SetupTest() {
	s.RunMigration()
	s.DBSeeder()
}

func (s *Suite) TearDownTest() {
	s.DB.DropTableIfExists(&model.Denomination{}, &model.BatchRequest{}, &model.ChainTransaction{}, &model.Transaction{}, &model.UserAddress{}, &model.UserAsset{}, &model.HotWalletAsset{}, &model.TransactionQueue{})
}

// RegisterRoutes ...
func (s *Suite) RegisterRoutes(Config config.Data, router *mux.Router, validator *validation.Validate) {

	once.Do(func() {

		baseRepository := database.BaseRepository{Database: s.Database}
		userAssetRepository := database.UserAssetRepository{BaseRepository: baseRepository}
		userAssetController := controllers.NewUserAssetController(authCache, s.Config, validator, &userAssetRepository)
		apiRouter := router.PathPrefix("").Subrouter()
		router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		// User Asset Routes
		var requestTimeout = time.Duration(s.Config.RequestTimeout) * time.Second
		apiRouter.HandleFunc("/users/assets", middlewares.NewMiddleware(s.Config, userAssetController.CreateUserAssets).ValidateAuthToken(utility.Permissions["CreateUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/users/{userId}/assets", middlewares.NewMiddleware(s.Config, userAssetController.GetUserAssets).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/credit", middlewares.NewMiddleware(s.Config, userAssetController.CreditUserAsset).ValidateAuthToken(utility.Permissions["CreditUserAsset"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/onchain-deposit", middlewares.NewMiddleware(s.Config, userAssetController.OnChainCreditUserAsset).ValidateAuthToken(utility.Permissions["OnChainDeposit"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/debit", middlewares.NewMiddleware(s.Config, userAssetController.DebitUserAsset).ValidateAuthToken(utility.Permissions["DebitUserAsset"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/transfer-internal", middlewares.NewMiddleware(s.Config, userAssetController.InternalTransfer).ValidateAuthToken(utility.Permissions["InternalTransfer"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/by-id/{assetId}", middlewares.NewMiddleware(s.Config, userAssetController.GetUserAssetById).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/by-address/{address}", middlewares.NewMiddleware(s.Config, userAssetController.GetUserAssetByAddress).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/{assetId}/address", middlewares.NewMiddleware(s.Config, userAssetController.GetAssetAddress).ValidateAuthToken(utility.Permissions["GetAssetAddress"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/transactions/{reference}", middlewares.NewMiddleware(s.Config, userAssetController.GetTransaction).ValidateAuthToken(utility.Permissions["GetTransaction"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/{assetId}/transactions", middlewares.NewMiddleware(s.Config, userAssetController.GetTransactionsByAssetId).ValidateAuthToken(utility.Permissions["GetTransaction"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/transfer-external", middlewares.NewMiddleware(s.Config, userAssetController.ExternalTransfer).ValidateAuthToken(utility.Permissions["ExternalTransfer"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/confirm-transaction", middlewares.NewMiddleware(s.Config, userAssetController.ConfirmTransaction).ValidateAuthToken(utility.Permissions["ConfirmTransaction"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/process-transaction", middlewares.NewMiddleware(s.Config, userAssetController.ProcessTransactions).LogAPIRequests().Build()).Methods(http.MethodPost)

	})
}

// RunDbMigrations ... This creates corresponding tables for dtos on the db for testing
func (s *Suite) RunMigration() {
	s.DB.AutoMigrate(&model.Denomination{}, &model.BatchRequest{}, &model.SharedAddress{}, &model.ChainTransaction{}, &model.Transaction{}, &model.UserAddress{}, &model.UserAsset{}, &model.HotWalletAsset{}, &model.TransactionQueue{})
}

// DBSeeder .. This seeds supported assets into the database for testing
func (s *Suite) DBSeeder() {
	isToken := true
	isNonNative := false

	testDenominations = []model.Denomination{
		{Name: "Binance Coin", AssetSymbol: "BNB", CoinType: 714, Decimal: 8, IsEnabled: true, IsToken: &isNonNative, MainCoinAssetSymbol: "BNB", SweepFee: 37500, TradeActivity: "ACTIVE", DepositActivity: "ACTIVE", WithdrawActivity: "ACTIVE", TransferActivity: "ACTIVE"},
		{Name: "Binance USD", AssetSymbol: "BUSD", CoinType: 714, Decimal: 8, IsEnabled: true, IsToken: &isToken, MainCoinAssetSymbol: "BNB", SweepFee: 37500, TradeActivity: "ACTIVE", DepositActivity: "ACTIVE", WithdrawActivity: "ACTIVE", TransferActivity: "ACTIVE"},
		{Name: "Ethereum Coin", AssetSymbol: "ETH", CoinType: 60, Decimal: 18, IsEnabled: true, IsToken: &isNonNative, MainCoinAssetSymbol: "ETH", TradeActivity: "ACTIVE", DepositActivity: "ACTIVE", WithdrawActivity: "ACTIVE", TransferActivity: "ACTIVE"},
		{Name: "Bitcoin", AssetSymbol: "BTC", CoinType: 0, Decimal: 8, IsEnabled: true, IsToken: &isNonNative, MainCoinAssetSymbol: "BTC", TradeActivity: "ACTIVE", DepositActivity: "ACTIVE", WithdrawActivity: "ACTIVE", TransferActivity: "ACTIVE"},
		{Name: "ChainLink", AssetSymbol: "LINK", CoinType: 60, Decimal: 18, IsEnabled: true, IsToken: &isToken, MainCoinAssetSymbol: "ETH", TradeActivity: "ACTIVE", DepositActivity: "NONE", WithdrawActivity: "NONE", TransferActivity: "NONE"},
	}

	denominationsId := []uuid.UUID{}
	for _, asset := range testDenominations {
		if err := s.DB.FirstOrCreate(&asset, model.Denomination{AssetSymbol: asset.AssetSymbol}).Error; err != nil {
			logger.Error("Error with creating asset record %s : %s", asset.AssetSymbol, err)
		}
		denominationsId = append(denominationsId, asset.ID)
	}

	userAssets1 := []model.UserAsset{
		{
			UserID:         testUserId1,
			DenominationID: denominationsId[0],
		},
		{
			UserID:         testUserId1,
			DenominationID: denominationsId[1],
		},
		{
			UserID:         testUserId1,
			DenominationID: denominationsId[2],
		},
		{
			UserID:         testUserId1,
			DenominationID: denominationsId[3],
		},
		{
			UserID:         testUserId1,
			DenominationID: denominationsId[4],
		},
	}
	userAssets2 := []model.UserAsset{
		{
			UserID:         testUserId2,
			DenominationID: denominationsId[0],
		},
		{
			UserID:         testUserId2,
			DenominationID: denominationsId[1],
		},
		{
			UserID:         testUserId2,
			DenominationID: denominationsId[2],
		},
		{
			UserID:         testUserId2,
			DenominationID: denominationsId[3],
		},
		{
			UserID:         testUserId2,
			DenominationID: denominationsId[4],
		},
	}

	for _, asset := range userAssets1 {
		if err := s.DB.Create(&asset).Error; err != nil {
			logger.Error(fmt.Sprintf("Error with creating user asset record for %v : %s", asset.UserID, err))
		}
		testUserAssets1 = append(testUserAssets1, asset)
		testUserAssets1Ids = append(testUserAssets1Ids, asset.ID)
	}

	for _, asset := range userAssets2 {
		if err := s.DB.Create(&asset).Error; err != nil {
			logger.Error(fmt.Sprintf("Error with creating user asset record for %v : %s", asset.UserID, err))
		}
		testUserAssets2 = append(testUserAssets2, asset)
		testUserAssets2Ids = append(testUserAssets2Ids, asset.ID)
	}

	testUserAddresses = []model.UserAddress{
		{
			AssetID:   testUserAssets1[0].ID,
			V2Address: "bnb10f7jqrvg3d978cgtsqydtlk20y992yeapjzd3a",
			Memo:      "639469678",
			IsValid:   true,
		},
		{
			AssetID:   testUserAssets1[1].ID,
			V2Address: "bnb10f7jqrvg3d978cgtsqydtlk20y992yeapjzd3a",
			Memo:      "639469678",
			IsValid:   true,
		},
		{
			AssetID: testUserAssets1[2].ID,
			Address: "0xce4B800c0aB49Dda535BCe18F87f81D13f142A3C",
			IsValid: true,
		},
		{
			AssetID:     testUserAssets1[3].ID,
			Address:     "1F824Xzdnv3bu29npK7ZZaN9aPAnN31kaD",
			AddressType: "Legacy",
			IsValid:     true,
		},
		{
			AssetID:     testUserAssets1[3].ID,
			Address:     "bc1q2fv8dmp3hdeu49azalvwh9w7dd8wvw2jl62l6m",
			AddressType: "Segwit",
			IsValid:     true,
		},
	}
	for _, userAddress := range testUserAddresses {
		if err := s.DB.Create(&userAddress).Error; err != nil {
			logger.Error(fmt.Sprintf("Error with creating asset address record for %v : %s", userAddress.AssetID, err))
		}
	}

	testSharedAddress := model.SharedAddress{
		UserId:      testUserId1,
		Address:     "bnb10f7jqrvg3d978cgtsqydtlk20y992yeapjzd3a",
		AssetSymbol: "BNB",
		CoinType:    714,
	}
	if err := s.DB.Create(&testSharedAddress).Error; err != nil {
		logger.Error(fmt.Sprintf("Error with creating shared address record for %v : %s", testSharedAddress.Address, err))
	}

	logger.Info("Supported assets seeded successfully")
}
