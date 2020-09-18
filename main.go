package main

import (
	"fmt"
	"wallet-adapter/app"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/migration"
	"wallet-adapter/services"
	"wallet-adapter/utility"
	"wallet-adapter/utility/logger"

	"github.com/getsentry/sentry-go"

	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	validation "gopkg.in/go-playground/validator.v9"
)

func main() {
	fmt.Println("Starting application: Wallet Adapter")
	config := Config.Data{}
	config.Init("")

	router := mux.NewRouter()
	validator := validation.New()

	Database := &database.Database{
		Config: config,
	}
	Database.LoadDBInstance()
	defer Database.CloseDBInstance()
	if err := migration.RunDbMigrations(config); err != nil {
		logger.Fatal("Error running migration: ", err)
	}

	purgeInterval := config.PurgeCacheInterval * time.Second
	cacheDuration := config.ExpireCacheDuration * time.Second
	authCache := utility.InitializeCache(cacheDuration, purgeInterval)

	DenominationServices := services.NewDenominationServices(authCache, config)
	DenominationServices.SeedSupportedAssets(Database.DB)

	HotWalletService := services.NewHotWalletService(authCache, config)
	if err := HotWalletService.InitHotWallet(authCache, Database.DB, config); err != nil {
		logger.Error("Error with InitHotWallet %s", err)
	}
	SharedAddressService := services.NewSharedAddressService(authCache, config)
	if err := SharedAddressService.InitSharedAddress(authCache, Database.DB, config); err != nil {
		logger.Error("Error with InitSharedAddress %s", err)
	}

	app.RegisterRoutes(router, validator, config, Database.DB, authCache)

	serviceAddress := ":" + config.AppPort

	// middleware := middlewares.NewMiddleware(logger, config, router).ValidateAuthToken().LogAPIRequests().Build()

	err := sentry.Init(sentry.ClientOptions{
		// Either set your DSN here or set the SENTRY_DSN environment variable.
		Dsn: config.SentryDsn,
		// Enable printing of SDK debug messages.
		// Useful when getting started or trying to figure something out.
		Debug: false,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
	// Flush buffered events before the program terminates.
	// Set the timeout to the maximum duration the program can afford to wait.
	defer sentry.Flush(2 * time.Second)
	logger.Info("Server started and listening on port %s", config.AppPort)
	log.Fatal(http.ListenAndServe(serviceAddress, router))
}
