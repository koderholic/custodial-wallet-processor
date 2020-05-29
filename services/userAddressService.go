package services

import (
	"errors"
	uuid "github.com/satori/go.uuid"
	"strconv"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"
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
		return v1Address, errors.New(utility.SYSTEM_ERR)
	}

	return v1Address, nil
}

func GenerateV2AddressWithMemo(repository database.IUserAssetRepository, logger *utility.Logger, cache *utility.MemoryCache, config Config.Data, userAsset model.UserAsset, addressWithMemo *dto.AddressWithMemo) error {
	v2Address, err := GetSharedAddressFor(cache, repository.Db(), logger, config, userAsset.AssetSymbol)
	if err != nil || v2Address == "" {
		logger.Error("Error response from shared address service : %s ", err)
		return errors.New(utility.SYSTEM_ERR)
	}
	addressWithMemo.Address = v2Address
	addressWithMemo.Memo, err = GenerateMemo(repository, userAsset.UserID)
	if err != nil {
		return err
	}
	return nil
}

// CheckCoinTypeAddressExist... checks if an address has been created for one of it's user's assets with same coinType and use that instead
func CheckCoinTypeAddressExist(repository database.IUserAssetRepository, logger *utility.Logger, userAsset model.UserAsset, coinTypeAddress *dto.AddressWithMemo) (bool, error) {

	coinTypeToAddrMap := map[int64]dto.AddressWithMemo{}

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
		coinTypeToAddrMap[asset.CoinType] = dto.AddressWithMemo{
			Address: userAddress.Address,
		}
	}

	if coinTypeToAddrMap[userAsset.CoinType] != (dto.AddressWithMemo{}) {
		coinTypeAddress.Address = coinTypeToAddrMap[userAsset.CoinType].Address
		return true, nil
	} else {
		return false, nil
	}
}

func GetV1Address(repository database.IUserAssetRepository, logger *utility.Logger, cache *utility.MemoryCache, config Config.Data, userAsset model.UserAsset) (string, error) {
	var userAddress model.UserAddress
	var assetAddress dto.AddressWithMemo

	err := repository.GetByFieldName(&model.UserAddress{AssetID: userAsset.ID}, &userAddress)
	if (err != nil && err.Error() == utility.SQL_404) || (err == nil && userAddress.Address == "") {
		isExist, err := CheckCoinTypeAddressExist(repository, logger, userAsset, &assetAddress)
		if err != nil {
			return "", err
		}
		userAddress.AssetID = userAsset.ID
		userAddress.Address = assetAddress.Address
		if !isExist {
			address, err := GenerateV1Address(logger, cache, config, userAsset)
			if err != nil {
				return "", err
			}
			userAddress.Address = address
		}

		if err := repository.UpdateOrCreate(model.UserAddress{AssetID: userAddress.AssetID}, &userAddress, model.UserAddress{Address: userAddress.Address}); err != nil {
			logger.Error("Error response from userAddress service, could not generate user address : %s ", err)
			return "", errors.New(utility.GetSQLErr(err))
		}

	} else if err != nil {
		return "", err
	}

	return userAddress.Address, nil
}

func GetV2AddressWithMemo(repository database.IUserAssetRepository, logger *utility.Logger, cache *utility.MemoryCache, config Config.Data, userAsset model.UserAsset) (dto.AddressWithMemo, error) {
	var userAddress model.UserAddress
	var assetAddress dto.AddressWithMemo

	err := repository.GetByFieldName(&model.UserAddress{AssetID: userAsset.ID}, &userAddress)
	if (err != nil && err.Error() == utility.SQL_404) || (err == nil && userAddress.V2Address == "") {
		if err := GenerateV2AddressWithMemo(repository, logger, cache, config, userAsset, &assetAddress); err != nil {
			return dto.AddressWithMemo{}, err
		}
		userAddress.AssetID = userAsset.ID
		userAddress.V2Address = assetAddress.Address
		userAddress.Memo = assetAddress.Memo

		if createErr := repository.UpdateOrCreate(model.UserAddress{AssetID: userAsset.ID}, &userAddress, model.UserAddress{V2Address: userAddress.V2Address, Memo: userAddress.Memo}); createErr != nil {
			logger.Error("Error response from userAddress service, could not generate user address : %s ", err)
			return dto.AddressWithMemo{}, errors.New(utility.GetSQLErr(err))
		}
		userAddress.V2Address = assetAddress.Address
		userAddress.Memo = assetAddress.Memo
	} else if err != nil {
		return dto.AddressWithMemo{}, err
	}

	return dto.AddressWithMemo{Address: userAddress.V2Address, Memo: userAddress.Memo}, nil
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
		if err.Error() == utility.SQL_404 {
			return false, nil
		}
		return false, err
	}

	return true, nil

}

