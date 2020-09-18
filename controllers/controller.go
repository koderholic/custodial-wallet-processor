package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/errorcode"
	"wallet-adapter/utility"
	"wallet-adapter/utility/logger"

	validation "gopkg.in/go-playground/validator.v9"
)

//Controller : Controller struct
type Controller struct {
	Cache     *utility.MemoryCache
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

//BatchController : Batch controller struct
type BatchController struct {
	Controller
	Repository database.IBatchRepository
}

// NewController ... Create a new base controller instance
func NewController(cache *utility.MemoryCache, configData config.Data, validator *validation.Validate, repository database.IRepository) *BaseController {
	controller := &BaseController{}
	controller.Cache = cache
	controller.Config = configData
	controller.Validator = validator
	controller.Repository = repository

	return controller
}

// NewUserAssetController ... Create a new user asset controller instance
func NewUserAssetController(cache *utility.MemoryCache, configData config.Data, validator *validation.Validate, repository database.IUserAssetRepository) *UserAssetController {
	controller := &UserAssetController{}
	controller.Cache = cache
	controller.Config = configData
	controller.Validator = validator
	controller.Repository = repository

	return controller
}

// NewBatchController ... Create a new batch controller instance
func NewBatchController(cache *utility.MemoryCache, configData config.Data, validator *validation.Validate, repository database.IBatchRepository) *BatchController {
	controller := &BatchController{}
	controller.Cache = cache
	controller.Config = configData
	controller.Validator = validator
	controller.Repository = repository

	return controller
}

//Ping : Ping function
func (controller *Controller) Ping(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()

	logger.Info("Ping request successful! Server is up and listening")

	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess(utility.SUCCESSFUL, "Ping request successful! Server is up and listening"))
}

func ValidateRequest(validator *validation.Validate, requestData interface{}) error {
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
	return utility.AppError{
		ErrCode: http.StatusBadRequest,
		ErrType: errorcode.VALIDATION_ERR_CODE,
		Err:     errors.New(errorcode.VALIDATION_ERR),
		ErrData: validationErr,
	}
}
func ReturnError(responseWriter http.ResponseWriter, executingMethod string, err interface{}, response interface{}) {
	logger.Error("Outgoing response to %s : %+v. Additional context : %s", executingMethod, response, err)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(err.(utility.AppError).ErrCode)
	json.NewEncoder(responseWriter).Encode(response)
}
