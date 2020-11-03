package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"wallet-adapter/dto"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

// GetAllAssetAddresses ... Retrieves all addresses for the given asset, if non exist, it calls key-management to generate one
func (controller UserAssetController) GetAllAssetAddresses(responseWriter http.ResponseWriter, requestReader *http.Request) {
	var userAsset model.UserAsset
	var responseData dto.AllAssetAddresses
	apiResponse := utility.NewResponse()
	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		ReturnError(responseWriter, "GetAllAssetAddresses", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR", errorcode.UUID_CAST_ERR), controller.Logger)
		return
	}
	controller.Logger.Info("Incoming request details for GetAllAssetAddresses : assetID : %+v", assetID)

	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetID}}, &userAsset); err != nil {
		ReturnError(responseWriter, "GetAllAssetAddresses", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get userAsset with id = %s", utility.GetSQLErr(err), assetID)), controller.Logger)
		return
	}

	// Check if deposit is ACTIVE on this asset
	userAssetService := services.NewService(controller.Cache, controller.Logger, controller.Config)
	isActive, err := userAssetService.IsDepositActive(userAsset.AssetSymbol, controller.Repository)
	if err != nil {
		ReturnError(responseWriter, "GetAllAssetAddresses", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("%s, for get asset address with id = %s", utility.GetSQLErr(err), assetID)), controller.Logger)
		return
	}
	if !isActive {
		ReturnError(responseWriter, "GetAllAssetAddresses", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get asset address with id = %s", errorcode.DEPOSIT_NOT_ACTIVE, assetID)), controller.Logger)
		return
	}

	if userAsset.RequiresMemo {
		v2Address, err := services.GetV2AddressWithMemo(controller.Repository, controller.Logger, controller.Cache, controller.Config, userAsset)
		if err != nil {
			controller.Logger.Info("Error from GetV2AddressWithMemo service : %s", err)
			ReturnError(responseWriter, "GetAllAssetAddresses", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERROR", errorcode.SYSTEM_ERR), controller.Logger)
			return
		}
		responseData.Addresses = append(responseData.Addresses, dto.AssetAddress{
			Address: v2Address.Address,
			Memo:    v2Address.Memo,
		})
	} else {
		var err error
		var address string
		AddressService := services.BaseService{Config: controller.Config, Cache: controller.Cache, Logger: controller.Logger}

		if userAsset.AssetSymbol == utility.COIN_BTC {
			responseData.Addresses, err = AddressService.GetBTCAddresses(controller.Repository, userAsset)
		} else {
			address, err = services.GetV1Address(controller.Repository, controller.Logger, controller.Cache, controller.Config, userAsset)
			responseData.Addresses = append(responseData.Addresses, dto.AssetAddress{
				Address: address,
			})
		}

		if err != nil {
			ReturnError(responseWriter, "GetAllAssetAddresses", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERROR", errorcode.SYSTEM_ERR), controller.Logger)
			return
		}
	}

	responseData.DefaultAddressType = utility.DefaultAddressesTypes[userAsset.CoinType]
	controller.Logger.Info("Outgoing response to GetAllAssetAddresses request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(responseWriter).Encode(responseData)

}