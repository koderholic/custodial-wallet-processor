package test

// import (
// 	"net/http"
// 	"sync"
// 	"testing"
// 	"time"
// 	Config "wallet-adapter/config"
// 	config "wallet-adapter/config"
// 	"wallet-adapter/controllers"
// 	"wallet-adapter/database"
// 	"wallet-adapter/dto"
// 	"wallet-adapter/middlewares"
// 	"wallet-adapter/utility"

// 	"github.com/gorilla/mux"
// 	"github.com/stretchr/testify/require"
// 	"github.com/stretchr/testify/suite"

// 	"github.com/DATA-DOG/go-sqlmock"

// 	"github.com/jinzhu/gorm"
// 	_ "github.com/jinzhu/gorm/dialects/sqlite"

// 	httpSwagger "github.com/swaggo/http-swagger"
// 	validation "gopkg.in/go-playground/validator.v9"
// )

// type Test struct {
// 	pingEndpoint           string
// 	CreateAssetEndpoint    string
// 	GetAssetEndpoint       string
// 	CreditAssetEndpoint    string
// 	DebitAssetEndpoint     string
// 	GetAddressEndpoint     string
// 	GetTransactionByRef    string
// 	GetTransactionsByAsset string
// }

// var test = Test{
// 	pingEndpoint:           "/ping",
// 	CreateAssetEndpoint:    "/users/assets",
// 	GetAssetEndpoint:       "/users/dbd77a9f-0dd9-4ff0-b17b-430e3895b82f/assets",
// 	CreditAssetEndpoint:    "/assets/credit",
// 	DebitAssetEndpoint:     "/assets/debit",
// 	GetAddressEndpoint:     "/assets/a10fce7b-7844-43af-9ed1-e130723a1ea3/address",
// 	GetTransactionByRef:    "/assets/transactions/9b7227pba3d915ef756a",
// 	GetTransactionsByAsset: "/assets/a10fce7b-7844-43af-9ed1-e130723a1ea3/transactions",
// }

// //BaseController : Base controller struct
// type Controller struct {
// 	Logger     *utility.Logger
// 	Config     Config.Data
// 	Validator  *validation.Validate
// 	Repository database.IRepository
// }

// //Suite ...
// type Suite struct {
// 	suite.Suite
// 	DB       *gorm.DB
// 	Mock     sqlmock.Sqlmock
// 	Database database.Database
// 	Logger   *utility.Logger
// 	Config   config.Data
// 	Router   *mux.Router
// }

// var (
// 	once sync.Once
// )

// var purgeInterval = 5 * time.Second
// var cacheDuration = 60 * time.Second
// var authCache = utility.InitializeCache(cacheDuration, purgeInterval)

// func TestInit(t *testing.T) {
// 	suite.Run(t, new(Suite))
// }

// // SetupSuite ...
// func (s *Suite) SetupSuite() {

// 	db, err := gorm.Open("sqlite3", "./walletAdapter.db")
// 	s.DB = db
// 	require.NoError(s.T(), err)
// 	s.DB.LogMode(true)

// 	logger := utility.NewLogger()
// 	router := mux.NewRouter()
// 	validator := validation.New()
// 	Config := config.Data{
// 		AppPort:               "9000",
// 		ServiceName:           "crypto-wallet-adapter",
// 		AuthenticatorKey:      "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUE0ZjV3ZzVsMmhLc1RlTmVtL1Y0MQpmR25KbTZnT2Ryajh5bTNyRmtFVS93VDhSRHRuU2dGRVpPUXBIRWdRN0pMMzh4VWZVMFkzZzZhWXc5UVQwaEo3Cm1DcHo5RXI1cUxhTVhKd1p4ekh6QWFobGZBMGljcWFidkpPTXZRdHpENnVRdjZ3UEV5WnREVFdpUWk5QVh3QnAKSHNzUG5wWUdJbjIwWlp1TmxYMkJyQ2xjaUhoQ1BVSUlaT1FuL01tcVREMzFqU3lqb1FvVjdNaGhNVEFUS0p4MgpYckhoUisxRGNLSnpRQlNUQUducFlWYXFwc0FSYXArbndSaXByM25VVHV4eUdvaEJUU21qSjJ1c1NlUVhISTNiCk9ESVJlMUF1VHlIY2VBYmV3bjhiNDYyeUVXS0FSZHBkOUFqUVc1U0lWUGZkc3o1QjZHbFlRNUxkWUt0em5UdXkKN3dJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0t",
// 		PurgeCacheInterval:    60,
// 		ServiceID:             "4b0bde7a-9201-4cf9-859f-e61d976e376d",
// 		ServiceKey:            "32e1f6396de342e879ca07ec68d4d907",
// 		AuthenticationService: "https://internal.dev.bundlewallet.com/authentication",
// 		KeyManagementService:  "https://internal.dev.bundlewallet.com/key-management",
// 		CryptoAdapterService:  "https://internal.dev.bundlewallet.com/crypto-adapter",
// 		LockerService:         "https://internal.dev.bundlewallet.com/locker",
// 		ExpireCacheDuration:   400,
// 		RequestTimeout:        60,
// 		MaxIdleConns:          25,
// 		MaxOpenConns:          50,
// 		ConnMaxLifetime:       300,
// 		LockerPrefix:          "Wallet-Adapter-Lock-",
// 	}

// 	Database := database.Database{
// 		Logger: logger,
// 		Config: Config,
// 		DB:     s.DB,
// 	}

// 	s.Database = Database
// 	s.Logger = logger
// 	s.Config = Config
// 	s.Router = router

// 	s.RunMigration()
// 	s.DBSeeder()
// 	s.RegisterRoutes(logger, Config, router, validator)
// }

