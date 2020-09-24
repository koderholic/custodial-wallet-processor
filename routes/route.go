package routes

import (
	"net/http"
	"sync"
	"time"
	"wallet-adapter/controllers"
	"wallet-adapter/database"
	"wallet-adapter/middlewares"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/logger"
	"wallet-adapter/utility/permissions"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	httpSwagger "github.com/swaggo/http-swagger"
	validation "gopkg.in/go-playground/validator.v9"

	Config "wallet-adapter/config"
)

var (
	once sync.Once
)

// Register ... Adds router handle to general handler function
func Register(router *mux.Router, validator *validation.Validate, config Config.Data, db *gorm.DB, memoryCache *cache.Memory) {

	once.Do(func() {
		DB := database.Database{Config: config, DB: db}
		baseRepository := database.BaseRepository{Database: DB}
		userAssetRepository := database.UserAssetRepository{BaseRepository: baseRepository}
		transactionRepository := database.TransactionRepository{userAssetRepository}
		userAddressRepository := database.UserAddressRepository{userAssetRepository}
		batchRepository := database.BatchRepository{userAssetRepository}

		controller := controllers.NewController(memoryCache, config, validator, &baseRepository)
		userAssetController := controllers.NewUserAssetController(memoryCache, config, validator, &userAssetRepository)
		transactionController := controllers.NewTransactionController(memoryCache, config, validator, &transactionRepository)
		userAddressController := controllers.NewUserAddressController(memoryCache, config, validator, &userAddressRepository)
		BatchController := controllers.NewBatchController(memoryCache, config, validator, &batchRepository)

		apiRouter := router.PathPrefix("").Subrouter()
		router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		// General Routes
		apiRouter.HandleFunc("/ping", controller.Ping).Methods(http.MethodGet)

		// middleware := middlewares.NewMiddleware(config, router).ValidateAuthToken().LogAPIRequests().Timeout(requestTimeout).Build()

		// User Asset Routes
		var requestTimeout = time.Duration(config.RequestTimeout) * time.Second
		apiRouter.HandleFunc("/users/assets", middlewares.NewMiddleware(config, userAssetController.CreateUserAssets).ValidateAuthToken(permissions.All["CreateUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/users/{userId}/assets", middlewares.NewMiddleware(config, userAssetController.GetUserAssets).ValidateAuthToken(permissions.All["GetUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/credit", middlewares.NewMiddleware(config, userAssetController.CreditUserAsset).ValidateAuthToken(permissions.All["CreditUserAsset"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/onchain-deposit", middlewares.NewMiddleware(config, userAssetController.OnChainCreditUserAsset).ValidateAuthToken(permissions.All["OnChainDeposit"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/debit", middlewares.NewMiddleware(config, userAssetController.DebitUserAsset).ValidateAuthToken(permissions.All["DebitUserAsset"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/transfer-internal", middlewares.NewMiddleware(config, userAssetController.InternalTransfer).ValidateAuthToken(permissions.All["InternalTransfer"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/by-id/{assetId}", middlewares.NewMiddleware(config, userAssetController.GetUserAssetById).ValidateAuthToken(permissions.All["GetUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/by-address/{address}", middlewares.NewMiddleware(config, userAssetController.GetUserAssetByAddress).ValidateAuthToken(permissions.All["GetUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/{assetId}/address", middlewares.NewMiddleware(config, userAddressController.GetAssetAddress).ValidateAuthToken(permissions.All["GetAssetAddress"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/{assetId}/all-addresses", middlewares.NewMiddleware(config, userAddressController.GetAllAssetAddresses).ValidateAuthToken(permissions.All["GetAssetAddress"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/transactions/{reference}", middlewares.NewMiddleware(config, transactionController.GetTransaction).ValidateAuthToken(permissions.All["GetTransaction"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/{assetId}/transactions", middlewares.NewMiddleware(config, transactionController.GetTransactionsByAssetId).ValidateAuthToken(permissions.All["GetTransaction"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/transfer-external", middlewares.NewMiddleware(config, transactionController.ExternalTransfer).ValidateAuthToken(permissions.All["ExternalTransfer"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/confirm-transaction", middlewares.NewMiddleware(config, transactionController.ConfirmTransaction).ValidateAuthToken(permissions.All["ConfirmTransaction"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/process-transaction", middlewares.NewMiddleware(config, transactionController.ProcessTransactions).LogAPIRequests().Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/process-batched-transactions", middlewares.NewMiddleware(config, BatchController.ProcessBatchBTCTransactions).LogAPIRequests().Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/trigger-float-manager", middlewares.NewMiddleware(config, userAssetController.TriggerFloat).ValidateAuthToken(permissions.All["TriggerFloat"]).LogAPIRequests().Build()).Methods(http.MethodPost)

	})

	logger.Info("App routes registered successfully!")
}
