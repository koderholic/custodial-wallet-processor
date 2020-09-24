package services

import (
	"errors"
	"strconv"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/constants"
	"wallet-adapter/utility/errorcode"
	"wallet-adapter/utility/logger"

	"wallet-adapter/utility"

	uuid "github.com/satori/go.uuid"
)

//UserAddressService object
type UserAddressService struct {
	Cache      *cache.Memory
	Config     Config.Data
	Error      *dto.ExternalServicesRequestErr
	Repository database.IRepository
}

func NewUserAddressService(cache *cache.Memory, config Config.Data, repository database.IRepository, serviceErr *dto.ExternalServicesRequestErr) *UserAddressService {
	baseService := UserAddressService{
		Cache:      cache,
		Config:     config,
		Repository: repository,
		Error:      serviceErr,
	}
	return &baseService
}

func (service *UserAddressService) GenerateV1Address(userAsset model.UserAsset) (string, error) {
	var externalServiceErr dto.ExternalServicesRequestErr
	KeyManagementService := NewKeyManagementService(service.Cache, service.Config, service.Repository, service.Error)
	v1Address, err := KeyManagementService.GenerateAddress(userAsset.UserID, userAsset.AssetSymbol, userAsset.CoinType)
	if err != nil || v1Address == "" {
		logger.Error("Error response from userAddress service, could not generate user address : %v => %s ", externalServiceErr, err)
		if externalServiceErr.Code != "" {
			return v1Address, errors.New(externalServiceErr.Message)
		}
		return v1Address, errors.New(errorcode.SERVER_ERR)
	}

	return v1Address, nil
}

func (service *UserAddressService) GenerateV2AddressWithMemo(userAsset model.UserAsset, addressWithMemo *dto.AssetAddress) error {
	repository := service.Repository.(database.IUserAddressRepository)

	SharedAddressService := NewSharedAddressService(service.Cache, service.Config, service.Repository, service.Error)
	v2Address, err := SharedAddressService.GetSharedAddressFor(repository.Db(), userAsset.AssetSymbol)
	if err != nil || v2Address == "" {
		logger.Error("Error response from shared address service : %s ", err)
		return errors.New(errorcode.SERVER_ERR)
	}
	addressWithMemo.Address = v2Address
	addressWithMemo.Memo, err = service.GenerateMemo(userAsset.UserID)
	if err != nil {
		return err
	}
	return nil
}

