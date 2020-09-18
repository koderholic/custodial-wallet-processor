package main

import (
	"fmt"
	"log"
	"time"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/tasks"
	"wallet-adapter/utility"
)

func main() {
	fmt.Println("Starting FloatManager")

	config := Config.Data{}
	config.Init("")
	if !config.EnableFloatManager {
		log.Println("Float manager is disabled... exiting")
		return
	}

	Database := &database.Database{
		Config: config,
	}
	Database.LoadDBInstance()
	defer Database.CloseDBInstance()

	purgeInterval := config.PurgeCacheInterval * time.Second
	cacheDuration := config.ExpireCacheDuration * time.Second
	//authCache, logger, config, baseRepository
	authCache := utility.InitializeCache(cacheDuration, purgeInterval)
	baseRepository := database.BaseRepository{Database: *Database}
	userAssetRepository := database.UserAssetRepository{BaseRepository: baseRepository}

	tasks.ManageFloat(authCache, config, baseRepository, userAssetRepository)
}
