package services

import (
	"errors"
	"strconv"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/utility/logger"

	"wallet-adapter/utility"

	uuid "github.com/satori/go.uuid"
)

//UserAddressService object
type UserAddressService struct {
	Cache  *utility.MemoryCache
	Config Config.Data
	Error  *dto.ExternalServicesRequestErr
}

func NewUserAddressService(cache *utility.MemoryCache, config Config.Data) *UserAddressService {
	baseService := UserAddressService{
		Cache:  cache,
		Config: config,
	}
	return &baseService
}

func (service *UserAddressService) GenerateV1Address(cache *utility.MemoryCache, config Config.Data, userAsset model.UserAsset) (string, error) {
	var externalServiceErr dto.ExternalServicesRequestErr
	KeyManagementService := NewKeyManagementService(service.Cache, service.Config)
	v1Address, err := KeyManagementService.GenerateAddress(userAsset.UserID, userAsset.AssetSymbol, userAsset.CoinType, &externalServiceErr)
	if err != nil || v1Address == "" {
		logger.Error("Error response from userAddress service, could not generate user address : %v => %s ", externalServiceErr, err)
		if externalServiceErr.Code != "" {
			return v1Address, errors.New(externalServiceErr.Message)
		}
		return v1Address, errors.New(errorcode.SERVER_ERR)
	}

	return v1Address, nil
}

func (service *UserAddressService) GenerateV2AddressWithMemo(repository database.IUserAssetRepository, cache *utility.MemoryCache, config Config.Data, userAsset model.UserAsset, addressWithMemo *dto.AssetAddress) error {

	SharedAddressService := NewSharedAddressService(service.Cache, service.Config)
	v2Address, err := SharedAddressService.GetSharedAddressFor(cache, repository.Db(), config, userAsset.AssetSymbol)
	if err != nil || v2Address == "" {
		logger.Error("Error response from shared address service : %s ", err)
		return errors.New(errorcode.SERVER_ERR)
	}
	addressWithMemo.Address = v2Address
	addressWithMemo.Memo, err = service.GenerateMemo(repository, userAsset.UserID)
	if err != nil {
		return err
	}
	return nil
}

// CheckCoinTypeAddressExist... checks if an address has been created for one of it's user's assets with same coinType and use that instead
func (service *UserAddressService) CheckCoinTypeAddressExist(repository database.IUserAssetRepository, userAsset model.UserAsset, coinTypeAddress *dto.AssetAddress) (bool, error) {

	coinTypeToAddrMap := map[int64]dto.AssetAddress{}

	var userAssets []model.UserAsset
	if err := repository.GetAssetsByID(&model.UserAsset{UserID: userAsset.UserID}, &userAssets); err != nil {
		logger.Error("Error response from userAddress service, could not get user address : ", err)
		return false, err
	}

	for _, asset := range userAssets {
		userAddress := model.UserAddress{}
		if err := repository.GetByFieldName(&model.UserAddress{AssetID: asset.ID}, &userAddress); err != nil {
			continue
		}
		coinTypeToAddrMap[asset.CoinType] = dto.AssetAddress{
			Address: userAddress.Address,
			Type:    userAddress.AddressType,
		}
	}

	if coinTypeToAddrMap[userAsset.CoinType] != (dto.AssetAddress{}) {
		coinTypeAddress.Address = coinTypeToAddrMap[userAsset.CoinType].Address
		coinTypeAddress.Type = coinTypeToAddrMap[userAsset.CoinType].Type
		return true, nil
	}
	return false, nil
}

