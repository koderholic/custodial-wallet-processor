package main

import (
	"wallet-adapter/app"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/services"
	"wallet-adapter/utility"

	"log"
	"net/http"
	"time"

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

	// done := make(chan bool, 1)
	// go migration(done)

	purgeInterval := config.PurgeCacheInterval * time.Second
	cacheDuration := config.ExpireCacheDuration * time.Second
	authCache := utility.InitializeCache(cacheDuration, purgeInterval)

	if err := services.InitHotWallet(authCache, Database.DB, logger, config); err != nil {
		logger.Error("Server started and listening on port %s", config.AppPort)
	}

	app.RegisterRoutes(router, validator, config, logger, Database.DB, authCache)

	serviceAddress := ":" + config.AppPort

	// middleware := middlewares.NewMiddleware(logger, config, router).ValidateAuthToken().LogAPIRequests().Build()

	logger.Info("Server started and listening on port %s", config.AppPort)
	log.Fatal(http.ListenAndServe(serviceAddress, router))

	// <-done
}

// Handles database migrations
// func migration(done chan bool) {
// 	var keyword, action, location string
// 	go func() {
// 		for {
// 			if fmt.Scan(&keyword, &action, &location); keyword == "migration:" {
// 				if action == "CREATE" || action == "DROP" || action == "ALTER" || action == "ALTERKEY" || action == "ALTERREF" || action == "TRUNCATE" {
// 					if location != "" {
// 						database.Migration.Action = action
// 						database.Migration.Location = location
// 						database.Migrate()

// 					}
// 				}
// 			}
// 		}
// 		done <- true

// 	}()

// }