package controllers

import (
	"wallet-adapter/utility/variables"

	"encoding/json"
	"net/http"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/logger"
	Response "wallet-adapter/utility/response"

	"github.com/gorilla/mux"
)

// GetAllAssetAddresses ... Retrieves all addresses for the given asset, if non exist, it calls key-management to generate one
func (controller UserAddressController) GetAllAssetAddresses(responseWriter http.ResponseWriter, requestReader *http.Request) {
	var userAsset model.UserAsset
	var responseData dto.AllAssetAddresses
	apiResponse := Response.New()
	routeParams := mux.Vars(requestReader)
	assetID, err := utility.ToUUID(routeParams["assetId"])
	if err != nil {
		err := err.(appError.Err)
		ReturnError(responseWriter, "GetAllAssetAddresses", err, apiResponse.PlainError(err.ErrType, err.Error()))
		return
	}

	UserAddressService := services.NewUserAddressService(controller.Cache, controller.Config, controller.Repository)
	responseData, err := UserAddressService.GetV2AddressWithMemo(userAsset)

	responseData.DefaultAddressType = variables.DefaultAddressesTypes[userAsset.CoinType]
	logger.Info("Outgoing response to GetAllAssetAddresses request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(responseWriter).Encode(responseData)

}