func (service *UserAddressService) GetV1Address(repository database.IUserAssetRepository, cache *utility.MemoryCache, config Config.Data, userAsset model.UserAsset) (string, error) {
	var userAddress model.UserAddress
	var assetAddress dto.AssetAddress
	var addressResponse []dto.AllAddressResponse
	var externalServiceErr dto.ExternalServicesRequestErr
	var addressType string

	err := repository.GetByFieldName(&model.UserAddress{AssetID: userAsset.ID}, &userAddress)
	if (err != nil && err.Error() == errorcode.SQL_404) || (err == nil && userAddress.Address == "") {
		isExist, err := service.CheckCoinTypeAddressExist(repository, userAsset, &assetAddress)
		if err != nil {
			return "", err
		}
		userAddress.AssetID = userAsset.ID
		userAddress.Address = assetAddress.Address
		userAddress.AddressType = assetAddress.Type
		if !isExist {
			if userAsset.AssetSymbol == utility.COIN_BTC {
				addressType = utility.ADDRESS_TYPE_SEGWIT
			}
			KeyManagementService := NewKeyManagementService(service.Cache, service.Config)
			addressResponse, err = KeyManagementService.GenerateAllAddresses(userAsset.UserID, userAsset.AssetSymbol, userAsset.CoinType, addressType, externalServiceErr)
			if err != nil {
				return "", err
			}
			userAddress.Address = addressResponse[0].Data
			userAddress.AddressType = addressResponse[0].Type
			userAddress.AssetID = userAsset.ID
		}

		if err := repository.Create(&userAddress); err != nil {
			logger.Error("Error response from userAddress service, could not generate user address : %s ", err)
			return "", errors.New(utility.GetSQLErr(err))
		}

	} else if err != nil {
		return "", err
	}

	return userAddress.Address, nil
}

func (service *UserAddressService) GetV2AddressWithMemo(repository database.IUserAssetRepository, cache *utility.MemoryCache, config Config.Data, userAsset model.UserAsset) (dto.AssetAddress, error) {
	var userAddress model.UserAddress
	var assetAddress dto.AssetAddress

	err := repository.GetByFieldName(&model.UserAddress{AssetID: userAsset.ID}, &userAddress)
	if (err != nil && err.Error() == errorcode.SQL_404) || (err == nil && userAddress.V2Address == "") {
		if err := service.GenerateV2AddressWithMemo(repository, cache, config, userAsset, &assetAddress); err != nil {
			return dto.AssetAddress{}, err
		}
		userAddress.AssetID = userAsset.ID
		userAddress.V2Address = assetAddress.Address
		userAddress.Memo = assetAddress.Memo

		if createErr := repository.UpdateOrCreate(model.UserAddress{AssetID: userAsset.ID}, &userAddress, model.UserAddress{V2Address: userAddress.V2Address, Memo: userAddress.Memo}); createErr != nil {
			logger.Error("Error response from userAddress service, could not generate user address : %s ", err)
			return dto.AssetAddress{}, errors.New(utility.GetSQLErr(err))
		}
		userAddress.V2Address = assetAddress.Address
		userAddress.Memo = assetAddress.Memo
	} else if err != nil {
		return dto.AssetAddress{}, err
	}

	return dto.AssetAddress{Address: userAddress.V2Address, Memo: userAddress.Memo}, nil
}

func (service *UserAddressService) GenerateMemo(repository database.IUserAssetRepository, userId uuid.UUID) (string, error) {
	// Memo lookup on the db
	memo := strconv.Itoa(utility.RandNo(100000000, 999999999))
	userMemo := model.UserMemo{
		UserID: userId,
		Memo:   memo,
	}
	if err := repository.FindOrCreate(&model.UserMemo{UserID: userId}, &userMemo); err != nil {
		return "", err
	}
	// Generates a 9 digit memo
	return userMemo.Memo, nil
}

func (service *UserAddressService) CheckV2Address(repository database.IUserAssetRepository, address string) (bool, error) {
	sharedAddress := model.SharedAddress{}

	if err := repository.GetByFieldName(&model.SharedAddress{Address: address}, &sharedAddress); err != nil {
		if err.Error() == errorcode.SQL_404 {
			return false, nil
		}
		return false, err
	}

	return true, nil

}

func (service *UserAddressService) GetAssetForV1Address(repository database.IUserAssetRepository, address string, assetSymbol string) (model.UserAsset, error) {
	var userAsset model.UserAsset

	if err := repository.GetAssetByAddressAndSymbol(address, assetSymbol, &userAsset); err != nil {
		return model.UserAsset{}, err
	}
	logger.Info("GetAssetForV2Address logs : address : %s, assetSymbol : %s, assest : %+v", address, assetSymbol, userAsset)

	return userAsset, nil
}

