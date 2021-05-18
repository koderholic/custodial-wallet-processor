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
	var coinType int64
	var responseData dto.AllAssetAddresses
	apiResponse := utility.NewResponse()
	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		ReturnError(responseWriter, "GetAllAssetAddresses", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR", errorcode.UUID_CAST_ERR), controller.Logger)
		return
	}

	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetID}}, &userAsset); err != nil {
		ReturnError(responseWriter, "GetAllAssetAddresses", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR",
			fmt.Sprintf("Failed to get asset record for %s", assetID)), controller.Logger)
		return
	}

	userAddressService := services.NewService(controller.Cache, controller.Logger, controller.Config)
	//GET all networks for the asset, loop through and create addresses for each network with deposit as ACTIVE
	assetNetworks, err := userAddressService.GetNetworksByDenom(controller.Repository, userAsset.AssetSymbol)
	if err != nil {
		ReturnError(responseWriter, "GetAllAssetAddresses", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR",
			fmt.Sprintf("Failed to get network record for %s", userAsset.AssetSymbol)), controller.Logger)
		return
	}

	if len(assetNetworks) > 0 {
		for _, network := range assetNetworks {
			if  userAsset.DefaultNetwork == network.Network {
				coinType = network.CoinType
			}
			// Check if deposit is ACTIVE for the asset on this network
			isActive, err := userAddressService.IsDepositActive(userAsset.AssetSymbol, network.Network, controller.Repository)
			if err != nil || !isActive  {
				controller.Logger.Debug("%s for %s network", errorcode.DEPOSIT_NOT_ACTIVE, network.Network)
				continue
			}
			networkAsset := mapNetworkToAssetStruct(network, userAsset)

			addresses, err := controller.GetAddressesForNetwork(networkAsset, userAddressService)
			if err != nil {
				controller.Logger.Info("could not get addresses for additional networks %s, error : %s", networkAsset.Network, err)
				continue
			}
			responseData.Addresses = append(responseData.Addresses, addresses...)
		}
	}

	if len(responseData.Addresses) == 0 {
		ReturnError(responseWriter, "GetAllAssetAddresses", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR",
			fmt.Sprintf("No addresses found for asset, ensure deposit is ACTIVE for asset %s", assetID)), controller.Logger)
		return
	}

	responseData.DefaultAddressType = userAsset.DefaultNetwork
	responseData.DefaultNetwork = userAsset.DefaultNetwork
	if len(utility.AddressTypesPerAsset[coinType]) > 0 {
		responseData.DefaultAddressType = utility.AddressTypesPerAsset[coinType][0]
	}

	controller.Logger.Info("Outgoing response to GetAllAssetAddresses request %+v", http.StatusOK)
	responseWriter.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(responseWriter).Encode(responseData)

}

func mapNetworkToAssetStruct(network model.Network, userAsset model.UserAsset) dto.NetworkAsset {
	networkAsset := dto.NetworkAsset{
		UserID:          userAsset.UserID,
		AssetID: userAsset.ID,
		DefaultNetwork: userAsset.DefaultNetwork,
		DenominationID:  userAsset.DenominationID,
		AssetSymbol:     userAsset.AssetSymbol,
		NativeAsset:     network.NativeAsset,
		NativeDecimals:  network.NativeDecimals,
		CoinType:        network.CoinType,
		RequiresMemo:    network.RequiresMemo,
		AddressProvider: network.AddressProvider,
		Network:         network.Network,
		IsMultiAddresses: network.IsMultiAddresses,
		IsBatchable: network.IsBatchable,
		IsToken: network.IsToken,
		DepositActivity: network.DepositActivity,
		WithdrawActivity: network.WithdrawActivity,

	}
	return networkAsset
}

func (controller UserAssetController) GetAddressesForNetwork(networkAsset dto.NetworkAsset, userAddressService *services.BaseService ) ([]dto.AssetAddress,  error) {
	addresses := []dto.AssetAddress{}

	if networkAsset.RequiresMemo {
		v2Address, err := userAddressService.GetV2AddressWithMemo(controller.Repository, networkAsset)
		if err != nil {
			controller.Logger.Info("Error from GetV2AddressWithMemo service : %s", err)
			return addresses, err
		}
		addresses = append(addresses, dto.AssetAddress{
			Address: v2Address.Address,
			Memo:    v2Address.Memo,
			Network: networkAsset.Network,
			Type: networkAsset.Network,
		})
	} else {
		var err error
		var address string
		IsMultiAddresses, err := userAddressService.IsMultipleAddresses(networkAsset.AssetSymbol, networkAsset.Network, controller.Repository)
		if err != nil {
			controller.Logger.Info("Error from GetV2AddressWithMemo service : %s", err)
			return addresses, err
		}

		if IsMultiAddresses {
			addresses, err = userAddressService.GetMultipleAddresses(controller.Repository, networkAsset, networkAsset.Network)
		} else {
			address, err = services.GetV1Address(controller.Repository, controller.Logger, controller.Cache, controller.Config, networkAsset)
			addresses = append(addresses, dto.AssetAddress{
				Address: address,
				Network: networkAsset.Network,
				Type: networkAsset.Network,
			})
		}
		if err != nil {
			return addresses, err
		}
	}
	return addresses, nil
}

