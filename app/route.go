package app

import (
	"net/http"
	"sync"
	"time"
	"wallet-adapter/controllers"
	"wallet-adapter/database"
	"wallet-adapter/middlewares"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	httpSwagger "github.com/swaggo/http-swagger"
	validation "gopkg.in/go-playground/validator.v9"

	Config "wallet-adapter/config"
)

var (
	once sync.Once
)

// RegisterRoutes ... Adds router handle to general handler function
func RegisterRoutes(router *mux.Router, validator *validation.Validate, config Config.Data, logger *utility.Logger, db *gorm.DB, memoryCache *utility.MemoryCache) {

	once.Do(func() {
		DB := database.Database{Logger: logger, Config: config, DB: db}
		baseRepository := database.BaseRepository{Database: DB}
		userAssetRepository := database.UserAssetRepository{BaseRepository: baseRepository}
		batchRepository := database.BatchRepository{BaseRepository: baseRepository}

		controller := controllers.NewController(memoryCache, logger, config, validator, &baseRepository)
		userAssetController := controllers.NewUserAssetController(memoryCache, logger, config, validator, &userAssetRepository)
		BatchController := controllers.NewBatchController(memoryCache, logger, config, validator, &batchRepository)

		apiRouter := router.PathPrefix("").Subrouter()
		router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		// General Routes
		apiRouter.HandleFunc("/ping", controller.Ping).Methods(http.MethodGet)

		// middleware := middlewares.NewMiddleware(logger, config, router).ValidateAuthToken().LogAPIRequests().Timeout(requestTimeout).Build()

		// User Asset Routes
		var requestTimeout = time.Duration(config.RequestTimeout) * time.Second
		apiRouter.HandleFunc("/users/assets", middlewares.NewMiddleware(logger, config, userAssetController.CreateUserAssets).ValidateAuthToken(utility.Permissions["CreateUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/users/{userId}/assets", middlewares.NewMiddleware(logger, config, userAssetController.GetUserAssets).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/credit", middlewares.NewMiddleware(logger, config, userAssetController.CreditUserAsset).ValidateAuthToken(utility.Permissions["CreditUserAsset"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/onchain-deposit", middlewares.NewMiddleware(logger, config, userAssetController.OnChainCreditUserAsset).ValidateAuthToken(utility.Permissions["OnChainDeposit"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/debit", middlewares.NewMiddleware(logger, config, userAssetController.DebitUserAsset).ValidateAuthToken(utility.Permissions["DebitUserAsset"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/transfer-internal", middlewares.NewMiddleware(logger, config, userAssetController.InternalTransfer).ValidateAuthToken(utility.Permissions["InternalTransfer"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/by-id/{assetId}", middlewares.NewMiddleware(logger, config, userAssetController.GetUserAssetById).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/by-address/{address}", middlewares.NewMiddleware(logger, config, userAssetController.GetUserAssetByAddress).ValidateAuthToken(utility.Permissions["GetUserAssets"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/{assetId}/address", middlewares.NewMiddleware(logger, config, userAssetController.GetAssetAddress).ValidateAuthToken(utility.Permissions["GetAssetAddress"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/{assetId}/all-addresses", middlewares.NewMiddleware(logger, config, userAssetController.GetAllAssetAddresses).ValidateAuthToken(utility.Permissions["GetAssetAddress"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/transactions/{reference}", middlewares.NewMiddleware(logger, config, userAssetController.GetTransaction).ValidateAuthToken(utility.Permissions["GetTransaction"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/{assetId}/transactions", middlewares.NewMiddleware(logger, config, userAssetController.GetTransactionsByAssetId).ValidateAuthToken(utility.Permissions["GetTransaction"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/transfer-external", middlewares.NewMiddleware(logger, config, userAssetController.ExternalTransfer).ValidateAuthToken(utility.Permissions["ExternalTransfer"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/confirm-transaction", middlewares.NewMiddleware(logger, config, userAssetController.ConfirmTransaction).ValidateAuthToken(utility.Permissions["ConfirmTransaction"]).LogAPIRequests().Timeout(requestTimeout).Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/process-transaction", middlewares.NewMiddleware(logger, config, userAssetController.ProcessTransactions).LogAPIRequests().Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/assets/process-batched-transactions", middlewares.NewMiddleware(logger, config, BatchController.ProcessBatchBTCTransactions).LogAPIRequests().Build()).Methods(http.MethodPost)
		apiRouter.HandleFunc("/trigger-float-manager", middlewares.NewMiddleware(logger, config, userAssetController.TriggerFloat).ValidateAuthToken(utility.Permissions["TriggerFloat"]).LogAPIRequests().Build()).Methods(http.MethodPost)

	})

	logger.Info("App routes registered successfully!")
}
