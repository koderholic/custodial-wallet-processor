package services

import (
	"errors"
	"fmt"
	"github.com/trustwallet/blockatlas/pkg/logger"
	"regexp"
	"strconv"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/utility"
	"wallet-adapter/utility/constants"

	uuid "github.com/satori/go.uuid"
)

func (service BaseService) GenerateV2AddressWithMemo(repository database.IUserAssetRepository, userAsset model.UserAsset,
	addressWithMemo *dto.AssetAddress, isPrimaryAddress bool) error {
	v2Address, err := GetSharedAddressFor(service.Cache, repository.Db(), service.Logger, service.Config, userAsset.AssetSymbol)
	if err != nil || v2Address == "" {
		logger.Error("Error response from shared address service : %s ", err)
		return errors.New(errorcode.SYSTEM_ERR)
	}
	addressWithMemo.Address = v2Address
	addressWithMemo.Memo, err = GenerateMemo(repository, userAsset.UserID, isPrimaryAddress)
	if err != nil {
		return err
	}
	return nil
}

func GetV1Address(repository database.IUserAssetRepository, logger *utility.Logger, cache *utility.MemoryCache, config Config.Data, userAsset model.UserAsset) (string, error) {
	var userAddress model.UserAddress

	err := repository.GetByFieldName(&model.UserAddress{AssetID: userAsset.ID, IsPrimaryAddress: true}, &userAddress)
	if (err != nil && err.Error() == errorcode.SQL_404) || (err == nil && userAddress.Address == "") {
		userAddress.Address, err = GenerateV1Address(repository, logger, cache, config, userAsset, userAddress, true)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	return userAddress.Address, nil
}

func GetBinanceProvidedAddressforAsset(repository database.IUserAssetRepository, userAssetId uuid.UUID) (string, error) {
	var userAddress model.UserAddress
	err := repository.GetByFieldName(&model.UserAddress{AssetID: userAssetId, AddressProvider: model.AddressProvider.BINANCE, IsPrimaryAddress: true}, &userAddress)
	if err != nil {
		return "", err
	}
	return userAddress.Address, nil
}

func GenerateV1Address(repository database.IUserAssetRepository, logger *utility.Logger, cache *utility.MemoryCache,
	config Config.Data, userAsset model.UserAsset, userAddress model.UserAddress, isPrimaryAddress bool) (string, error) {
	service := BaseService{Config: config, Cache: cache, Logger: logger}

	if userAsset.AddressProvider == model.AddressProvider.BINANCE {
		if !isPrimaryAddress {
			return "", errors.New(errorcode.MULTIPLE_ADDRESS_ERROR)
		}
		addressResponse, err := service.GenerateUserAddressOnBBS(userAsset.UserID, userAsset.AssetSymbol, "")
		if err != nil {
			return "", err
		}
		addressArray := []string{addressResponse.Address}
		if err := service.subscribeAddress(dto.ServicesRequestErr{}, addressArray, userAsset.CoinType); err != nil {
			return "", err
		}
		userAddress.Address = addressResponse.Address
		userAddress.AddressProvider = model.AddressProvider.BINANCE
		userAddress.AssetID = userAsset.ID
		userAddress.IsPrimaryAddress = isPrimaryAddress

	} else {
		addressResponse, err := service.GenerateAllAddresses(userAsset.UserID, userAsset.AssetSymbol, userAsset.CoinType, "")
		if err != nil {
			return "", err
		}
		userAddress.Address = addressResponse[0].Data
		userAddress.AddressType = addressResponse[0].Type
		userAddress.AddressProvider = model.AddressProvider.BUNDLE
		userAddress.AssetID = userAsset.ID
		userAddress.IsPrimaryAddress = isPrimaryAddress
	}

	if err := repository.Create(&userAddress); err != nil {
		logger.Error("Error response from userAddress service, could not generate user address : %s ", err)
		return "", errors.New(utility.GetSQLErr(err))
	}
	return userAddress.Address, nil
}

func (service BaseService) GetV2AddressWithMemo(repository database.IUserAssetRepository, userAsset model.UserAsset) (dto.AssetAddress, error) {
	var userAddress model.UserAddress
	var assetAddress dto.AssetAddress

	err := repository.GetByFieldName(&model.UserAddress{AssetID: userAsset.ID, IsPrimaryAddress: true}, &userAddress)
	if (err != nil && err.Error() == errorcode.SQL_404) || (err == nil && userAddress.V2Address == "") {
		if err := service.GenerateV2AddressWithMemo(repository, userAsset, &assetAddress, true); err != nil {
			return dto.AssetAddress{}, err
		}
		userAddress.AssetID = userAsset.ID
		userAddress.V2Address = assetAddress.Address
		userAddress.AddressProvider = model.AddressProvider.BUNDLE
		userAddress.Memo = assetAddress.Memo
		userAddress.IsPrimaryAddress = true

		if createErr := repository.UpdateOrCreate(model.UserAddress{AssetID: userAsset.ID}, &userAddress, model.UserAddress{V2Address: userAddress.V2Address, Memo: userAddress.Memo, IsPrimaryAddress: true}); createErr != nil {
			logger.Error("Error response from userAddress service, could not generate user address : %s ", err)
			return dto.AssetAddress{}, errors.New(utility.GetSQLErr(err))
		}
	} else if err != nil {
		return dto.AssetAddress{}, err
	}

	return dto.AssetAddress{Address: userAddress.V2Address, Memo: userAddress.Memo}, nil
}

func GenerateMemo(repository database.IUserAssetRepository, userId uuid.UUID, isPrimaryAddress bool) (string, error) {
	// Memo lookup on the db
	memo := strconv.Itoa(utility.RandNo(100000000, 999999999))
	userMemo := model.UserMemo{
		UserID: userId,
		Memo:   memo,
		IsPrimaryAddress: isPrimaryAddress,
	}

	if !isPrimaryAddress {
		if err := repository.Create(&userMemo); err != nil {
			return "", err
		}
		return userMemo.Memo, nil
	}
	if err := repository.FindOrCreate(&model.UserMemo{UserID: userId, IsPrimaryAddress: isPrimaryAddress}, &userMemo); err != nil {
		return "", err
	}
	// Generates a 9 digit memo
	return userMemo.Memo, nil
}

func CheckV2Address(repository database.IUserAssetRepository, address string) (bool, error) {
	sharedAddress := model.SharedAddress{}

	if err := repository.GetByFieldName(&model.SharedAddress{Address: address}, &sharedAddress); err != nil {
		if err.Error() == errorcode.SQL_404 {
			return false, nil
		}
		return false, err
	}

	return true, nil

}

func GetAssetForV1Address(repository database.IUserAssetRepository, logger *utility.Logger, address string, assetSymbol string) (model.UserAsset, error) {
	var userAsset model.UserAsset

	if err := repository.GetAssetByAddressAndSymbol(address, assetSymbol, &userAsset); err != nil {

		/*if err.Error() != errorcode.SQL_404 {
			logger.Info("GetAssetForV2Address logs : error with fetching asset for address : %s, assetSymbol : %s, error : %+v", address, assetSymbol, err)
			return model.UserAsset{}, err
		}
		asset, err2 := GetUserAssetForSupportedERC20TokeOrETH(repository, address, assetSymbol, userAsset)
		if err2 == nil {
			return asset, nil
		}

		logger.Info("GetAssetForV2Address logs : error with fetching asset for address : %s, assetSymbol : %s, error : %+v", address, assetSymbol, err2)*/
		logger.Info("GetAssetForV2Address logs : error with fetching asset for address : %s, assetSymbol : %s, error : %+v", address, assetSymbol, err)
		return model.UserAsset{}, err
	}
	logger.Info("GetAssetForV1Address logs : address : %s, assetSymbol : %s, assest : %+v", address, assetSymbol, userAsset)

	return userAsset, nil
}

func GetUserAssetForSupportedERC20TokeOrETH(repository database.IUserAssetRepository, address string, assetSymbol string, userAsset model.UserAsset) (model.UserAsset, error) {

	isETHAddr, err := regexp.MatchString("^(0x)[0-9A-Fa-f]{40}$", address)
	if err != nil || !isETHAddr {
		return model.UserAsset{}, errors.New(fmt.Sprintf("Asset not found, more context : %s", err))
	}

	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: assetSymbol, IsEnabled: true}, &denomination);
		err != nil || denomination.CoinType != constants.ETH_COINTYPE {
		return model.UserAsset{}, errors.New(fmt.Sprintf("Asset not found, more context : %s", err))
	}

	var assetAddressDetails model.UserAsset
	if err := repository.GetAssetAddressDetails(address, &assetAddressDetails); err != nil {
		return model.UserAsset{}, err
	}

	if assetAddressDetails.AddressProvider != model.AddressProvider.BINANCE {
		return model.UserAsset{}, errors.New("Address is not a binance provided address, cannot recover token or ETH sent")
	}

	if err := repository.GetAssetsByID(&model.UserAsset{UserID: assetAddressDetails.UserID, DenominationID: denomination.ID}, &userAsset); err != nil {
		return model.UserAsset{}, err
	}

	return userAsset, nil
}

