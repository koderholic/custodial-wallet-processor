package controllers

import (
	"encoding/json"
	"net/http"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
)

// GetAsset ...
func (controller BaseController) GetAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	responseData := model.Asset{}
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	assetSymbol := routeParams["asset"]

	controller.Logger.Info("Incoming request details for GetAsset : %s", assetSymbol)

	if err := controller.Repository.GetByFieldName(model.Asset{Symbol: assetSymbol}, &responseData); err != nil {

		if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
			return
		}

		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", err.(utility.AppError).Error()))
		return
	}

	controller.Logger.Info("Outgoing response to GetAsset request %+v", responseData)

	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.Success("SUCCESS", utility.SUCCESS, responseData))

}

// FetchSupportedAssets ...
func (controller BaseController) FetchSupportedAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData []model.Asset
	apiResponse := utility.NewResponse()

	if err := controller.Repository.FetchByFieldName(&model.Asset{IsEnabled: true}, &responseData); err != nil {

		if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
			return
		}

		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", err.(utility.AppError).Error()))
		return
	}

	controller.Logger.Info("Outgoing response to FetchSupportedAssets request %+v", responseData)

	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.Success("SUCCESS", utility.SUCCESS, responseData))

}

// FetchAllAssets ...
func (controller BaseController) FetchAllAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData []model.Asset
	apiResponse := utility.NewResponse()

	if err := controller.Repository.Fetch(&responseData); err != nil {

		if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
			return
		}

		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", err.(utility.AppError).Error()))
		return
	}

	controller.Logger.Info("Outgoing response to FetchAllAssets request %+v", responseData)

	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.Success("SUCCESS", utility.SUCCESS, responseData))

}
