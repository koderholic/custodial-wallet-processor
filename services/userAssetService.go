package services

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
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

	return normalize(userAsset), nil
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

	return normalize(userAsset), nil
}

func (service *UserAssetService) CreditUserAsset(repository database.IUserAssetRepository, creditRequest dto.CreditUserAssetRequest, serviceID uuid.UUID) (dto.TransactionReceipt, error) {

	// ensure asset exists and fetch asset
	assetDetails := model.UserAsset{}
	if err := repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: creditRequest.AssetID}}, &assetDetails); err != nil {
		return dto.TransactionReceipt{}, serviceError(http.StatusNotFound, errorcode.RECORD_NOT_FOUND, errors.New(fmt.Sprintf("Record not found for asset with id : %v", creditRequest.AssetID)))
	}

	// increment user account by value
	newAvailableBalance := service.ComputeNewAssetBalance(assetDetails, creditRequest.Value)

	// Update asset balance
	transaction, err := UpdateAssetBalance(repository, assetDetails, creditRequest, newAvailableBalance, serviceID)
	if err != nil {
		return dto.TransactionReceipt{}, serviceError(err.(utility.AppError).ErrCode, err.(utility.AppError).ErrType, errors.New(fmt.Sprintf("User asset account (%s) could not be credited :  %s", creditRequest.AssetID, err)))
	}

	return dto.TransactionReceipt{
		AssetID:              creditRequest.AssetID,
		Value:                transaction.Value,
		TransactionReference: transaction.TransactionReference,
		PaymentReference:     transaction.PaymentReference,
		TransactionStatus:    transaction.TransactionStatus,
	}, nil
}

func (service *UserAssetService) ComputeNewAssetBalance(assetDetails model.UserAsset, creditValue float64) string {
	currentAvailableBalance := utility.Add(creditValue, assetDetails.AvailableBalance, assetDetails.Decimal)
	return currentAvailableBalance
}

func UpdateAssetBalance(repository database.IUserAssetRepository, assetDetails model.UserAsset, creditRequest dto.CreditUserAssetRequest, newAvailableBalance string, serviceID uuid.UUID) (model.Transaction, error) {
	tx := repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		return model.Transaction{}, err
	}

	if err := tx.Model(assetDetails).Updates(model.UserAsset{AvailableBalance: newAvailableBalance}).Error; err != nil {
		tx.Rollback()
		return model.Transaction{}, err
	}
	// Create transaction record
	paymentRef := utility.RandomString(16)
	value := strconv.FormatFloat(creditRequest.Value, 'g', utility.DigPrecision, 64)
	transaction := model.Transaction{
		InitiatorID:          serviceID, // serviceId
		RecipientID:          assetDetails.ID,
		TransactionReference: creditRequest.TransactionReference,
		PaymentReference:     paymentRef,
		Memo:                 creditRequest.Memo,
		TransactionType:      model.TransactionType.OFFCHAIN,
		TransactionStatus:    model.TransactionStatus.COMPLETED,
		TransactionTag:       model.TransactionTag.CREDIT,
		Value:                value,
		PreviousBalance:      assetDetails.AvailableBalance,
		AvailableBalance:     newAvailableBalance,
		ProcessingType:       model.ProcessingType.SINGLE,
		TransactionStartDate: time.Now(),
		TransactionEndDate:   time.Now(),
		AssetSymbol:          assetDetails.AssetSymbol,
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		return model.Transaction{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return model.Transaction{}, err
	}

	return transaction, nil
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
