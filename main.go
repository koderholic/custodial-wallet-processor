package main

import (
	"wallet-adapter/app"
	"wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/utility"

	"github.com/gorilla/handlers"

	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	APP := &app.App{}

	Config := config.Data{}
	Config.Init("")
	APP.Config = Config

	APP.Logger = utility.NewLogger(Config.LogFile, Config.AppName, Config.LogFolder)
	APP.Router = mux.NewRouter()

	Database := &database.Database{
		Logger: APP.Logger,
		Config: APP.Config,
	}
	Database.LoadDBInstance()

	APP.DB = Database.DB
	APP.RegisterRoutes()

	serviceAddress := ":" + Config.AppPort

	APP.Logger.Info("Server started and listening on port %s", Config.AppPort)
	log.Fatal(http.ListenAndServe(serviceAddress, handlers.CompressHandler(APP.Router)))
}
