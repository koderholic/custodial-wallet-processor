package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

// CreateUserAssets ... Creates all supported crypto asset record on the given user account
func (controller UserAssetController) CreateUserAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := model.CreateUserAssetRequest{}
	responseData := model.CreateUserAssetResponse{}

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for CreateUserAssets : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		controller.Logger.Info("Outgoing response to CreateUserAssets request %+v", validationErr)
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.ValidateError("INPUT_ERR", utility.INPUT_ERR, validationErr))
		return
	}

	// Create user asset record for each given asset
	for i := 0; i < len(requestData.Assets); i++ {
		assetSymbol := requestData.Assets[i]
		asset := dto.Asset{}

		if err := controller.Repository.GetByFieldName(&dto.Asset{Symbol: assetSymbol, IsEnabled: true}, &asset); err != nil {
			controller.Logger.Info("Outgoing response to CreateUserAssets request %+v", err)
			responseWriter.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("Asset (%s) is currently not supported", assetSymbol)))
			return
		}

		userAsset := dto.UserBalance{AssetID: asset.ID, UserID: requestData.UserID}
		_ = controller.Repository.FindOrCreateUserAsset(userAsset, &userAsset)
		userAsset.Symbol = asset.Symbol
		responseData.Assets = append(responseData.Assets, userAsset)
	}

	controller.Logger.Info("Outgoing response to CreateUserAssets request %+v", responseData)
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.Success("SUCCESS", utility.SUCCESS, responseData))

}

// GetUserAssets ... Get all user asset balance
func (controller UserAssetController) GetUserAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData []dto.UserAssetBalance
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	userID, err := uuid.FromString(routeParams["userId"])
	if err != nil {
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR))
		return
	}

	if err := controller.Repository.GetAssetsByUserID(&dto.UserAssetBalance{UserID: userID}, &responseData); err != nil {
		if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
			return
		}

		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", err.(utility.AppError).Error()))
		return
	}
	controller.Logger.Info("Outgoing response to GetUserAssets request %+v", responseData)

	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.Success("SUCCESS", utility.SUCCESS, responseData))

}