// // func (s *Suite) TearDownTestSuite() {
// // 	err := os.Remove("./walletAdapter.db")
// // 	if err != nil {
// // 		s.Logger.Error("Error with deleting the test database : ", err)
// // 	}
// // }

// // RegisterRoutes ...
// func (s *Suite) RegisterRoutes(logger *utility.Logger, Config config.Data, router *mux.Router, validator *validation.Validate) {

// 	once.Do(func() {

// 		baseRepository := database.BaseRepository{Database: s.Database}
// 		userAssetRepository := database.UserAssetRepository{BaseRepository: baseRepository}

// 		controller := controllers.NewController(authCache, s.Logger, s.Config, validator, &baseRepository)
// 		userAssetController := controllers.NewUserAssetController(authCache, s.Logger, s.Config, validator, &userAssetRepository)

// 		apiRouter := router.PathPrefix("").Subrouter()
// 		router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

// 		// User Asset Routes
// 		apiRouter.HandleFunc("/users/assets", middlewares.NewMiddleware(logger, Config, userAssetController.CreateUserAssets).ValidateAuthToken(utility.Permissions["CreateUserAssets"]).LogAPIRequests().Build()).Methods(http.MethodPost)
// 		apiRouter.HandleFunc("/users/{userId}/assets", middlewares.NewMiddleware(logger, Config, userAssetController.GetUserAssets).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Build()).Methods(http.MethodGet)
// 		apiRouter.HandleFunc("/assets/credit", middlewares.NewMiddleware(logger, Config, userAssetController.CreditUserAsset).ValidateAuthToken(utility.Permissions["CreditUserAsset"]).LogAPIRequests().Build()).Methods(http.MethodPost)
// 		apiRouter.HandleFunc("/assets/debit", middlewares.NewMiddleware(logger, Config, userAssetController.DebitUserAsset).ValidateAuthToken(utility.Permissions["DebitUserAsset"]).LogAPIRequests().Build()).Methods(http.MethodPost)
// 		apiRouter.HandleFunc("/assets/transfer-internal", middlewares.NewMiddleware(logger, Config, userAssetController.InternalTransfer).ValidateAuthToken(utility.Permissions["InternalTransfer"]).LogAPIRequests().Build()).Methods(http.MethodPost)
// 		apiRouter.HandleFunc("/assets/by-id/{assetId}", middlewares.NewMiddleware(logger, Config, userAssetController.GetUserAssetById).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Build()).Methods(http.MethodGet)
// 		apiRouter.HandleFunc("/assets/by-address/{address}", middlewares.NewMiddleware(logger, Config, userAssetController.GetUserAssetByAddress).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Build()).Methods(http.MethodGet)
// 		apiRouter.HandleFunc("/assets/{assetId}/address", middlewares.NewMiddleware(logger, Config, userAssetController.GetAssetAddress).ValidateAuthToken(utility.Permissions["GetAssetAddress"]).LogAPIRequests().Build()).Methods(http.MethodGet)
// 		apiRouter.HandleFunc("/assets/transactions/{reference}", middlewares.NewMiddleware(logger, Config, controller.GetTransaction).ValidateAuthToken(utility.Permissions["GetTransaction"]).LogAPIRequests().Build()).Methods(http.MethodGet)
// 		apiRouter.HandleFunc("/assets/{assetId}/transactions", middlewares.NewMiddleware(logger, Config, controller.GetTransactionsByAssetId).ValidateAuthToken(utility.Permissions["GetTransaction"]).LogAPIRequests().Build()).Methods(http.MethodGet)

// 	})
// }

// // RunDbMigrations ... This creates corresponding tables for dtos on the db for testing
// func (s *Suite) RunMigration() {
// 	s.DB.AutoMigrate(&dto.Denomination{}, &dto.BatchRequest{}, &dto.ChainTransaction{}, &dto.Transaction{}, &dto.UserAddress{}, &dto.UserAsset{}, &dto.HotWalletAsset{}, &dto.TransactionQueue{})
// 	s.DB.Model(&dto.UserAsset{}).AddForeignKey("denomination_id", "denominations(id)", "CASCADE", "CASCADE")
// 	s.DB.Model(&dto.UserAddress{}).AddForeignKey("asset_id", "user_assets(id)", "CASCADE", "CASCADE")
// 	s.DB.Model(&dto.Transaction{}).AddForeignKey("recipient_id", "user_assets(id)", "CASCADE", "CASCADE")
// 	s.DB.Model(&dto.TransactionQueue{}).AddForeignKey("transaction_id", "transactions(id)", "CASCADE", "CASCADE")
// 	s.DB.Model(&dto.TransactionQueue{}).AddForeignKey("debit_reference", "transactions(transaction_reference)", "NO ACTION", "NO ACTION")
// }

// // DBSeeder .. This seeds supported assets into the database for testing
// func (s *Suite) DBSeeder() {

// 	assets := []dto.Denomination{
// 		dto.Denomination{Name: "Binance Coin", AssetSymbol: "BNB", CoinType: 714, Decimal: 8},
// 		dto.Denomination{Name: "Ethereum Coin", AssetSymbol: "ETH", CoinType: 60, Decimal: 18},
// 		dto.Denomination{Name: "Bitcoin", AssetSymbol: "BTC", CoinType: 0, Decimal: 8},
// 	}

// 	for _, asset := range assets {
// 		if err := s.DB.FirstOrCreate(&asset, dto.Denomination{AssetSymbol: asset.AssetSymbol}).Error; err != nil {
// 			s.Logger.Error("Error with creating asset record %s : %s", asset.AssetSymbol, err)
// 		}
// 	}
// 	s.Logger.Info("Supported assets seeded successfully")
// }
