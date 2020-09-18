package services

import (
	"errors"
	"fmt"
	"net/http"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/utility"
	"wallet-adapter/utility/logger"

	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
)

//userAssetService object
type UserAssetService struct {
	Cache  *utility.MemoryCache
	Config Config.Data
	Error  *dto.ExternalServicesRequestErr
}

func NewUserAssetService(cache *utility.MemoryCache, config Config.Data) *UserAssetService {
	baseService := UserAssetService{
		Cache:  cache,
		Config: config,
	}
	return &baseService
}

// CreateUserAsset ... Create given assets for the specified user
func (service *UserAssetService) CreateAsset(repository database.IUserAssetRepository, assetDenominations []string, userID uuid.UUID) ([]dto.Asset, error) {
	assets := []dto.Asset{}
	for i := 0; i < len(assetDenominations); i++ {
		denominationSymbol := assetDenominations[i]
		denomination := model.Denomination{}

		if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: denominationSymbol, IsEnabled: true}, &denomination); err != nil {
			if err.Error() == errorcode.SQL_404 {
				return []dto.Asset{}, utility.AppError{
					ErrCode: err.(utility.AppError).ErrCode,
					ErrType: errorcode.ASSET_NOT_SUPPORTED,
					Err:     errors.New(fmt.Sprintf("Asset (%s) is currently not supported", assetDenominations[i])),
				}
			}
			return []dto.Asset{}, err
		}
		balance, _ := decimal.NewFromString("0.00")
		userAssetmodel := model.UserAsset{DenominationID: denomination.ID, UserID: userID, AvailableBalance: balance.String()}
		_ = repository.FindOrCreateAssets(model.UserAsset{DenominationID: denomination.ID, UserID: userID}, &userAssetmodel)

		asset := normalize(userAssetmodel)
		assets = append(assets, asset)
	}
	return assets, nil
}

// FetchAssets by userId
func (service *UserAssetService) FetchAssets(repository database.IUserAssetRepository, userID uuid.UUID) ([]dto.Asset, error) {

	var userAssets []model.UserAsset
	var assets []dto.Asset

	if err := repository.GetAssetsByID(&model.UserAsset{UserID: userID}, &userAssets); err != nil {
		return assets, err
	}
	if len(userAssets) < 1 {
		return assets, utility.AppError{
			ErrType: errorcode.RECORD_NOT_FOUND,
			ErrCode: http.StatusBadRequest,
			Err:     errors.New(fmt.Sprintf("No assets found for userId : %v", userID)),
		}
	}

	for i := 0; i < len(userAssets); i++ {
		userAssetmodel := userAssets[i]
		asset := normalize(userAssetmodel)
		assets = append(assets, asset)
	}

	return assets, nil
}

// GetAssetById returns user asset for given id
func (service *UserAssetService) GetAssetById(repository database.IUserAssetRepository, assetID uuid.UUID) (dto.Asset, error) {
	userAsset := model.UserAsset{}
	if err := repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetID}}, &userAsset); err != nil {
		if err.Error() == errorcode.SQL_404 {
			return dto.Asset{}, utility.AppError{
				ErrCode: err.(utility.AppError).ErrCode,
				ErrType: errorcode.RECORD_NOT_FOUND,
				Err:     errors.New(fmt.Sprintf("Asset not found for assetId > %v", assetID)),
			}
		}
		return dto.Asset{}, err
	}

	asset := normalize(userAsset)

	return asset, nil
}

func (service *UserAssetService) GetAssetByAddressSymbolAndMemo(repository database.IUserAssetRepository, address, assetSymbol, memo string) (dto.Asset, error) {
	userAsset := model.UserAsset{}
	UserAddressService := NewUserAddressService(service.Cache, service.Config)

	// Ensure assetSymbol is not empty
	if assetSymbol == "" {
		return dto.Asset{}, serviceError(http.StatusBadRequest, errorcode.INPUT_ERR_CODE, errors.New(fmt.Sprintf("assetSymbol cannot be empty")))
	}

	// Ensure Memos are provided for v2_addresses
	IsV2Address, err := UserAddressService.CheckV2Address(repository, address)
	if err != nil {
		return dto.Asset{}, serviceError(http.StatusInternalServerError, errorcode.SERVER_ERR_CODE, err)
	}

	if IsV2Address {
		userAsset, err = UserAddressService.GetAssetForV2Address(repository, address, assetSymbol, memo)
	} else {
		userAsset, err = UserAddressService.GetAssetForV1Address(repository, address, assetSymbol)
	}
	if err != nil {
		if err.Error() == errorcode.SQL_404 {
			return dto.Asset{}, serviceError(http.StatusNotFound, errorcode.RECORD_NOT_FOUND, errors.New(fmt.Sprintf("Record not found for address : %s, with asset symbol : %s and memo : %s", address, assetSymbol, memo)))
		}
	}
	logger.Info("GetUserAssetByAddress logs : Response from GetAssetForV2Address / GetAssetForV1Address for address : %v, memo : %v, assetSymbol : %s, asset : %+v", address, memo, assetSymbol, userAsset)

	asset := normalize(userAsset)
	return asset, nil
}

func normalize(userAssetmodel model.UserAsset) dto.Asset {
	userAsset := dto.Asset{}
	userAsset.ID = userAssetmodel.ID
	userAsset.UserID = userAssetmodel.UserID
	userAsset.AssetSymbol = userAssetmodel.AssetSymbol
	userAsset.AvailableBalance = userAssetmodel.AvailableBalance
	userAsset.Decimal = userAssetmodel.Decimal
	return userAsset
}

func serviceError(status int, errType string, err error) error {
	return utility.AppError{
		ErrCode: status,
		ErrType: errType,
		Err:     err,
	}
}
