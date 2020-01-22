package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"wallet-adapter/dto"
	"wallet-adapter/services"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

// GetAssetAddress ... Retrieves the blockchain address of an address, if non exist, it calls key-management to generate one
func (controller BaseController) GetAssetAddress(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData map[string]string
	var userAddress dto.UserAddress
	var userAsset dto.UserAssetBalance
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		controller.Logger.Error("Outgoing response to GetAssetAddress request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR))
		return
	}
	controller.Logger.Info("Incoming request details for GetAssetAddress : assetID : %+v", assetID)

	if err := controller.Repository.Get(assetID, &userAsset); err != nil {
		controller.Logger.Error("Outgoing response to GetAssetAddress request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	if err := controller.Repository.GetByFieldName(&dto.UserAddress{AssetID: assetID}, &userAddress); err != nil {
		if err.Error() == "record not found" {
			// Calls key-management service to create an address for the user asset
			address, errGenerateAddress := services.GenerateAddress(controller.Logger, controller.Config, userAsset.UserID, userAsset.Symbol)
			if errGenerateAddress != nil {
				controller.Logger.Error("Outgoing response to GetAssetAddress request %+v", errGenerateAddress)
				responseWriter.Header().Set("Content-Type", "application/json")
				responseWriter.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("%s : %s", utility.SYSTEM_ERR, errGenerateAddress.Error())))
				return
			}
			userAddress.AssetID = userAsset.ID
			userAddress.Address = address
			if createErr := controller.Repository.Create(&userAddress); createErr != nil {
				controller.Logger.Error("Outgoing response to GetAssetAddress request %+v", err)
				responseWriter.Header().Set("Content-Type", "application/json")
				responseWriter.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(createErr)))
				return
			}
		}
		controller.Logger.Error("Outgoing response to GetAssetAddress request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}
	controller.Logger.Info("Outgoing response to GetUserAssets request %+v", responseData)

	responseData = map[string]string{
		"address": userAddress.Address,
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(apiResponse.Successful("SUCCESS", utility.SUCCESS, responseData))

}
