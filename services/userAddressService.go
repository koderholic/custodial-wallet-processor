package services

import (
	"errors"
	"strconv"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	uuid "github.com/satori/go.uuid"
)

func GenerateV1Address(logger *utility.Logger, cache *utility.MemoryCache, config Config.Data, userAsset model.UserAsset) (string, error) {
	var externalServiceErr dto.ServicesRequestErr

	// Calls key-management service to create an address for the user asset
	AddressService := BaseService{Config: config, Cache: cache, Logger: logger}
	v1Address, err := AddressService.GenerateAddress(userAsset.UserID, userAsset.AssetSymbol, userAsset.CoinType, &externalServiceErr)
	if err != nil || v1Address == "" {
		logger.Error("Error response from userAddress service, could not generate user address : %v => %s ", externalServiceErr, err)
		if externalServiceErr.Code != "" {
			return v1Address, errors.New(externalServiceErr.Message)
		}
		return v1Address, errors.New(errorcode.SYSTEM_ERR)
	}

	return v1Address, nil
}

func GenerateV2AddressWithMemo(repository database.IUserAssetRepository, logger *utility.Logger, cache *utility.MemoryCache, config Config.Data, userAsset model.UserAsset, addressWithMemo *dto.AssetAddress) error {
	v2Address, err := GetSharedAddressFor(cache, repository.Db(), logger, config, userAsset.AssetSymbol)
	if err != nil || v2Address == "" {
		logger.Error("Error response from shared address service : %s ", err)
		return errors.New(errorcode.SYSTEM_ERR)
	}
	addressWithMemo.Address = v2Address
	addressWithMemo.Memo, err = GenerateMemo(repository, userAsset.UserID)
	if err != nil {
		return err
	}
	return nil
}

