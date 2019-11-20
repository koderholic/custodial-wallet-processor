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
		baseRepository := &database.BaseRepository{}
		baseRepository.Logger = app.Logger
		baseRepository.Config = app.Config
		baseRepository.DB = app.DB

		assetRepository := &database.AssetRepository{}
		assetRepository.Logger = app.Logger
		assetRepository.Config = app.Config
		assetRepository.DB = app.DB

		controller := controllers.NewController(app.Logger, app.Config, baseRepository)
		AssetController := controllers.NewController(app.Logger, app.Config, assetRepository)

		baseURL := "/api/v1"

		apiRouter := app.Router.PathPrefix(baseURL).Subrouter()
		app.Router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		// General Routes
		apiRouter.HandleFunc("/crypto/ping", controller.Ping).Methods(http.MethodGet)

		// Asset Routes
		apiRouter.HandleFunc("/crypto/add-asset", AssetController.AddSupportedAsset).Methods(http.MethodPost)
		apiRouter.HandleFunc("/crypto/update-asset/{assetId}", AssetController.UpdateAsset).Methods(http.MethodPut)
		apiRouter.HandleFunc("/crypto/assets", AssetController.FetchAssets).Methods(http.MethodGet)
		apiRouter.HandleFunc("/crypto/assets/{assetId}", AssetController.GetAsset).Methods(http.MethodGet)
		apiRouter.HandleFunc("/crypto/remove-asset/{assetId}", AssetController.RemoveAsset).Methods(http.MethodPost)

	})

	app.Logger.Info("App routes registered successfully!")
}