func GetAssetForV2Address(repository database.IUserAssetRepository, logger *utility.Logger, address string, assetSymbol string, memo string) (model.UserAsset, error) {
	var userAsset model.UserAsset

	if err := repository.GetAssetBySymbolMemoAndAddress(assetSymbol, memo, address, &userAsset); err != nil {
		logger.Info("GetAssetForV2Address logs : error with fetching asset for address : %s and memo : %s, assetSymbol : %s, error : %+v", address, memo, assetSymbol, err)
		return model.UserAsset{}, err
	}
	logger.Info("GetAssetForV2Address logs : address : %s and memo : %s, assetSymbol : %s, assest : %+v", address, memo, assetSymbol, userAsset)

	return userAsset, nil
}

func (service BaseService) GetMultipleAddresses(repository database.IUserAssetRepository, userAsset model.UserAsset) ([]dto.AssetAddress, error) {
	var userAddresses []model.UserAddress
	var assetAddresses []dto.AssetAddress
	var responseAddresses []dto.AllAddressResponse

	err := repository.FetchByFieldName(&model.UserAddress{AssetID: userAsset.ID, IsPrimaryAddress: true}, &userAddresses)
	if err != nil {
		service.Logger.Error("Error response from userAddress service, could not generate user BTC addresses : %s ", err)
		return []dto.AssetAddress{}, err
	}
	if len(userAddresses) == 0 {
		responseAddresses, err = service.GenerateAndCreateAssetMultipleAddresses(repository, userAsset, "", true)
		if err != nil {
			return []dto.AssetAddress{}, err
		}
		assetAddresses = TransformAddressesResponse(responseAddresses)
	} else {
		availableAddress := map[string]bool{}
		for _, address := range userAddresses {
			availableAddress[address.AddressType] = true
			assetAddress := dto.AssetAddress{
				Address: address.Address,
				Type:    address.AddressType,
			}
			assetAddresses = append(assetAddresses, assetAddress)
		}
		if len(assetAddresses) != len(utility.AddressTypesPerAsset[userAsset.CoinType]) {
			for _, addressType := range utility.AddressTypesPerAsset[userAsset.CoinType] {
				if !availableAddress[addressType] {
					// Create missing addressType
					responseAddresses, err = service.GenerateAndCreateAssetMultipleAddresses(repository, userAsset, addressType, true)
					if err != nil {
						return []dto.AssetAddress{}, err
					}
					transformedResponse := TransformAddressesResponse(responseAddresses)
					assetAddresses = append(assetAddresses, transformedResponse...)
				}
			}
		}
	}

	return assetAddresses, nil
}

