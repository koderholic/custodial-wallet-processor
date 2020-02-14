package main

import (
	"wallet-adapter/app"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/services"
	"wallet-adapter/tasks"
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
	Database.DBSeeder()

	if err := services.InitHotWallet(Database.DB, logger, config); err != nil {
		logger.Error("Server started and listening on port %s", config.AppPort)
	}

	app.RegisterRoutes(router, validator, config, logger, Database.DB)

	serviceAddress := ":" + config.AppPort

	// middleware := middlewares.NewMiddleware(logger, config, router).ValidateAuthToken().LogAPIRequests().Build()
	db := *Database
	baseRepository := database.BaseRepository{Database: db}
	tasks.ExecuteCronJob(logger, config, baseRepository)

	logger.Info("Server started and listening on port %s", config.AppPort)
	log.Fatal(http.ListenAndServe(serviceAddress, router))
}
