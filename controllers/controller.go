package controllers

import (
	"encoding/json"
	"net/http"
	"wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/utility"

	validation "gopkg.in/go-playground/validator.v9"
)

//Controller : Controller struct
type Controller struct {
	Logger    *utility.Logger
	Config    config.Data
	Validator *validation.Validate
}

//BaseController : Base controller struct
type BaseController struct {
	Controller
	Repository database.IRepository
}

//UserAssetController : UserAsset controller struct
type UserAssetController struct {
	Controller
	Repository database.IUserAssetRepository
}

// NewController ... Create a new base controller instance
func NewController(logger *utility.Logger, configData config.Data, validator *validation.Validate, repository database.IRepository) *BaseController {
	controller := &BaseController{}
	controller.Logger = logger
	controller.Config = configData
	controller.Validator = validator
	controller.Repository = repository

	return controller
}

// NewUserAssetController ... Create a new user asset controller instance
func NewUserAssetController(logger *utility.Logger, configData config.Data, validator *validation.Validate, repository database.IUserAssetRepository) *UserAssetController {
	controller := &UserAssetController{}
	controller.Logger = logger
	controller.Config = configData
	controller.Validator = validator
	controller.Repository = repository

	return controller
}

//Ping : Ping function
func (controller *Controller) Ping(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()

	controller.Logger.Info("Ping request successful! Server is up and listening")

	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess("SUCCESS", "Ping request successful! Server is up and listening"))
}

func ValidateRequest(validator *validation.Validate, requestData interface{}, logger *utility.Logger) []map[string]string {
	validationErr := []map[string]string{}
	translation, err := utility.CustomizeValidationMessages(validator)
	if err != nil {
		logger.Error("Failed to set custom validation error messages : %s", err)
	}
	if err := validator.Struct(requestData); err != nil {
		for _, err := range err.(validation.ValidationErrors) {

			validationErr = append(validationErr, map[string]string{
				"field":   err.Field(),
				"message": err.Translate(translation),
			})
		}
	}
	return validationErr
}
