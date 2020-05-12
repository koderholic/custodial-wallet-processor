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
	v1Address, err := GenerateAddress(cache, logger, config, userAsset.UserID, userAsset.AssetSymbol, &externalServiceErr)
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
	var externalServiceErr dto.ServicesRequestErr
	v2Address, err := GenerateAddress(cache, logger, config, userAsset.UserID, userAsset.AssetSymbol, &externalServiceErr)
	if err != nil || v2Address == "" {
		logger.Error("Error response from userAddress service, could not generate user address : %v => %s ", externalServiceErr, err)
		if externalServiceErr.Code != "" {
			return errors.New(externalServiceErr.Message)
		}
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
		userAddress.Address = assetAddress.Address
		if !isExist {
			address, err := GenerateV1Address(logger, cache, config, userAsset)
			if err != nil {
				return "", err
			}
			userAddress.AssetID = userAsset.ID
			userAddress.Address = address

			if createErr := repository.Create(&userAddress); createErr != nil {
				logger.Error("Error response from userAddress service, could not generate user address : %s ", err)
				return "", errors.New(utility.GetSQLErr(err))
			}

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

		if createErr := repository.Create(&userAddress); createErr != nil {
			logger.Error("Error response from userAddress service, could not generate user address : %s ", err)
			return assetAddress, errors.New(utility.GetSQLErr(err))
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
