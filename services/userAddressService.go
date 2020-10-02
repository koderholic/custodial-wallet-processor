package services

import (
	"errors"
	"fmt"
	"net/http"
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
	Repository database.IUserAssetRepository
}

func NewUserAddressService(cache *cache.Memory, config Config.Data, repository database.IUserAssetRepository) *UserAddressService {
	baseService := UserAddressService{
		Cache:      cache,
		Config:     config,
		Repository: repository,
		Error:      &dto.ExternalServicesRequestErr{},
	}
	return &baseService
}

func (service *UserAddressService) GetAddressesFor(assetID uuid.UUID, addressVersion string) (dto.AllAssetAddresses, error) {
	var addressses []dto.AssetAddress
	userAssetRepository := database.UserAssetRepository{BaseRepository: database.BaseRepository{Database: database.Database{Config: service.Config, DB: service.Repository.Db()}}}
	userAssetService := NewUserAssetService(service.Cache, service.Config, &userAssetRepository)
	userAsset, err := userAssetService.GetAssetBy(assetID)
	if err != nil {
		return dto.AssetAddress{}, err
	}

	DenominationServices := NewDenominationServices(service.Cache, service.Config, service.Repository)
	denomination, err := DenominationServices.GetDenominationByAssetSymbol(assetSymbol)
	if err != nil {
		return dto.AssetAddress{}, err
	}

	if userAsset.RequiresMemo {
		addressses[0], err = service.GetV2Address(userAsset)
	} else {
		if denomination.IsBatch {
			addressses, err = service.GetBTCAddresses(userAsset)
		} else {
			addressses.Address, err = UserAddressService.GetV1Address(userAsset)
		}
	}
	if err != nil {
		return dto.AllAssetAddresses{}, err
	}

	assetAddresses, err := service.AssetAddresses(userAsset.AssetSymbol, defaultAddress, addressses)
	if err != nil {
		return dto.AllAssetAddresses{}, err
	}

	logger.Info(fmt.Sprintf("UserAddressService logs : Address fetched for asset %v", assetID))
	return assetAddresses, nil
}

func (service *UserAddressService) AssetAddresses(assetSymbol string, addressAndMemo dto.AssetAddress) (dto.AssetAddress, error) {
	// Check if deposit is ACTIVE on this asset
	DenominationServices := NewDenominationServices(service.Cache, service.Config, service.Repository)
	isActive, err := DenominationServices.IsDepositActive(assetSymbol)
	if err != nil {
		return dto.AssetAddress{}, err
	}
	if !isActive {
		logger.Error(fmt.Sprintf("UserAddressService logs : Deposit is not available for asset %s Error : %s", assetSymbol, err))
		return dto.AssetAddress{}, appError.Err{
			ErrCode: http.StatusBadRequest,
			ErrType: errorcode.INPUT_ERR_CODE,
			Err:     errors.New(fmt.Sprintf("%s, for get %s address", errorcode.DEPOSIT_NOT_ACTIVE, assetSymbol)),
		}
	}
	return addressAndMemo, nil
}

func (service *UserAddressService) GenerateV1Address(userAsset model.UserAsset) (string, error) {
	KeyManagementService := NewKeyManagementService(service.Cache, service.Config, service.Repository)
	v1Address, err := KeyManagementService.GenerateAddress(userAsset.UserID, userAsset.AssetSymbol, userAsset.CoinType)
	if err != nil {
		logger.Error("UserAddressService logs : Error generating v1 address for asset %v, error : %s ", err)
		return v1Address, err
	}
	logger.Error("UserAddressService logs : Address generated for asset with id: %v, symbol:%s, coinType:%d", userAsset.ID, userAsset.AssetSymbol, userAsset.CoinType)
	return v1Address, nil
}

