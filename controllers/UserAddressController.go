package controllers

import (
	"wallet-adapter/utility/variables"

	"encoding/json"
	"fmt"
	"net/http"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/constants"
	"wallet-adapter/utility/errorcode"
	"wallet-adapter/utility/logger"
	Response "wallet-adapter/utility/response"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

// GetAssetAddress ... Retrieves the blockchain address of an address, if non exist, it calls key-management to generate one
func (controller UserAddressController) GetAssetAddress(responseWriter http.ResponseWriter, requestReader *http.Request) {
	var responseData dto.AssetAddress
	var userAsset model.UserAsset
	addressVersion := requestReader.URL.Query().Get("addressVersion")
	var address string
	var memo string
	apiResponse := Response.New()

	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		ReturnError(responseWriter, "GetAssetAddress", err, apiResponse.PlainError("INPUT_ERR", errorcode.UUID_CAST_ERR))
		return
	}
	logger.Info("Incoming request details for GetAssetAddress : assetID : %+v", assetID)

	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetID}}, &userAsset); err != nil {
		ReturnError(responseWriter, "GetAssetAddress", err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get asset address with id = %s", appError.GetSQLErr(err), assetID)))
		return
	}

	// Check if deposit is ACTIVE on this asset
	DenominationServices := services.NewDenominationServices(controller.Cache, controller.Config, controller.Repository, nil)
	isActive, err := DenominationServices.IsDepositActive(userAsset.AssetSymbol)
	if err != nil {
		ReturnError(responseWriter, "GetAssetAddress", err, apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("%s, for get asset address with id = %s", appError.GetSQLErr(err), assetID)))
		return
	}
	if !isActive {
		ReturnError(responseWriter, "GetAssetAddress", err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get asset address with id = %s", errorcode.DEPOSIT_NOT_ACTIVE, assetID)))
		return
	}

	UserAddressService := services.NewUserAddressService(controller.Cache, controller.Config, controller.Repository, nil)
	if addressVersion == constants.ADDRESS_VERSION_V2 && userAsset.RequiresMemo {
		v2Address, err := UserAddressService.GetV2AddressWithMemo(userAsset)
		if err != nil {
			logger.Info("Error from GetV2AddressWithMemo service : %s", err)
			ReturnError(responseWriter, "GetAssetAddress", err, apiResponse.PlainError("SERVER_ERROR", errorcode.SERVER_ERR))
			return
		}
		address = v2Address.Address
		memo = v2Address.Memo
	} else {
		address, err = UserAddressService.GetV1Address(userAsset)
		if err != nil {
			logger.Info("Error from GetV1Address service : %s", err)
			ReturnError(responseWriter, "GetAssetAddress", err, apiResponse.PlainError("SERVER_ERROR", errorcode.SERVER_ERR))
			return
		}
	}

	responseData = dto.AssetAddress{
		Address: address,
		Memo:    memo,
	}

	logger.Info("Outgoing response to GetAssetAddress request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(responseWriter).Encode(responseData)

}

// GetAllAssetAddresses ... Retrieves all addresses for the given asset, if non exist, it calls key-management to generate one
func (controller UserAddressController) GetAllAssetAddresses(responseWriter http.ResponseWriter, requestReader *http.Request) {
	var userAsset model.UserAsset
	var responseData dto.AllAssetAddresses
	apiResponse := Response.New()
	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		ReturnError(responseWriter, "GetAllAssetAddresses", err, apiResponse.PlainError("INPUT_ERR", errorcode.UUID_CAST_ERR))
		return
	}
	logger.Info("Incoming request details for GetAllAssetAddresses : assetID : %+v", assetID)

	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetID}}, &userAsset); err != nil {
		ReturnError(responseWriter, "GetAllAssetAddresses", err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get userAsset with id = %s", appError.GetSQLErr(err), assetID)))
		return
	}

	// Check if deposit is ACTIVE on this asset
	DenominationServices := services.NewDenominationServices(controller.Cache, controller.Config, controller.Repository, nil)
	isActive, err := DenominationServices.IsDepositActive(userAsset.AssetSymbol)
	if err != nil {
		ReturnError(responseWriter, "GetAllAssetAddresses", err, apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("%s, for get asset address with id = %s", appError.GetSQLErr(err), assetID)))
		return
	}
	if !isActive {
		ReturnError(responseWriter, "GetAllAssetAddresses", err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get asset address with id = %s", errorcode.DEPOSIT_NOT_ACTIVE, assetID)))
		return
	}

	UserAddressService := services.NewUserAddressService(controller.Cache, controller.Config, controller.Repository, nil)
	if userAsset.RequiresMemo {
		v2Address, err := UserAddressService.GetV2AddressWithMemo(userAsset)
		if err != nil {
			logger.Info("Error from GetV2AddressWithMemo service : %s", err)
			ReturnError(responseWriter, "GetAllAssetAddresses", err, apiResponse.PlainError("SERVER_ERROR", errorcode.SERVER_ERR))
			return
		}
		responseData.Addresses = append(responseData.Addresses, dto.AssetAddress{
			Address: v2Address.Address,
			Memo:    v2Address.Memo,
		})
	} else {
		var err error
		var address string

		if userAsset.AssetSymbol == constants.COIN_BTC {
			responseData.Addresses, err = UserAddressService.GetBTCAddresses(userAsset)
		} else {
			address, err = UserAddressService.GetV1Address(userAsset)
			responseData.Addresses = append(responseData.Addresses, dto.AssetAddress{
				Address: address,
			})
		}

		if err != nil {
			ReturnError(responseWriter, "GetAllAssetAddresses", err, apiResponse.PlainError("SERVER_ERROR", errorcode.SERVER_ERR))
			return
		}
	}

	responseData.DefaultAddressType = variables.DefaultAddressesTypes[userAsset.CoinType]
	logger.Info("Outgoing response to GetAllAssetAddresses request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(responseWriter).Encode(responseData)

}