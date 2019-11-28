package app

import (
	"net/http"
	"sync"
	"wallet-adapter/controllers"
	"wallet-adapter/database"
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
func RegisterRoutes(router *mux.Router, validator *validation.Validate, config Config.Data, logger *utility.Logger, db *gorm.DB) {

	once.Do(func() {
		DB := database.Database{Logger: logger, Config: config, DB: db}
		baseRepository := database.BaseRepository{Database: DB}
		userAssetRepository := database.UserAssetRepository{BaseRepository: baseRepository}

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