func (service *UserAddressService) GenerateV2AddressWithMemo(userAsset model.UserAsset, addressWithMemo *dto.AssetAddress) error {
	SharedAddressService := NewSharedAddressService(service.Cache, service.Config, service.Repository)
	v2Address, err := SharedAddressService.GetSharedAddressFor(service.Repository.Db(), userAsset.AssetSymbol)
	if err != nil {
		logger.Error("UserAddressService logs : Error generating v2 address for asset %v, error : %s ", err)
		return err
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

	coinTypeToAddrMap := map[int64]dto.AssetAddress{}

	var userAssets []model.UserAsset
	if err := service.Repository.GetAssetsByID(&model.UserAsset{UserID: userAsset.UserID}, &userAssets); err != nil {
		logger.Error("Error response from userAddress service, could not get user address : ", err)
		return false, err
	}

	for _, asset := range userAssets {
		userAddress := model.UserAddress{}
		if err := service.Repository.GetByFieldName(&model.UserAddress{AssetID: asset.ID}, &userAddress); err != nil {
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
	var addressType string

	err := service.Repository.GetByFieldName(&model.UserAddress{AssetID: userAsset.ID}, &userAddress)
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
			KeyManagementService := NewKeyManagementService(service.Cache, service.Config, service.Repository)
			addressResponse, err = KeyManagementService.GenerateAllAddresses(userAsset.UserID, userAsset.AssetSymbol, userAsset.CoinType, addressType)
			if err != nil {
				return "", err
			}
			userAddress.Address = addressResponse[0].Data
			userAddress.AddressType = addressResponse[0].Type
			userAddress.AssetID = userAsset.ID
		}

		if err := service.Repository.Create(&userAddress); err != nil {
			logger.Error("Error response from userAddress service, could not generate user address : %s ", err)
			return "", err
		}

	} else if err != nil {
		return "", err
	}

	return userAddress.Address, nil
}

// GetV2Address returns address with memo for mostly BEP2 assets
func (service *UserAddressService) GetV2Address(userAsset model.UserAsset) (dto.AssetAddress, error) {

	var userAddress model.UserAddress
	var assetAddress dto.AssetAddress

	err := service.Repository.GetByFieldName(&model.UserAddress{AssetID: userAsset.ID}, &userAddress)
	if (err != nil && err.Error() == errorcode.SQL_404) || (err == nil && userAddress.V2Address == "") {
		if err := service.GenerateV2AddressWithMemo(userAsset, &assetAddress); err != nil {
			return dto.AssetAddress{}, err
		}
		userAddress.AssetID = userAsset.ID
		userAddress.V2Address = assetAddress.Address
		userAddress.Memo = assetAddress.Memo

		if createErr := service.Repository.UpdateOrCreate(model.UserAddress{AssetID: userAsset.ID}, &userAddress, model.UserAddress{V2Address: userAddress.V2Address, Memo: userAddress.Memo}); createErr != nil {
			logger.Error("Error response from userAddress service, could not generate user address : %s ", err)
			return dto.AssetAddress{}, err
		}
		userAddress.V2Address = assetAddress.Address
		userAddress.Memo = assetAddress.Memo
	} else if err != nil {
		return dto.AssetAddress{}, err
	}

	return dto.AssetAddress{Address: userAddress.V2Address, Memo: userAddress.Memo}, nil
}

func (service *UserAddressService) GenerateMemo(userId uuid.UUID) (string, error) {

	// Memo lookup on the db
	memo := strconv.Itoa(utility.RandNo(100000000, 999999999))
	userMemo := model.UserMemo{
		UserID: userId,
		Memo:   memo,
	}
	if err := service.Repository.FindOrCreate(&model.UserMemo{UserID: userId}, &userMemo); err != nil {
		return "", err
	}
	// Generates a 9 digit memo
	return userMemo.Memo, nil
}

func (service *UserAddressService) CheckV2Address(address string) (bool, error) {
	sharedAddress := model.SharedAddress{}

	if err := service.Repository.GetByFieldName(&model.SharedAddress{Address: address}, &sharedAddress); err != nil {
		if err.Error() == errorcode.SQL_404 {
			return false, nil
		}
		return false, err
	}

	return true, nil

}

func (service *UserAddressService) GetBTCAddresses(userAsset model.UserAsset) ([]dto.AssetAddress, error) {

	var userAddresses []model.UserAddress
	var assetAddresses []dto.AssetAddress
	var responseAddresses []dto.AllAddressResponse

	err := service.Repository.FetchByFieldName(&model.UserAddress{AssetID: userAsset.ID}, &userAddresses)
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

	KeyManagementService := NewKeyManagementService(service.Cache, service.Config, service.Repository)
	responseAddresses, err := KeyManagementService.GenerateAllAddresses(asset.UserID, asset.AssetSymbol, asset.CoinType, addressType)
	if err != nil {
		return []dto.AllAddressResponse{}, err
	}

	for _, address := range responseAddresses {
		if err := service.Repository.Create(&model.UserAddress{Address: address.Data, AddressType: address.Type, AssetID: asset.ID}); err != nil {
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
