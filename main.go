package main

import (
	"fmt"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/migration"
	"wallet-adapter/routes"
	"wallet-adapter/services"
	"wallet-adapter/utility/cache"
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
	authCache := cache.Initialize(cacheDuration, purgeInterval)

	DenominationServices := services.NewDenominationServices(authCache, config, nil, nil)
	DenominationServices.SeedSupportedAssets(Database.DB)

	HotWalletService := services.NewHotWalletService(authCache, config, nil, nil)
	if err := HotWalletService.InitHotWallet(Database.DB); err != nil {
		logger.Error("Error with InitHotWallet %s", err)
	}
	SharedAddressService := services.NewSharedAddressService(authCache, config, nil, nil)
	if err := SharedAddressService.InitSharedAddress(Database.DB); err != nil {
		logger.Error("Error with InitSharedAddress %s", err)
	}

	routes.Register(router, validator, config, Database.DB, authCache)

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
