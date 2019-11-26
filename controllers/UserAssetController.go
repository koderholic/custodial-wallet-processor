package controllers

import (
	"encoding/json"
	"net/http"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

// CreateUserAssets ... Creates all supported crypto asset record on the given user account
func (c UserAssetController) CreateUserAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {
	apiResponse := utility.NewResponse()
	var supportedAssets []model.Asset

	routeParams := mux.Vars(requestReader)
	userID, err := uuid.FromString(routeParams["userId"])
	if err != nil {
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("UUID_CAST_ERR", utility.UUID_CAST_ERR))
		return
	}

	if err := c.Repository.Fetch(&supportedAssets); err != nil {
		if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
			return
		}
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", err.(utility.AppError).Error()))
		return
	}
	for i := 0; i < len(supportedAssets); i++ {
		assetID := supportedAssets[i].ID
		userAsset := model.UserBalance{AssetID: assetID, UserID: userID}
		if err := c.Repository.FindOrCreateUserAsset(userAsset, &userAsset); err != nil {
			if err.(utility.AppError).Type() == utility.SYSTEM_ERR {
				responseWriter.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
				return
			}
			responseWriter.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", err.(utility.AppError).Error()))
			return
		}
	}

	responseData := []model.UserAssetBalance{}
	c.Repository.GetAssetsByUserID(&model.UserAssetBalance{UserID: userID}, &responseData)

	c.Logger.Info("Outgoing response to successful request %+v", responseData)

	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.Success("SUCCESS", utility.SUCCESS, responseData))

}

// GetUserAssets ... Get all user asset balance
func (c UserAssetController) GetUserAssets(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData []model.UserAssetBalance
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	userID, err := uuid.FromString(routeParams["userId"])
	if err != nil {
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR))
		return
	}

	if err := c.Repository.GetAssetsByUserID(&model.UserAssetBalance{UserID: userID}, &responseData); err != nil {
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