func (service *UserAddressService) GetAssetForV2Address(repository database.IUserAssetRepository, address string, assetSymbol string, memo string) (model.UserAsset, error) {
	var userAsset model.UserAsset

	if err := repository.GetAssetByAddressAndMemo(address, memo, assetSymbol, &userAsset); err != nil {
		return model.UserAsset{}, err
	}
	logger.Info("GetAssetForV2Address logs : address : %s and memo : %s, assetSymbol : %s, assest : %+v", address, memo, assetSymbol, userAsset)

	return userAsset, nil
}

func (service *UserAddressService) GetBTCAddresses(repository database.IUserAssetRepository, userAsset model.UserAsset) ([]dto.AssetAddress, error) {
	var userAddresses []model.UserAddress
	var assetAddresses []dto.AssetAddress
	var responseAddresses []dto.AllAddressResponse

	err := repository.FetchByFieldName(&model.UserAddress{AssetID: userAsset.ID}, &userAddresses)
	if err != nil {
		logger.Error("Error response from userAddress service, could not generate user BTC addresses : %s ", err)
		return []dto.AssetAddress{}, err
	}

	if len(userAddresses) == 0 {
		responseAddresses, err = service.GenerateAndCreateBTCAddresses(repository, userAsset, "")
		if err != nil {
			return []dto.AssetAddress{}, err
		}
		assetAddresses = service.TransformAddressesResponse(responseAddresses)
	} else {
		// Create for the missing address
		availbleAddress := map[string]bool{}
		for _, address := range userAddresses {
			availbleAddress[address.AddressType] = true
			assetAddress := dto.AssetAddress{
				Address: address.Address,
				Type:    address.AddressType,
			}
			assetAddresses = append(assetAddresses, assetAddress)
		}

		if !availbleAddress[utility.ADDRESS_TYPE_SEGWIT] {
			// Create Segwit Address
			responseAddresses, err = service.GenerateAndCreateBTCAddresses(repository, userAsset, utility.ADDRESS_TYPE_SEGWIT)
			if err != nil {
				return []dto.AssetAddress{}, err
			}
			transformedResponse := service.TransformAddressesResponse(responseAddresses)
			assetAddresses = append(assetAddresses, transformedResponse...)
		}

		if !availbleAddress[utility.ADDRESS_TYPE_LEGACY] {
			// Create Segwit Address
			responseAddresses, err = service.GenerateAndCreateBTCAddresses(repository, userAsset, utility.ADDRESS_TYPE_LEGACY)
			if err != nil {
				return []dto.AssetAddress{}, err
			}
			transformedResponse := service.TransformAddressesResponse(responseAddresses)
			assetAddresses = append(assetAddresses, transformedResponse...)
		}

	}

	return assetAddresses, nil
}

func (service *UserAddressService) GenerateAndCreateBTCAddresses(repository database.IUserAssetRepository, asset model.UserAsset, addressType string) ([]dto.AllAddressResponse, error) {
	var externalServiceErr dto.ExternalServicesRequestErr

	KeyManagementService := NewKeyManagementService(service.Cache, service.Config)
	responseAddresses, err := KeyManagementService.GenerateAllAddresses(asset.UserID, asset.AssetSymbol, asset.CoinType, addressType, externalServiceErr)
	if err != nil {
		return []dto.AllAddressResponse{}, err
	}

	for _, address := range responseAddresses {
		if err := repository.Create(&model.UserAddress{Address: address.Data, AddressType: address.Type, AssetID: asset.ID}); err != nil {
			logger.Error("Error response from userAddress service, could not save user BTC addresses : %s ", err)
			return []dto.AllAddressResponse{}, errors.New(utility.GetSQLErr(err))
		}
	}

	return responseAddresses, nil
}

func (service *UserAddressService) TransformAddressesResponse(responseAddresses []dto.AllAddressResponse) []dto.AssetAddress {
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
