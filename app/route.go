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

func (app *App) RegisterRoutes() {

	once.Do(func() {
		db := database.Database{Logger: app.Logger, Config: app.Config, DB: app.DB}
		baseRepository := database.BaseRepository{Database: db}
		userAssetRepository := database.UserAssetRepository{BaseRepository: baseRepository}

		controller := controllers.NewController(app.Logger, app.Config, &baseRepository)
		userAssetController := controllers.NewUserAssetController(app.Logger, app.Config, &userAssetRepository)

		baseURL := "/api/v1"

		apiRouter := app.Router.PathPrefix(baseURL).Subrouter()
		app.Router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		// General Routes
		apiRouter.HandleFunc("/crypto/ping", controller.Ping).Methods(http.MethodGet)

		// Asset Routes
		apiRouter.HandleFunc("/crypto/add-asset", controller.AddSupportedAsset).Methods(http.MethodPost)
		apiRouter.HandleFunc("/crypto/update-asset/{assetId}", controller.UpdateAsset).Methods(http.MethodPut)
		apiRouter.HandleFunc("/crypto/assets", controller.FetchAssets).Methods(http.MethodGet)
		apiRouter.HandleFunc("/crypto/assets/{assetId}", controller.GetAsset).Methods(http.MethodGet)
		apiRouter.HandleFunc("/crypto/remove-asset/{assetId}", controller.RemoveAsset).Methods(http.MethodPost)

		// User Asset Routes
		apiRouter.HandleFunc("/crypto/users/{userId}/create-assets", userAssetController.CreateUserAssets).Methods(http.MethodPost)
		apiRouter.HandleFunc("/crypto/users/{userId}/assets", userAssetController.GetUserAssets).Methods(http.MethodGet)

	})

	app.Logger.Info("App routes registered successfully!")
}
