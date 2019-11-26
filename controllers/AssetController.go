package controllers

import (
	"encoding/json"
	"net/http"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

func (c AssetController) GetAsset(responseWriter http.ResponseWriter, requestReader *http.Request) {

	responseData := model.Asset{}
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("UUID_CAST_ERR", utility.UUID_CAST_ERR))
		return
	}

	c.Logger.Info("Incoming request details: %+v", assetID)

	if err := c.Repository.Get(assetID, &responseData); err != nil {

		if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
			return
		}

		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", err.(utility.AppError).Error()))
		return
	}

	c.Logger.Info("Outgoing response to successful request %+v", responseData)

	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.Success("SUCCESS", utility.SUCCESS, responseData))

}

// FetchSupportedAssets ...
func (c AssetController) FetchSupportedAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData []model.Asset
	apiResponse := utility.NewResponse()

	if err := c.Repository.GetSupportedCrypto(&responseData); err != nil {

		if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
			return
		}

		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", err.(utility.AppError).Error()))
		return
	}

	c.Logger.Info("Outgoing response to successful request %+v", responseData)

	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.Success("SUCCESS", utility.SUCCESS, responseData))

}

// FetchAllAssets ...
func (c AssetController) FetchAllAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData []model.Asset
	apiResponse := utility.NewResponse()

	if err := c.Repository.Fetch(&responseData); err != nil {

		if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
			return
		}

		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", err.(utility.AppError).Error()))
		return
	}

	c.Logger.Info("Outgoing response to successful request %+v", responseData)

	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.Success("SUCCESS", utility.SUCCESS, responseData))

}
