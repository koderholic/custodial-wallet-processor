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
	var responseData dto.AssetAddress
	var userAsset model.UserAsset
	addressVersion := requestReader.URL.Query().Get("addressVersion")
	var address string
	var memo string
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		ReturnError(responseWriter, "GetAssetAddress", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR), controller.Logger)
		return
	}
	controller.Logger.Info("Incoming request details for GetAssetAddress : assetID : %+v", assetID)

	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetID}}, &userAsset); err != nil {
		ReturnError(responseWriter, "GetAssetAddress", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get userAsset with id = %s", utility.GetSQLErr(err), assetID)), controller.Logger)
		return
	}

	if addressVersion == utility.ADDRESS_VERSION_V2 && userAsset.RequiresMemo {
		v2Address, err := services.GetV2AddressWithMemo(controller.Repository, controller.Logger, controller.Cache, controller.Config, userAsset)
		if err != nil {
			controller.Logger.Info("Error from GetV2AddressWithMemo service : %s", err)
			ReturnError(responseWriter, "GetAssetAddress", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERROR", utility.SYSTEM_ERR), controller.Logger)
			return
		}
		address = v2Address.Address
		memo = v2Address.Memo
	} else {
		address, err = services.GetV1Address(controller.Repository, controller.Logger, controller.Cache, controller.Config, userAsset)
		if err != nil {
			controller.Logger.Info("Error from GetV1Address service : %s", err)
			ReturnError(responseWriter, "GetAssetAddress", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERROR", utility.SYSTEM_ERR), controller.Logger)
			return
		}
	}

	responseData = dto.AssetAddress{
		Address: address,
		Memo:    memo,
	}

	controller.Logger.Info("Outgoing response to GetAssetAddress request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(responseWriter).Encode(responseData)

}

// GetAllAssetAddresses ... Retrieves all addresses for the given asset, if non exist, it calls key-management to generate one
func (controller UserAssetController) GetAllAssetAddresses(responseWriter http.ResponseWriter, requestReader *http.Request) {
	var assetAddresses []dto.AssetAddress
	var userAsset model.UserAsset
	var responseData dto.AllAssetAddresses
	apiResponse := utility.NewResponse()
	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		ReturnError(responseWriter, "GetAllAssetAddresses", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR), controller.Logger)
		return
	}
	controller.Logger.Info("Incoming request details for GetAllAssetAddresses : assetID : %+v", assetID)

	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetID}}, &userAsset); err != nil {
		ReturnError(responseWriter, "GetAllAssetAddresses", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get userAsset with id = %s", utility.GetSQLErr(err), assetID)), controller.Logger)
		return
	}

	if userAsset.RequiresMemo {
		v2Address, err := services.GetV2AddressWithMemo(controller.Repository, controller.Logger, controller.Cache, controller.Config, userAsset)
		if err != nil {
			controller.Logger.Info("Error from GetV2AddressWithMemo service : %s", err)
			ReturnError(responseWriter, "GetAllAssetAddresses", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERROR", utility.SYSTEM_ERR), controller.Logger)
			return
		}
		assetAddresses = append(assetAddresses, dto.AssetAddress{
			Address: v2Address.Address,
			Memo:    v2Address.Memo,
		})
		responseData = dto.AllAssetAddresses{
			Addresses: assetAddresses,
		}
	} else {
		var err error
		var address string
		AddressService := services.BaseService{Config: controller.Config, Cache: controller.Cache, Logger: controller.Logger}

		if userAsset.AssetSymbol == utility.BTC {
			responseData.Addresses, err = AddressService.GetBTCAddresses(controller.Repository, userAsset)
			responseData.DefaultAddressType = utility.DEFAULT_BTC_ADDRESS_TYPE
			// responseData = dto.AllAssetAddresses{
			// 	Addresses:          assetAddresses,
			// 	DefaultAddressType: utility.DEFAULT_BTC_ADDRESS_TYPE,
			// }
		} else {
			address, err = services.GetV1Address(controller.Repository, controller.Logger, controller.Cache, controller.Config, userAsset)
			responseData.Addresses = append(assetAddresses, dto.AssetAddress{
				Address: address,
			})
			// responseData = dto.AllAssetAddresses{
			// 	Addresses: assetAddresses,
			// }
		}

		if err != nil {
			ReturnError(responseWriter, "GetAllAssetAddresses", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERROR", utility.SYSTEM_ERR), controller.Logger)
			return
		}
	}

	controller.Logger.Info("Outgoing response to GetAllAssetAddresses request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(responseWriter).Encode(responseData)

}
