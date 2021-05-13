package services

import (
	"errors"
	uuid "github.com/satori/go.uuid"
	"github.com/trustwallet/blockatlas/pkg/logger"
	"strconv"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/utility"
)

func (service BaseService) GenerateV2AddressWithMemo(repository database.IUserAssetRepository, networkAsset dto.NetworkAsset,
	addressWithMemo *dto.AssetAddress, isPrimaryAddress bool) error {
	v2Address, err := GetSharedAddressFor(service.Cache, repository.Db(), service.Logger, service.Config, networkAsset.AssetSymbol, networkAsset.Network)
	if err != nil || v2Address == "" {
		logger.Error("Error response from shared address service : %s ", err)
		return errors.New(errorcode.SYSTEM_ERR)
	}
	addressWithMemo.Address = v2Address
	addressWithMemo.Memo, err = GenerateMemo(repository, networkAsset.UserID, isPrimaryAddress)
	if err != nil {
		return err
	}
	return nil
}

func GetV1Address(repository database.IUserAssetRepository, logger *utility.Logger, cache *utility.MemoryCache, config Config.Data, networkAsset dto.NetworkAsset, network string) (string, error) {
	var userAddress model.UserAddress

	err := repository.GetByFieldName(&model.UserAddress{AssetID: networkAsset.AssetID, IsPrimaryAddress: true, Network: networkAsset.Network}, &userAddress)
	if (err != nil && err.Error() == errorcode.SQL_404) || (err == nil && userAddress.Address == "") {
		userAddress.Address, err = GenerateV1Address(repository, logger, cache, config, networkAsset, userAddress, true, network)
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
	config Config.Data, networkAsset dto.NetworkAsset, userAddress model.UserAddress, isPrimaryAddress bool, network string) (string, error) {
	service := BaseService{Config: config, Cache: cache, Logger: logger}

	if networkAsset.AddressProvider == model.AddressProvider.BINANCE {
		if !isPrimaryAddress {
			return "", errors.New(errorcode.MULTIPLE_ADDRESS_ERROR)
		}
		addressResponse, err := service.GenerateUserAddressOnBBS(networkAsset.UserID, networkAsset.AssetSymbol, networkAsset.NativeAsset)
		if err != nil {
			return "", err
		}
		addressArray := []string{addressResponse.Address}
		if err := service.subscribeAddress(dto.ServicesRequestErr{}, addressArray, networkAsset.CoinType); err != nil {
			return "", err
		}
		userAddress.Address = addressResponse.Address
		userAddress.AddressProvider = model.AddressProvider.BINANCE
		userAddress.AssetID = networkAsset.AssetID
		userAddress.IsPrimaryAddress = isPrimaryAddress
		userAddress.Network = network
		userAddress.AddressType = network

	} else {
		addressResponse, err := service.GenerateAllAddresses(networkAsset.UserID, networkAsset.AssetSymbol, networkAsset.CoinType, "", network)
		if err != nil {
			return "", err
		}
		userAddress.Address = addressResponse[0].Data
		userAddress.AddressType = addressResponse[0].Type
		userAddress.AddressProvider = model.AddressProvider.BUNDLE
		userAddress.AssetID = networkAsset.AssetID
		userAddress.IsPrimaryAddress = isPrimaryAddress
		userAddress.Network = network
		userAddress.AddressType = network
	}

	if err := repository.Create(&userAddress); err != nil {
		logger.Error("Error response from userAddress service, could not generate user address : %s ", err)
		return "", errors.New(utility.GetSQLErr(err))
	}
	return userAddress.Address, nil
}

func (service BaseService) GetV2AddressWithMemo(repository database.IUserAssetRepository, networkAsset dto.NetworkAsset) (dto.AssetAddress, error) {
	var userAddress model.UserAddress
	var assetAddress dto.AssetAddress

	err := repository.GetByFieldName(&model.UserAddress{AssetID: networkAsset.AssetID, IsPrimaryAddress: true, Network: networkAsset.Network}, &userAddress)
	if (err != nil && err.Error() == errorcode.SQL_404) || (err == nil && userAddress.V2Address == "") {
		if err := service.GenerateV2AddressWithMemo(repository, networkAsset, &assetAddress, true); err != nil {
			return dto.AssetAddress{}, err
		}
		userAddress.AssetID = networkAsset.AssetID
		userAddress.V2Address = assetAddress.Address
		userAddress.AddressProvider = model.AddressProvider.BUNDLE
		userAddress.Memo = assetAddress.Memo
		userAddress.IsPrimaryAddress = true
		userAddress.Network = networkAsset.Network
		userAddress.AddressType = networkAsset.Network

		if createErr := repository.UpdateOrCreate(model.UserAddress{AssetID: networkAsset.AssetID}, &userAddress, model.UserAddress{V2Address: userAddress.V2Address, Memo: userAddress.Memo, IsPrimaryAddress: true, Network: networkAsset.Network, AddressType: networkAsset.Network}); createErr != nil {
			logger.Error("Error response from userAddress service, could not generate user address : %s ", err)
			return dto.AssetAddress{}, errors.New(utility.GetSQLErr(err))
		}
	} else if err != nil {
		return dto.AssetAddress{}, err
	}

	return dto.AssetAddress{Address: userAddress.V2Address, Memo: userAddress.Memo, Network: networkAsset.Network, Type: networkAsset.Network}, nil
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

func GetAssetForV1Address(repository database.IUserAssetRepository, logger *utility.Logger, address, assetSymbol, network string) (model.UserNetworkAsset, error) {
	var userNetworkAsset model.UserNetworkAsset
	if err := repository.GetAssetByAddressSymbolAndNetwork(address, assetSymbol, network, &userNetworkAsset); err != nil {
		logger.Info("GetAssetForV2Address logs : error with fetching asset for address : %s, assetSymbol : %s, metwork : %s, error : %+v", address, assetSymbol, network, err)
		return model.UserNetworkAsset{}, err
	}
	logger.Info("GetAssetForV1Address logs : address : %s, assetSymbol : %s, assest : %+v", address, assetSymbol, userNetworkAsset)

	return userNetworkAsset, nil
}

func GetAssetForV2Address(repository database.IUserAssetRepository, logger *utility.Logger, address, assetSymbol, memo, network string) (model.UserNetworkAsset, error) {
	var userNetworkAsset model.UserNetworkAsset
	if err := repository.GetAssetBySymbolMemoAddressAndNetwork(assetSymbol, memo, address, network, &userNetworkAsset); err != nil {
		logger.Info("GetAssetForV2Address logs : error with fetching asset for address : %s and memo : %s, assetSymbol : %s, network : %s, error : %+v", address, memo, assetSymbol, network, err)
		return model.UserNetworkAsset{}, err
	}
	logger.Info("GetAssetForV2Address logs : address : %s and memo : %s, assetSymbol : %s, assest : %+v", address, memo, assetSymbol, userNetworkAsset)

	return userNetworkAsset, nil
}

func (service BaseService) GetMultipleAddresses(repository database.IUserAssetRepository, networkAsset dto.NetworkAsset, network string) ([]dto.AssetAddress, error) {
	var userAddresses []model.UserAddress
	var assetAddresses []dto.AssetAddress
	var responseAddresses []dto.AllAddressResponse

	err := repository.FetchByFieldName(&model.UserAddress{AssetID: networkAsset.AssetID, IsPrimaryAddress: true, Network: networkAsset.Network}, &userAddresses)
	if err != nil {
		service.Logger.Error("Error response from userAddress service, could not generate user BTC addresses : %s ", err)
		return []dto.AssetAddress{}, err
	}
	if len(userAddresses) == 0 {
		responseAddresses, err = service.GenerateAndCreateAssetMultipleAddresses(repository, networkAsset, "", true, network)
		if err != nil {
			return []dto.AssetAddress{}, err
		}
		assetAddresses = TransformAddressesResponse(responseAddresses, network)
	} else {
		availableAddress := map[string]bool{}
		for _, address := range userAddresses {
			availableAddress[address.AddressType] = true
			assetAddress := dto.AssetAddress{
				Address: address.Address,
				Type:    address.AddressType,
				Network: address.Network,
			}
			assetAddresses = append(assetAddresses, assetAddress)
		}
		if len(assetAddresses) != len(utility.AddressTypesPerAsset[networkAsset.CoinType]) {
			for _, addressType := range utility.AddressTypesPerAsset[networkAsset.CoinType] {
				if !availableAddress[addressType] {
					// Create missing addressType
					responseAddresses, err = service.GenerateAndCreateAssetMultipleAddresses(repository, networkAsset, addressType, true, network)
					if err != nil {
						return []dto.AssetAddress{}, err
					}
					transformedResponse := TransformAddressesResponse(responseAddresses, network)
					assetAddresses = append(assetAddresses, transformedResponse...)
				}
			}
		}
	}

	return assetAddresses, nil
}

func (service BaseService) GenerateAndCreateAssetMultipleAddresses(repository database.IUserAssetRepository, networkAsset dto.NetworkAsset, addressType string, isPrimaryAddress bool, network string) ([]dto.AllAddressResponse, error) {

	responseAddresses, err := service.GenerateAllAddresses(networkAsset.UserID, networkAsset.AssetSymbol, networkAsset.CoinType, addressType, network)
	if err != nil {
		return []dto.AllAddressResponse{}, err
	}
	for _, address := range responseAddresses {
		if err := repository.Create(&model.UserAddress{Address: address.Data, AddressType: address.Type, AssetID: networkAsset.AssetID,
			AddressProvider: model.AddressProvider.BUNDLE, IsPrimaryAddress : isPrimaryAddress, Network: networkAsset.Network}); err != nil {
			service.Logger.Error("Error response from userAddress service, could not save user BTC addresses : %s ", err)
			return []dto.AllAddressResponse{}, errors.New(utility.GetSQLErr(err))
		}
	}

	return responseAddresses, nil
}

func (service BaseService) CreateAuxiliaryAddressWithMemo(repository database.IUserAssetRepository, networkAsset dto.NetworkAsset) (dto.AssetAddress, error) {
	var userAddress model.UserAddress
	var assetAddress dto.AssetAddress

	if err := service.GenerateV2AddressWithMemo(repository, networkAsset, &assetAddress, false); err != nil {
		return dto.AssetAddress{}, err
	}
	userAddress.AssetID = networkAsset.AssetID
	userAddress.V2Address = assetAddress.Address
	userAddress.AddressProvider = model.AddressProvider.BUNDLE
	userAddress.Memo = assetAddress.Memo
	userAddress.Network = networkAsset.Network
	userAddress.AddressProvider = networkAsset.Network

	if createErr := repository.Create(&userAddress); createErr != nil {
		logger.Error("Error response from userAddress service, could not generate user address : %s ", createErr)
		return dto.AssetAddress{}, errors.New(utility.GetSQLErr(createErr))
	}

	return dto.AssetAddress{Address: userAddress.V2Address, Memo: userAddress.Memo, Network: networkAsset.Network}, nil
}

func (service BaseService) CreateAuxiliaryBTCAddress(repository database.IUserAssetRepository, networkAsset dto.NetworkAsset, addressType, network string) (dto.AssetAddress, error) {
	var responseAddresses []dto.AllAddressResponse

	responseAddresses, err := service.GenerateAndCreateAssetMultipleAddresses(repository, networkAsset, addressType, false, network)
	if err != nil {
		return dto.AssetAddress{}, err
	}
	assetAddresses := TransformAddressesResponse(responseAddresses, network)

	return assetAddresses[0], nil
}


func (service BaseService)  CreateAuxiliaryAddressWithoutMemo(repository database.IUserAssetRepository, networkAsset dto.NetworkAsset, network string) (dto.AssetAddress, error) {
	var userAddressModel model.UserAddress
	var userAddress dto.AssetAddress
	var err error

	userAddress.Address, err = GenerateV1Address(repository, service.Logger, service.Cache, service.Config, networkAsset, userAddressModel, false, network)
	if err != nil {
		return dto.AssetAddress{}, err
	}
	userAddress.Type = network

	return userAddress, nil
}


func TransformAddressesResponse(responseAddresses []dto.AllAddressResponse, network string) []dto.AssetAddress {
	assetAddresses := []dto.AssetAddress{}
	for _, item := range responseAddresses {
		address := dto.AssetAddress{
			Address: item.Data,
			Type:    item.Type,
			Network: network,
		}
		assetAddresses = append(assetAddresses, address)
	}
	return assetAddresses
}
