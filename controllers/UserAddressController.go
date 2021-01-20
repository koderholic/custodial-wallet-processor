package controllers

import (
	"encoding/json"
	"errors"
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

	if err := controller.VerifyDepositIsSupportedAndPopulateAsset(assetID, &userAsset); err != nil {
		ReturnError(responseWriter, "GetAllAssetAddresses", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR",
			fmt.Sprintf("%s, for get asset address with id = %s", errorcode.DEPOSIT_NOT_ACTIVE, assetID)), controller.Logger)
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
	controller.Logger.Info("Outgoing response to GetAllAssetAddresses request %+v", http.StatusOK)
	responseWriter.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(responseWriter).Encode(responseData)

}

func (controller UserAssetController) CreateAuxiliaryAddress(responseWriter http.ResponseWriter, requestReader *http.Request)  {

	var userAsset model.UserAsset
	var responseData dto.AssetAddress
	apiResponse := utility.NewResponse()
	routeParams := mux.Vars(requestReader)
	addressType := requestReader.URL.Query().Get("addressType")

	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		ReturnError(responseWriter, "CreateAuxiliaryAddress", http.StatusBadRequest, err, apiResponse.
			PlainError("INPUT_ERR", errorcode.UUID_CAST_ERR), controller.Logger)
		return
	}

	if err := controller.VerifyDepositIsSupportedAndPopulateAsset(assetID, &userAsset); err != nil {
		ReturnError(responseWriter, "CreateAuxiliaryAddress", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR",
			fmt.Sprintf("%s, for get asset address with id = %s", errorcode.DEPOSIT_NOT_ACTIVE, assetID)), controller.Logger)
		return
	}

	responseData, err = controller.createAuxiliaryAddress(userAsset, addressType)
	if err != nil {
		controller.Logger.Info("Error from CreateAuxiliaryAddress service : %s", err)
		ReturnError(responseWriter, "CreateAuxiliaryAddress", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERROR", errorcode.SYSTEM_ERR), controller.Logger)
		return
	}

	controller.Logger.Info("Outgoing response to CreateAuxiliaryAddress request %+v", http.StatusOK)
	responseWriter.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(responseWriter).Encode(responseData)

}

func (controller UserAssetController) VerifyDepositIsSupportedAndPopulateAsset(assetID uuid.UUID, userAsset *model.UserAsset) error {
	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetID}}, userAsset); err != nil {
		return err
	}

	// Check if deposit is ACTIVE on this asset
	userAssetService := services.NewService(controller.Cache, controller.Logger, controller.Config)
	isActive, err := userAssetService.IsDepositActive(userAsset.AssetSymbol, controller.Repository)
	if err != nil {
		return err
	}
	if !isActive {
		return errors.New(errorcode.DEPOSIT_NOT_ACTIVE)
	}
	return nil
}

func (controller UserAssetController) createAuxiliaryAddress(userAsset model.UserAsset, addressType string) (dto.AssetAddress, error) {
	var assetAddress dto.AssetAddress
	var err error
	AddressService := services.BaseService{Config: controller.Config, Cache: controller.Cache, Logger: controller.Logger}

	if userAsset.RequiresMemo {
		assetAddress, err = AddressService.CreateAuxiliaryAddressWithMemo(controller.Repository, userAsset)
		if err != nil {
			return dto.AssetAddress{}, err
		}
	} else {
		if userAsset.AssetSymbol == utility.COIN_BTC {
			assetAddress, err = AddressService.CreateAuxiliaryBTCAddress(controller.Repository, userAsset, addressType)
		} else {
			assetAddress, err = AddressService.CreateAuxiliaryAddressWithoutMemo(controller.Repository, userAsset)
		}
		if err != nil {
			return dto.AssetAddress{}, err
		}
	}
	return assetAddress, nil
}