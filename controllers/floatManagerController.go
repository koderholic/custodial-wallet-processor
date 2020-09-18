package controllers

import (
	"encoding/json"
	"net/http"
	"wallet-adapter/database"
	"wallet-adapter/tasks"
	"wallet-adapter/utility"
	"wallet-adapter/utility/logger"
)

func (controller UserAssetController) TriggerFloat(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()

	// Endpoint spins up a go-routine to process queued transactions and sends back an acknowledgement to the scheduler
	done := make(chan bool)

	go func() {

		Database := &database.Database{
			Config: controller.Config,
			DB:     controller.Repository.Db(),
		}

		db := *Database
		baseRepository := database.BaseRepository{Database: db}
		userAssetRepository := database.UserAssetRepository{BaseRepository: baseRepository}
		tasks.ManageFloat(controller.Cache, controller.Config, baseRepository, userAssetRepository)

		done <- true
	}()

	logger.Info("Outgoing response to TriggerFloat request %+v", utility.SUCCESS)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess(utility.SUCCESSFUL, utility.SUCCESS))

	<-done
}
