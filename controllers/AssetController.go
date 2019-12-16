package controllers

import (
	"encoding/json"
	"net/http"
	"wallet-adapter/dto"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
)

// GetAsset ... This returns the crypto asset for the given id
func (controller BaseController) GetAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	responseData := dto.Asset{}
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	assetSymbol := routeParams["asset"]

	controller.Logger.Info("Incoming request details for GetAsset : %s", assetSymbol)

	if err := controller.Repository.GetByFieldName(dto.Asset{Symbol: assetSymbol}, &responseData); err != nil {

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
	json.NewEncoder(responseWriter).Encode(apiResponse.Successful("SUCCESS", utility.SUCCESS, responseData))

}

// FetchSupportedAssets ... This returns all supported crypto assets on the system
func (controller BaseController) FetchSupportedAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData []dto.Asset
	apiResponse := utility.NewResponse()

	if err := controller.Repository.FetchByFieldName(&dto.Asset{IsEnabled: true}, &responseData); err != nil {

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
	json.NewEncoder(responseWriter).Encode(apiResponse.Successful("SUCCESS", utility.SUCCESS, responseData))

}

// FetchAllAssets ... This fetch all crypto assets on the system
func (controller BaseController) FetchAllAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData []dto.Asset
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
	json.NewEncoder(responseWriter).Encode(apiResponse.Successful("SUCCESS", utility.SUCCESS, responseData))

}
