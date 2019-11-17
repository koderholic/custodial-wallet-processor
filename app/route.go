package app

import (
	"net/http"
	"sync"
	"wallet-adapter/controllers"

	httpSwagger "github.com/swaggo/http-swagger"
)

var (
	once sync.Once
)

func (app *App) RegisterRoutes() {

	once.Do(func() {

		controller := controllers.Controller{
			Logger: app.Logger,
			Config: app.Config,
			DB:     app.DB,
		}

		baseURL := "/api/v1"

		apiRouter := app.Router.PathPrefix(baseURL).Subrouter()
		app.Router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

		apiRouter.HandleFunc("/crypto/ping", controller.Ping).Methods(http.MethodGet)

	})

	app.Logger.Info("App routes registered successfully!")
}