func GetAssetForV1Address(repository database.IUserAssetRepository, address string, assetSymbol string) (model.UserAsset, error) {
	var userAsset model.UserAsset
	var userAddresses []model.UserAddress

	if err := repository.FetchByFieldName(&model.UserAddress{Address: address}, &userAddresses); err != nil {
		return model.UserAsset{}, err
	}

	userAsset = findMatchingAsset(repository, userAddresses, assetSymbol)

	return userAsset, nil
}

func GetAssetForV2Address(repository database.IUserAssetRepository, address string, assetSymbol string, memo string) (model.UserAsset, error) {
	var userAsset model.UserAsset
	var userAddresses []model.UserAddress

	if err := repository.FetchByFieldName(&model.UserAddress{V2Address: address, Memo: memo}, &userAddresses); err != nil {
		return model.UserAsset{}, err
	}

	userAsset = findMatchingAsset(repository, userAddresses, assetSymbol)

	return userAsset, nil
}

func findMatchingAsset(repository database.IUserAssetRepository, userAddresses []model.UserAddress, assetSymbol string) model.UserAsset {
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

	return userAsset
}

func (service BaseService) GetBTCAddresses(repository database.IUserAssetRepository, userAsset model.UserAsset) ([]dto.AddressWithMemo, error) {
	var userAddresses []model.UserAddress
	var assetAddresses []dto.AddressWithMemo
	var responseAddresses []dto.BTCAddress

	err := repository.FetchByFieldName(&model.UserAddress{AssetID: userAsset.ID}, &userAddresses)
	if err != nil {
		service.Logger.Error("Error response from userAddress service, could not generate user BTC addresses : %s ", err)
		return []dto.AddressWithMemo{}, err
	}

	if len(userAddresses) <= 0 {
		responseAddresses, err = service.GenerateAndCreateBTCAddresses(repository, userAsset, "")
		if err != nil {
			return []dto.AddressWithMemo{}, err
		}
	}

	if len(userAddresses) != 2 {
		// Create for the missing address
		availbleAddress := map[string]bool{}
		for _, address := range userAddresses {
			availbleAddress[address.AddressType] = true
		}

		if !availbleAddress[utility.LEGACY] {
			// Create Segwit Address
			responseAddresses, err = service.GenerateAndCreateBTCAddresses(repository, userAsset, utility.LEGACY)
			if err != nil {
				return []dto.AddressWithMemo{}, err
			}
		}

		if !availbleAddress[utility.SEGWIT] {
			// Create Segwit Address
			responseAddresses, err = service.GenerateAndCreateBTCAddresses(repository, userAsset, utility.SEGWIT)
			if err != nil {
				return []dto.AddressWithMemo{}, err
			}
		}

	}

	for _, item := range responseAddresses {
		address := dto.AddressWithMemo{
			Address: item.Data,
			Type:    item.Type,
		}
		assetAddresses = append(assetAddresses, address)
	}

	return assetAddresses, nil
}

func (service BaseService) GenerateAndCreateBTCAddresses(repository database.IUserAssetRepository, asset model.UserAsset, addressType string) ([]dto.BTCAddress, error) {
	var externalServiceErr dto.ServicesRequestErr
	userAddress := model.UserAddress{}

	responseAddresses, err := service.GenerateAllAddresses(asset.UserID, asset.AssetSymbol, asset.CoinType, addressType, externalServiceErr)
	if err != nil {
		return []dto.BTCAddress{}, err
	}

	for _, address := range responseAddresses {
		if err := repository.UpdateOrCreate(model.UserAddress{AssetID: asset.ID}, &userAddress, model.UserAddress{Address: address.Data, AddressType: address.Type}); err != nil {
			service.Logger.Error("Error response from userAddress service, could not save user BTC addresses : %s ", err)
			return []dto.BTCAddress{}, errors.New(utility.GetSQLErr(err))
		}
	}

	return responseAddresses, nil
}
