package main

import (
	"fmt"
	"time"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/tasks"
	"wallet-adapter/utility"
)

func main() {
	fmt.Println("Starting Sweep Job")

	config := Config.Data{}
	config.Init("")

	logger := utility.NewLogger()

	Database := &database.Database{
		Logger: logger,
		Config: config,
	}
	Database.LoadDBInstance()
	defer Database.CloseDBInstance()

	purgeInterval := config.PurgeCacheInterval * time.Second
	cacheDuration := config.ExpireCacheDuration * time.Second
	//authCache, logger, config, baseRepository
	authCache := utility.InitializeCache(cacheDuration, purgeInterval)
	baseRepository := database.BaseRepository{Database: *Database}
	tasks.SweepTransactions(authCache, logger, config, baseRepository)

}