func (controller UserAssetController) CreateAuxiliaryAddress(responseWriter http.ResponseWriter, requestReader *http.Request)  {

	var userAsset model.UserAsset
	var responseData dto.AssetAddress
	apiResponse := utility.NewResponse()
	routeParams := mux.Vars(requestReader)
	addressType := requestReader.URL.Query().Get("addressType")
	network := requestReader.URL.Query().Get("network")

	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		ReturnError(responseWriter, "CreateAuxiliaryAddress", http.StatusBadRequest, err, apiResponse.
			PlainError("INPUT_ERR", errorcode.UUID_CAST_ERR), controller.Logger)
		return
	}

	if err := controller.VerifyDepositIsSupportedAndPopulateAsset(assetID, network, &userAsset); err != nil {
		ReturnError(responseWriter, "CreateAuxiliaryAddress", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR",
			fmt.Sprintf("%s, for get asset address with id = %s", errorcode.DEPOSIT_NOT_ACTIVE, assetID)), controller.Logger)
		return
	}

	if network == "" {
		network = userAsset.DefaultNetwork
	}

	networkRecord, err := services.GetNetworkByAssetAndNetwork(controller.Repository, network, userAsset.AssetSymbol)
	if err != nil {
		ReturnError(responseWriter, "CreateAuxiliaryAddress", http.StatusInternalServerError, err,
			apiResponse.PlainError("SYSTEM_ERROR", errorcode.SYSTEM_ERR), controller.Logger)
		return
	}
	networkAsset := mapNetworkToAssetStruct(networkRecord, userAsset)

	responseData, err = controller.createAuxiliaryAddress(networkAsset, addressType)
	if err != nil {
		controller.Logger.Info("Error from CreateAuxiliaryAddress service : %s", err)
		if err.Error() == errorcode.MULTIPLE_ADDRESS_ERROR {
			ReturnError(responseWriter, "CreateAuxiliaryAddress", http.StatusInternalServerError, err,
				apiResponse.PlainError(errorcode.MULTIPLE_ADDRESS_ERROR_CODE, err.Error()), controller.Logger)
			return
		}
		ReturnError(responseWriter, "CreateAuxiliaryAddress", http.StatusInternalServerError, err,
			apiResponse.PlainError("SYSTEM_ERROR", errorcode.SYSTEM_ERR), controller.Logger)
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(responseWriter).Encode(responseData)

}

func (controller UserAssetController) VerifyDepositIsSupportedAndPopulateAsset(assetID uuid.UUID, network string, userAsset *model.UserAsset) error {
	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetID}}, userAsset); err != nil {
		return err
	}
	if network == "" {
		network = userAsset.DefaultNetwork
	}

	// Check if deposit is ACTIVE on this asset
	userAssetService := services.NewService(controller.Cache, controller.Logger, controller.Config)
	isActive, err := userAssetService.IsDepositActive(userAsset.AssetSymbol, network, controller.Repository)
	if err != nil {
		return err
	}
	if !isActive {
		return errors.New(errorcode.DEPOSIT_NOT_ACTIVE)
	}
	return nil
}

func (controller UserAssetController) createAuxiliaryAddress(networkAsset dto.NetworkAsset, addressType string) (dto.AssetAddress, error) {
	var assetAddress dto.AssetAddress
	var err error
	AddressService := services.BaseService{Config: controller.Config, Cache: controller.Cache, Logger: controller.Logger}

	if networkAsset.RequiresMemo {
		assetAddress, err = AddressService.CreateAuxiliaryAddressWithMemo(controller.Repository, networkAsset)
		if err != nil {
			return dto.AssetAddress{}, err
		}
	} else {
		if networkAsset.AssetSymbol == utility.COIN_BTC {
			assetAddress, err = AddressService.CreateAuxiliaryBTCAddress(controller.Repository, networkAsset, addressType, networkAsset.Network)
		} else {
			assetAddress, err = AddressService.CreateAuxiliaryAddressWithoutMemo(controller.Repository, networkAsset, networkAsset.Network)
		}
		if err != nil {
			return dto.AssetAddress{}, err
		}
	}
	return assetAddress, nil
}