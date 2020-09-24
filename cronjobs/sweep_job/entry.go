package main

import (
	"fmt"
	"time"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/tasks/sweep"
	"wallet-adapter/utility/cache"
)

func main() {
	fmt.Println("Starting Sweep Job")

	config := Config.Data{}
	config.Init("")

	Database := &database.Database{
		Config: config,
	}
	Database.LoadDBInstance()
	defer Database.CloseDBInstance()

	purgeInterval := config.PurgeCacheInterval * time.Second
	cacheDuration := config.ExpireCacheDuration * time.Second
	//authCache, logger, config, baseRepository
	authCache := cache.Initialize(cacheDuration, purgeInterval)
	baseRepository := database.BaseRepository{Database: *Database}
	sweep.SweepTransactions(authCache, config, &baseRepository)

}
