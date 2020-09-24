package controllers

import (
	"encoding/json"
	"net/http"
	"wallet-adapter/database"
	"wallet-adapter/tasks/float"
	"wallet-adapter/utility/constants"
	"wallet-adapter/utility/logger"
	Response "wallet-adapter/utility/response"
)

func (controller UserAssetController) TriggerFloat(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := Response.New()

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
		float.ManageFloat(controller.Cache, controller.Config, &baseRepository, &userAssetRepository)

		done <- true
	}()

	logger.Info("Outgoing response to TriggerFloat request %+v", constants.SUCCESS)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess(constants.SUCCESSFUL, constants.SUCCESS))

	<-done
}
