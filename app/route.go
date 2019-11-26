package app

import (
	"net/http"
	"sync"
	"wallet-adapter/controllers"
	"wallet-adapter/database"

	httpSwagger "github.com/swaggo/http-swagger"
)

var (
	once sync.Once
)

// RegisterRoutes ... Adds router handle to general handler function
func (app *App) RegisterRoutes() {

	once.Do(func() {
		db := database.Database{Logger: app.Logger, Config: app.Config, DB: app.DB}
		baseRepository := database.BaseRepository{Database: db}
		assetRepository := database.AssetRepository{BaseRepository: baseRepository}
		userAssetRepository := database.UserAssetRepository{BaseRepository: baseRepository}

		controller := controllers.NewController(app.Logger, app.Config, &baseRepository)
		assetController := controllers.NewAssetController(app.Logger, app.Config, &assetRepository)
		userAssetController := controllers.NewUserAssetController(app.Logger, app.Config, &userAssetRepository)

		baseURL := "/api/v1"

		apiRouter := app.Router.PathPrefix(baseURL).Subrouter()
		app.Router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		// General Routes
		apiRouter.HandleFunc("/crypto/ping", controller.Ping).Methods(http.MethodGet)

		// Asset Routes
		apiRouter.HandleFunc("/crypto/assets", assetController.FetchAllAssets).Methods(http.MethodGet)
		apiRouter.HandleFunc("/crypto/assets/supported", assetController.FetchSupportedAssets).Methods(http.MethodGet)
		apiRouter.HandleFunc("/crypto/assets/{assetId}", assetController.GetAsset).Methods(http.MethodGet)

		// User Asset Routes
		apiRouter.HandleFunc("/crypto/users/{userId}/create-assets", userAssetController.CreateUserAssets).Methods(http.MethodPost)
		apiRouter.HandleFunc("/crypto/users/{userId}/assets", userAssetController.GetUserAssets).Methods(http.MethodGet)

	})

	app.Logger.Info("App routes registered successfully!")
}