func (service BaseService) GenerateAndCreateAssetMultipleAddresses(repository database.IUserAssetRepository, asset model.UserAsset, addressType string, isPrimaryAddress bool) ([]dto.AllAddressResponse, error) {

	responseAddresses, err := service.GenerateAllAddresses(asset.UserID, asset.AssetSymbol, asset.CoinType, addressType)
	if err != nil {
		return []dto.AllAddressResponse{}, err
	}
	for _, address := range responseAddresses {
		if err := repository.Create(&model.UserAddress{Address: address.Data, AddressType: address.Type, AssetID: asset.ID,
			AddressProvider: model.AddressProvider.BUNDLE, IsPrimaryAddress : isPrimaryAddress}); err != nil {
			service.Logger.Error("Error response from userAddress service, could not save user BTC addresses : %s ", err)
			return []dto.AllAddressResponse{}, errors.New(utility.GetSQLErr(err))
		}
	}

	return responseAddresses, nil
}

func (service BaseService) CreateAuxiliaryAddressWithMemo(repository database.IUserAssetRepository, userAsset model.UserAsset) (dto.AssetAddress, error) {
	var userAddress model.UserAddress
	var assetAddress dto.AssetAddress

	if err := service.GenerateV2AddressWithMemo(repository, userAsset, &assetAddress, false); err != nil {
		return dto.AssetAddress{}, err
	}
	userAddress.AssetID = userAsset.ID
	userAddress.V2Address = assetAddress.Address
	userAddress.AddressProvider = model.AddressProvider.BUNDLE
	userAddress.Memo = assetAddress.Memo

	if createErr := repository.Create(&userAddress); createErr != nil {
		logger.Error("Error response from userAddress service, could not generate user address : %s ", createErr)
		return dto.AssetAddress{}, errors.New(utility.GetSQLErr(createErr))
	}

	return dto.AssetAddress{Address: userAddress.V2Address, Memo: userAddress.Memo}, nil
}

func (service BaseService) CreateAuxiliaryBTCAddress(repository database.IUserAssetRepository, userAsset model.UserAsset, addressType string) (dto.AssetAddress, error) {
	var responseAddresses []dto.AllAddressResponse

	responseAddresses, err := service.GenerateAndCreateAssetMultipleAddresses(repository, userAsset, addressType, false)
	if err != nil {
		return dto.AssetAddress{}, err
	}
	assetAddresses := TransformAddressesResponse(responseAddresses)

	return assetAddresses[0], nil
}


func (service BaseService)  CreateAuxiliaryAddressWithoutMemo(repository database.IUserAssetRepository, userAsset model.UserAsset) (dto.AssetAddress, error) {
	var userAddressModel model.UserAddress
	var userAddress dto.AssetAddress
	var err error

	userAddress.Address, err = GenerateV1Address(repository, service.Logger, service.Cache, service.Config, userAsset, userAddressModel, false)
	if err != nil {
		return dto.AssetAddress{}, err
	}

	return userAddress, nil
}


func TransformAddressesResponse(responseAddresses []dto.AllAddressResponse) []dto.AssetAddress {
	assetAddresses := []dto.AssetAddress{}
	for _, item := range responseAddresses {
		address := dto.AssetAddress{
			Address: item.Data,
			Type:    item.Type,
		}
		assetAddresses = append(assetAddresses, address)
	}
	return assetAddresses
}
