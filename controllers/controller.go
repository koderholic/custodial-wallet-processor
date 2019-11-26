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

//AssetController : Asset controller struct
type AssetController struct {
	Logger     *utility.Logger
	Config     config.Data
	Repository database.IAssetRepository
}

//UserAssetController : UserAsset controller struct
type UserAssetController struct {
	Logger     *utility.Logger
	Config     config.Data
	Repository database.IUserAssetRepository
}

// NewController ... Create a new base controller instance
func NewController(logger *utility.Logger, configData config.Data, repository database.IRepository) *Controller {
	controller := &Controller{}
	controller.Logger = logger
	controller.Config = configData
	controller.Repository = repository

	return controller
}

// NewAssetController ... Create a new asset controller instance
func NewAssetController(logger *utility.Logger, configData config.Data, repository database.IAssetRepository) *AssetController {
	controller := &AssetController{}
	controller.Logger = logger
	controller.Config = configData
	controller.Repository = repository

	return controller
}

// NewUserAssetController ... Create a new user asset controller instance
func NewUserAssetController(logger *utility.Logger, configData config.Data, repository database.IUserAssetRepository) *UserAssetController {
	controller := &UserAssetController{}
	controller.Logger = logger
	controller.Config = configData
	controller.Repository = repository

	return controller
}

//Ping : Ping function
func (c *Controller) Ping(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()

	c.Logger.Info("Ping request successful! Server is up and listening")

	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess("SUCCESS", "Ping request successful! Server is up and listening"))
}
