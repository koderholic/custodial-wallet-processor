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
	var userAsset dto.UserAsset
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		ReturnError(responseWriter, "GetAssetAddress", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR), controller.Logger)
		return
	}
	controller.Logger.Info("Incoming request details for GetAssetAddress : assetID : %+v", assetID)

	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: assetID}}, &userAsset); err != nil {
		ReturnError(responseWriter, "GetAssetAddress", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	if err := controller.Repository.GetByFieldName(&dto.UserAddress{AssetID: assetID}, &userAddress); err != nil {
		if err.Error() != utility.SQL_404 {
			ReturnError(responseWriter, "GetAssetAddress", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
			return
		}

		coinTypeToAddrMap := map[int64]string{}
		var address string

		// checks if an address has been created for one of it's user's assets with same coinType and use that instead
		var userAssets []dto.UserAsset
		if err := controller.Repository.GetAssetsByID(&dto.UserAsset{UserID: userAsset.UserID}, &userAssets); err != nil {
			ReturnError(responseWriter, "GetUserAssets", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err.(utility.AppError))), controller.Logger)
			return
		}

		for _, asset := range userAssets {
			assetAddress := dto.UserAddress{}
			if err := controller.Repository.GetByFieldName(&dto.UserAddress{AssetID: asset.ID}, &assetAddress); err != nil {
				continue
			}
			coinTypeToAddrMap[asset.CoinType] = assetAddress.Address
		}

		if coinTypeToAddrMap[userAsset.CoinType] != "" {
			address = coinTypeToAddrMap[userAsset.CoinType]
		} else {
			// Calls key-management service to create an address for the user asset
			address, err = services.GenerateAddress(controller.Cache, controller.Logger, controller.Config, userAsset.UserID, userAsset.AssetSymbol, &externalServiceErr)
			if err != nil || address == "" {
				if externalServiceErr.Code != "" {
					ReturnError(responseWriter, "GetAssetAddress", http.StatusInternalServerError, err, apiResponse.PlainError(utility.SVCS_KEYMGT_ERR, externalServiceErr.Message), controller.Logger)
					return
				}
				ReturnError(responseWriter, "GetAssetAddress", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("%s : %s", utility.SYSTEM_ERR, err.Error())), controller.Logger)
				return
			}
		}

		userAddress.AssetID = assetID
		userAddress.Address = address
		if createErr := controller.Repository.Create(&userAddress); createErr != nil {
			ReturnError(responseWriter, "GetAssetAddress", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
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
