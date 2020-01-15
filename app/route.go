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

		apiRouter := router.PathPrefix("/crypto").Subrouter()
		router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		// General Routes
		apiRouter.HandleFunc("/ping", controller.Ping).Methods(http.MethodGet)

		// User Asset Routes
		apiRouter.HandleFunc("/users/assets", userAssetController.CreateUserAssets).Methods(http.MethodPost)
		apiRouter.HandleFunc("/users/{userId}/assets", userAssetController.GetUserAssets).Methods(http.MethodGet)
		apiRouter.HandleFunc("/assets/credit", userAssetController.CreditUserAssets).Methods(http.MethodPost)

	})

	logger.Info("App routes registered successfully!")
}