// CheckCoinTypeAddressExist... checks if an address has been created for one of it's user's assets with same coinType and use that instead
func CheckCoinTypeAddressExist(repository database.IUserAssetRepository, logger *utility.Logger, userAsset model.UserAsset, coinTypeAddress *dto.AssetAddress) (bool, error) {

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

func GetV1Address(repository database.IUserAssetRepository, logger *utility.Logger, cache *utility.MemoryCache, config Config.Data, userAsset model.UserAsset) (string, error) {
	var userAddress model.UserAddress
	var assetAddress dto.AssetAddress
	var addressResponse []dto.AllAddressResponse
	var externalServiceErr dto.ServicesRequestErr
	var addressType string

	err := repository.GetByFieldName(&model.UserAddress{AssetID: userAsset.ID}, &userAddress)
	if (err != nil && err.Error() == errorcode.SQL_404) || (err == nil && userAddress.Address == "") {
		isExist, err := CheckCoinTypeAddressExist(repository, logger, userAsset, &assetAddress)
		if err != nil {
			return "", err
		}
		userAddress.AssetID = userAsset.ID
		userAddress.Address = assetAddress.Address
		userAddress.AddressType = assetAddress.Type
		if !isExist {
			AddressService := BaseService{Config: config, Cache: cache, Logger: logger}
			if userAsset.AssetSymbol == utility.COIN_BTC {
				addressType = utility.ADDRESS_TYPE_SEGWIT
			}
			addressResponse, err = AddressService.GenerateAllAddresses(userAsset.UserID, userAsset.AssetSymbol, userAsset.CoinType, addressType, externalServiceErr)
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

func GetV2AddressWithMemo(repository database.IUserAssetRepository, logger *utility.Logger, cache *utility.MemoryCache, config Config.Data, userAsset model.UserAsset) (dto.AssetAddress, error) {
	var userAddress model.UserAddress
	var assetAddress dto.AssetAddress

	err := repository.GetByFieldName(&model.UserAddress{AssetID: userAsset.ID}, &userAddress)
	if (err != nil && err.Error() == errorcode.SQL_404) || (err == nil && userAddress.V2Address == "") {
		if err := GenerateV2AddressWithMemo(repository, logger, cache, config, userAsset, &assetAddress); err != nil {
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

func GenerateMemo(repository database.IUserAssetRepository, userId uuid.UUID) (string, error) {
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
	var userAddresses []model.UserAddress

	if err := repository.FetchByFieldName(&model.UserAddress{Address: address}, &userAddresses); err != nil {
		return model.UserAsset{}, err
	}

	userAsset = findMatchingAsset(repository, logger, userAddresses, assetSymbol)

	return userAsset, nil
}

func GetAssetForV2Address(repository database.IUserAssetRepository, logger *utility.Logger, address string, assetSymbol string, memo string) (model.UserAsset, error) {
	var userAsset model.UserAsset
	var userAddresses []model.UserAddress

	if err := repository.FetchByFieldName(&model.UserAddress{V2Address: address, Memo: memo}, &userAddresses); err != nil {
		return model.UserAsset{}, err
	}
	logger.Info("GetAssetForV2Address logs : Response from FetchByFieldName %+v", userAsset)

	userAsset = findMatchingAsset(repository, logger, userAddresses, assetSymbol)

	return userAsset, nil
}

func findMatchingAsset(repository database.IUserAssetRepository, logger *utility.Logger, userAddresses []model.UserAddress, assetSymbol string) model.UserAsset {
	userAsset := model.UserAsset{}
	for _, userAddress := range userAddresses {
		asset := model.UserAsset{}
		if err := repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: userAddress.AssetID}}, &asset); err != nil {
			continue
		}
		if asset.AssetSymbol == assetSymbol {
			userAsset = asset
			break
		}
	}
	logger.Info("findMatchingAsset logs : matching asset %+v", userAsset)

	return userAsset
}

func (service BaseService) GetBTCAddresses(repository database.IUserAssetRepository, userAsset model.UserAsset) ([]dto.AssetAddress, error) {
	var userAddresses []model.UserAddress
	var assetAddresses []dto.AssetAddress
	var responseAddresses []dto.AllAddressResponse

	err := repository.FetchByFieldName(&model.UserAddress{AssetID: userAsset.ID}, &userAddresses)
	if err != nil {
		service.Logger.Error("Error response from userAddress service, could not generate user BTC addresses : %s ", err)
		return []dto.AssetAddress{}, err
	}

	if len(userAddresses) == 0 {
		responseAddresses, err = service.GenerateAndCreateBTCAddresses(repository, userAsset, "")
		if err != nil {
			return []dto.AssetAddress{}, err
		}
		assetAddresses = TransformAddressesResponse(responseAddresses)
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
			transformedResponse := TransformAddressesResponse(responseAddresses)
			assetAddresses = append(assetAddresses, transformedResponse...)
		}

		if !availbleAddress[utility.ADDRESS_TYPE_LEGACY] {
			// Create Segwit Address
			responseAddresses, err = service.GenerateAndCreateBTCAddresses(repository, userAsset, utility.ADDRESS_TYPE_LEGACY)
			if err != nil {
				return []dto.AssetAddress{}, err
			}
			transformedResponse := TransformAddressesResponse(responseAddresses)
			assetAddresses = append(assetAddresses, transformedResponse...)
		}

	}

	return assetAddresses, nil
}

func (service BaseService) GenerateAndCreateBTCAddresses(repository database.IUserAssetRepository, asset model.UserAsset, addressType string) ([]dto.AllAddressResponse, error) {
	var externalServiceErr dto.ServicesRequestErr

	responseAddresses, err := service.GenerateAllAddresses(asset.UserID, asset.AssetSymbol, asset.CoinType, addressType, externalServiceErr)
	if err != nil {
		return []dto.AllAddressResponse{}, err
	}

	for _, address := range responseAddresses {
		if err := repository.Create(&model.UserAddress{Address: address.Data, AddressType: address.Type, AssetID: asset.ID}); err != nil {
			service.Logger.Error("Error response from userAddress service, could not save user BTC addresses : %s ", err)
			return []dto.AllAddressResponse{}, errors.New(utility.GetSQLErr(err))
		}
	}

	return responseAddresses, nil
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
