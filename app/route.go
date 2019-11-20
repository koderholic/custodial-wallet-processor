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

		controller := controllers.NewController(app.Logger, app.Config, baseRepository)

		baseURL := "/api/v1"

		apiRouter := app.Router.PathPrefix(baseURL).Subrouter()
		app.Router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		// General Routes
		apiRouter.HandleFunc("/crypto/ping", controller.Ping).Methods(http.MethodGet)

	})

	app.Logger.Info("App routes registered successfully!")
}
