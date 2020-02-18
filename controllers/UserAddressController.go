package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

// GetAssetAddress ... Retrieves the blockchain address of an address, if non exist, it calls key-management to generate one
func (controller UserAssetController) GetAssetAddress(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var externalServiceErr model.ServicesRequestErr
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
		_ = json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR))
		return
	}
	controller.Logger.Info("Incoming request details for GetAssetAddress : assetID : %+v", assetID)

	if err := controller.Repository.GetAssetsByID(&dto.UserAssetBalance{BaseDTO: dto.BaseDTO{ID: assetID}}, &userAsset); err != nil {
		controller.Logger.Error("Outgoing response to GetAssetAddress request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		_ = json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	if err := controller.Repository.GetByFieldName(&dto.UserAddress{AssetID: assetID}, &userAddress); err != nil {
		if err.Error() != utility.SQL_404 {
			controller.Logger.Error("Outgoing response to GetAssetAddress request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
			return
		}

		// Calls key-management service to create an address for the user asset
		address, errGenerateAddress := services.GenerateAddress(controller.Cache, controller.Logger, controller.Config, userAsset.UserID, userAsset.Symbol, &externalServiceErr)
		if errGenerateAddress != nil || address == "" {
			controller.Logger.Error("Outgoing response to GetAssetAddress request %+v", errGenerateAddress)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusInternalServerError)
			if externalServiceErr.Code != "" {
				_ = json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SVCS_KEYMGT_ERR", externalServiceErr.Message))
				return
			}
			_ = json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("%s : %s", utility.SYSTEM_ERR, errGenerateAddress.Error())))
			return
		}
		userAddress.AssetID = assetID
		userAddress.Address = address
		if createErr := controller.Repository.Create(&userAddress); createErr != nil {
			controller.Logger.Error("Outgoing response to GetAssetAddress request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(createErr)))
			return
		}
	}
	responseData = map[string]string{
		"address": userAddress.Address,
	}

	controller.Logger.Info("Outgoing response to GetAssetAddress request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(responseWriter).Encode(responseData)

}
