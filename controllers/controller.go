package controllers

import (
	"encoding/json"
	"net/http"
	"wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/utility"
)

//Controller : Controller struct
type Controller struct {
	Logger     *utility.Logger
	Config     config.Data
	Repository database.IRepository
}

func NewController(logger *utility.Logger, configData config.Data, repository database.IRepository) *Controller {
	controller := &Controller{}
	controller.Logger = logger
	controller.Config = configData
	controller.Repository = repository

	return controller
}

//Ping : Ping function
func (c *Controller) Ping(w http.ResponseWriter, r *http.Request) {

	apiResponse := utility.NewResponse()

	c.Logger.Info("Ping request successful! Server is up and listening")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse.PlainSuccess(utility.SUCCESS, "Ping request successful! Server is up and listening"))
}
