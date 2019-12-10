package test

import (
	"net/http"
	"sync"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/middlewares"
	"wallet-adapter/utility"

	"wallet-adapter/controllers"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	httpSwagger "github.com/swaggo/http-swagger"
	validation "gopkg.in/go-playground/validator.v9"
)

var (
	once sync.Once
)

func startUp() (Config.Data, *utility.Logger, http.Handler) {

	config := Config.Data{}
	config.Init("")

	logger := utility.NewLogger()
	router := mux.NewRouter()
	validator := validation.New()

	Database := &database.Database{
		Logger: logger,
		Config: config,
	}
	Database.LoadDBInstance()
	defer Database.CloseDBInstance()
	Database.RunDbMigrations()

	RegisterRoutes(router, validator, config, logger, Database.DB)

	middleware := middlewares.NewMiddleware(logger, config, router).ValidateAuthToken().LogAPIRequests().Build()

	return config, logger, middleware
}

var config, logger, router = startUp()

func RegisterRoutes(router *mux.Router, validator *validation.Validate, config Config.Data, logger *utility.Logger, db *gorm.DB) {

	once.Do(func() {
		baseRepository := BaseRepository{Logger: logger, Config: config, DB: db}
		userAssetRepository := UserAssetRepository{BaseRepository: baseRepository}

		controller := controllers.NewController(logger, config, validator, &baseRepository)
		userAssetController := controllers.NewUserAssetController(logger, config, validator, &userAssetRepository)

		basePath := config.BasePath

		apiRouter := router.PathPrefix(basePath).Subrouter()
		router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		// General Routes
		apiRouter.HandleFunc("/crypto/ping", controller.Ping).Methods(http.MethodGet)

		// Asset Routes
		apiRouter.HandleFunc("/crypto/assets", controller.FetchAllAssets).Methods(http.MethodGet)
		apiRouter.HandleFunc("/crypto/assets/supported", controller.FetchSupportedAssets).Methods(http.MethodGet)
		apiRouter.HandleFunc("/crypto/assets/{assetId}", controller.GetAsset).Methods(http.MethodGet)

		// User Asset Routes
		apiRouter.HandleFunc("/crypto/users/assets", userAssetController.CreateUserAssets).Methods(http.MethodPost)
		apiRouter.HandleFunc("/crypto/users/{userId}/assets", userAssetController.GetUserAssets).Methods(http.MethodGet)

	})

	logger.Info("App routes registered successfully!")
}
