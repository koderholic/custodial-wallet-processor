package main

import (
	"wallet-adapter/app"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/middlewares"
	"wallet-adapter/utility"

	"log"
	"net/http"

	"github.com/gorilla/mux"
	validation "gopkg.in/go-playground/validator.v9"
)

func main() {
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

	app.RegisterRoutes(router, validator, config, logger, Database.DB)

	serviceAddress := ":" + config.AppPort

	middleware := middlewares.NewMiddleware(logger, router).
		LogAPIRequests().
		Build()

	logger.Info("Server started and listening on port %s", config.AppPort)
	log.Fatal(http.ListenAndServe(serviceAddress, middleware))
}