// CheckCoinTypeAddressExist... checks if an address has been created for one of it's user's assets with same coinType and use that instead
func (service *UserAddressService) CheckCoinTypeAddressExist(userAsset model.UserAsset, coinTypeAddress *dto.AssetAddress) (bool, error) {
	repository := service.Repository.(database.IUserAddressRepository)

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

func (service *UserAddressService) GetV1Address(userAsset model.UserAsset) (string, error) {
	var userAddress model.UserAddress
	var assetAddress dto.AssetAddress
	var addressResponse []dto.AllAddressResponse
	var externalServiceErr dto.ExternalServicesRequestErr
	var addressType string
	repository := service.Repository.(database.IUserAddressRepository)

	err := repository.GetByFieldName(&model.UserAddress{AssetID: userAsset.ID}, &userAddress)
	if (err != nil && err.Error() == errorcode.SQL_404) || (err == nil && userAddress.Address == "") {
		isExist, err := service.CheckCoinTypeAddressExist(userAsset, &assetAddress)
		if err != nil {
			return "", err
		}
		userAddress.AssetID = userAsset.ID
		userAddress.Address = assetAddress.Address
		userAddress.AddressType = assetAddress.Type
		if !isExist {
			if userAsset.AssetSymbol == constants.COIN_BTC {
				addressType = constants.ADDRESS_TYPE_SEGWIT
			}
			KeyManagementService := NewKeyManagementService(service.Cache, service.Config, service.Repository, service.Error)
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
			return "", errors.New(appError.GetSQLErr(err))
		}

	} else if err != nil {
		return "", err
	}

	return userAddress.Address, nil
}

func (service *UserAddressService) GetV2AddressWithMemo(userAsset model.UserAsset) (dto.AssetAddress, error) {
	repository := service.Repository.(database.IUserAddressRepository)

	var userAddress model.UserAddress
	var assetAddress dto.AssetAddress

	err := repository.GetByFieldName(&model.UserAddress{AssetID: userAsset.ID}, &userAddress)
	if (err != nil && err.Error() == errorcode.SQL_404) || (err == nil && userAddress.V2Address == "") {
		if err := service.GenerateV2AddressWithMemo(userAsset, &assetAddress); err != nil {
			return dto.AssetAddress{}, err
		}
		userAddress.AssetID = userAsset.ID
		userAddress.V2Address = assetAddress.Address
		userAddress.Memo = assetAddress.Memo

		if createErr := repository.UpdateOrCreate(model.UserAddress{AssetID: userAsset.ID}, &userAddress, model.UserAddress{V2Address: userAddress.V2Address, Memo: userAddress.Memo}); createErr != nil {
			logger.Error("Error response from userAddress service, could not generate user address : %s ", err)
			return dto.AssetAddress{}, errors.New(appError.GetSQLErr(err))
		}
		userAddress.V2Address = assetAddress.Address
		userAddress.Memo = assetAddress.Memo
	} else if err != nil {
		return dto.AssetAddress{}, err
	}

	return dto.AssetAddress{Address: userAddress.V2Address, Memo: userAddress.Memo}, nil
}

func (service *UserAddressService) GenerateMemo(userId uuid.UUID) (string, error) {
	repository := service.Repository.(database.IUserAddressRepository)
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

func (service *UserAddressService) CheckV2Address(address string) (bool, error) {
	sharedAddress := model.SharedAddress{}
	repository := service.Repository.(database.IUserAddressRepository)

	if err := repository.GetByFieldName(&model.SharedAddress{Address: address}, &sharedAddress); err != nil {
		if err.Error() == errorcode.SQL_404 {
			return false, nil
		}
		return false, err
	}

	return true, nil

}

func (service *UserAddressService) GetBTCAddresses(userAsset model.UserAsset) ([]dto.AssetAddress, error) {
	repository := service.Repository.(database.IUserAddressRepository)
	var userAddresses []model.UserAddress
	var assetAddresses []dto.AssetAddress
	var responseAddresses []dto.AllAddressResponse

	err := repository.FetchByFieldName(&model.UserAddress{AssetID: userAsset.ID}, &userAddresses)
	if err != nil {
		logger.Error("Error response from userAddress service, could not generate user BTC addresses : %s ", err)
		return []dto.AssetAddress{}, err
	}

	if len(userAddresses) == 0 {
		responseAddresses, err = service.GenerateAndCreateBTCAddresses(userAsset, "")
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

		if !availbleAddress[constants.ADDRESS_TYPE_SEGWIT] {
			// Create Segwit Address
			responseAddresses, err = service.GenerateAndCreateBTCAddresses(userAsset, constants.ADDRESS_TYPE_SEGWIT)
			if err != nil {
				return []dto.AssetAddress{}, err
			}
			transformedResponse := service.TransformAddressesResponse(responseAddresses)
			assetAddresses = append(assetAddresses, transformedResponse...)
		}

		if !availbleAddress[constants.ADDRESS_TYPE_LEGACY] {
			// Create Segwit Address
			responseAddresses, err = service.GenerateAndCreateBTCAddresses(userAsset, constants.ADDRESS_TYPE_LEGACY)
			if err != nil {
				return []dto.AssetAddress{}, err
			}
			transformedResponse := service.TransformAddressesResponse(responseAddresses)
			assetAddresses = append(assetAddresses, transformedResponse...)
		}

	}

	return assetAddresses, nil
}

func (service *UserAddressService) GenerateAndCreateBTCAddresses(asset model.UserAsset, addressType string) ([]dto.AllAddressResponse, error) {
	var externalServiceErr dto.ExternalServicesRequestErr
	repository := service.Repository.(database.IUserAddressRepository)

	KeyManagementService := NewKeyManagementService(service.Cache, service.Config, service.Repository, service.Error)
	responseAddresses, err := KeyManagementService.GenerateAllAddresses(asset.UserID, asset.AssetSymbol, asset.CoinType, addressType, externalServiceErr)
	if err != nil {
		return []dto.AllAddressResponse{}, err
	}

	for _, address := range responseAddresses {
		if err := repository.Create(&model.UserAddress{Address: address.Data, AddressType: address.Type, AssetID: asset.ID}); err != nil {
			logger.Error("Error response from userAddress service, could not save user BTC addresses : %s ", err)
			return []dto.AllAddressResponse{}, errors.New(appError.GetSQLErr(err))
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
