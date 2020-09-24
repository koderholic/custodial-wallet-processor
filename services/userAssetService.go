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
	"wallet-adapter/model"
	"wallet-adapter/utility"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/errorcode"
	"wallet-adapter/utility/logger"

	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
)

//userAssetService object
type UserAssetService struct {
	Cache      *cache.Memory
	Config     Config.Data
	Error      *dto.ExternalServicesRequestErr
	Repository database.IRepository
}

func NewUserAssetService(cache *cache.Memory, config Config.Data, repository database.IRepository, serviceErr *dto.ExternalServicesRequestErr) *UserAssetService {
	baseService := UserAssetService{
		Cache:      cache,
		Config:     config,
		Repository: repository,
		Error:      serviceErr,
	}
	return &baseService
}

// CreateUserAsset ... Create given assets for the specified user
func (service *UserAssetService) CreateAsset(assetDenominations []string, userID uuid.UUID) ([]dto.Asset, error) {
	repository := service.Repository.(database.IUserAssetRepository)
	assets := []dto.Asset{}
	for i := 0; i < len(assetDenominations); i++ {
		denominationSymbol := assetDenominations[i]
		denomination := model.Denomination{}

		if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: denominationSymbol, IsEnabled: true}, &denomination); err != nil {
			if err.Error() == errorcode.SQL_404 {
				return []dto.Asset{}, appError.Err{
					ErrCode: err.(appError.Err).ErrCode,
					ErrType: errorcode.ASSET_NOT_SUPPORTED,
					Err:     errors.New(fmt.Sprintf("Asset (%s) is currently not supported", assetDenominations[i])),
				}
			}
			return []dto.Asset{}, err
		}
		balance, _ := decimal.NewFromString("0.00")
		userAssetmodel := model.UserAsset{DenominationID: denomination.ID, UserID: userID, AvailableBalance: balance.String()}
		_ = repository.FindOrCreateAssets(model.UserAsset{DenominationID: denomination.ID, UserID: userID}, &userAssetmodel)

		asset := service.Normalize(userAssetmodel)
		assets = append(assets, asset)
	}
	return assets, nil
}

// FetchAssets by userId
func (service *UserAssetService) FetchAssets(userID uuid.UUID) ([]dto.Asset, error) {
	repository := service.Repository.(database.IUserAddressRepository)

	var userAssets []model.UserAsset
	var assets []dto.Asset

	if err := repository.GetAssetsByID(&model.UserAsset{UserID: userID}, &userAssets); err != nil {
		return assets, err
	}
	if len(userAssets) < 1 {
		return assets, appError.Err{
			ErrType: errorcode.RECORD_NOT_FOUND,
			ErrCode: http.StatusBadRequest,
			Err:     errors.New(fmt.Sprintf("No assets found for userId : %v", userID)),
		}
	}

	for i := 0; i < len(userAssets); i++ {
		userAssetmodel := userAssets[i]
		asset := service.Normalize(userAssetmodel)
		assets = append(assets, asset)
	}

	return assets, nil
}

// GetAssetById returns user asset for given id
func (service *UserAssetService) GetAssetById(assetID uuid.UUID) (dto.Asset, error) {
	userAsset, err := service.GetAssetBy(assetID)
	if err != nil {
		return dto.Asset{}, err
	}

	return service.Normalize(userAsset), nil
}

func (service *UserAssetService) GetAssetByAddressSymbolAndMemo(address, assetSymbol, memo string) (dto.Asset, error) {
	userAsset := model.UserAsset{}
	UserAddressService := NewUserAddressService(service.Cache, service.Config, service.Repository, service.Error)

	// Ensure assetSymbol is not empty
	if assetSymbol == "" {
		return dto.Asset{}, serviceError(http.StatusBadRequest, errorcode.INPUT_ERR_CODE, errors.New(fmt.Sprintf("assetSymbol cannot be empty")))
	}

	// Ensure Memos are provided for v2_addresses
	IsV2Address, err := UserAddressService.CheckV2Address(address)
	if err != nil {
		return dto.Asset{}, serviceError(http.StatusInternalServerError, errorcode.SERVER_ERR_CODE, err)
	}

	if IsV2Address {
		userAsset, err = UserAddressService.GetAssetForV2Address(address, assetSymbol, memo)
	} else {
		userAsset, err = UserAddressService.GetAssetForV1Address(address, assetSymbol)
	}
	if err != nil {
		if err.Error() == errorcode.SQL_404 {
			return dto.Asset{}, serviceError(http.StatusNotFound, errorcode.RECORD_NOT_FOUND, errors.New(fmt.Sprintf("Record not found for address : %s, with asset symbol : %s and memo : %s", address, assetSymbol, memo)))
		}
	}
	logger.Info("GetUserAssetByAddress logs : Response from GetAssetForV2Address / GetAssetForV1Address for address : %v, memo : %v, assetSymbol : %s, asset : %+v", address, memo, assetSymbol, userAsset)

	return service.Normalize(userAsset), nil
}

func (service *UserAssetService) CreditAsset(requestDetails dto.CreditUserAssetRequest, assetDetails model.UserAsset, initiatorId uuid.UUID) (dto.TransactionReceipt, error) {
	repository := service.Repository.(database.IUserAddressRepository)

	// increment user account by value
	newAssetBalance := utility.Add(requestDetails.Value, assetDetails.AvailableBalance, assetDetails.Decimal)
	transaction := BuildTxnObject(assetDetails, requestDetails, newAssetBalance, initiatorId)

	tx := database.NewTx(repository.Db())
	if err := tx.Update(&assetDetails, model.UserAsset{AvailableBalance: newAssetBalance}).
		Create(&transaction).Commit(); err != nil {
		return dto.TransactionReceipt{}, serviceError(err.(appError.Err).ErrCode, err.(appError.Err).ErrType, errors.New(fmt.Sprintf("User asset account (%s) could not be credited :  %s", requestDetails.AssetID, err)))
	}

	return TxnReceipt(transaction, requestDetails.AssetID), nil
}

func (service *UserAssetService) OnChainCreditAsset(requestDetails dto.CreditUserAssetRequest, chainData dto.ChainData, assetDetails model.UserAsset, initiatorId uuid.UUID) (dto.TransactionReceipt, error) {

	repository := service.Repository.(database.IUserAddressRepository)
	// increment user account by value
	newAssetBalance := utility.Add(requestDetails.Value, assetDetails.AvailableBalance, assetDetails.Decimal)

	transaction := BuildTxnObject(assetDetails, requestDetails, newAssetBalance, initiatorId)

	//save chain tx model first, get id and use that in Transaction model
	var chainTransaction model.ChainTransaction
	newChainTransaction := model.ChainTransaction{
		Status:           *chainData.Status,
		TransactionHash:  chainData.TransactionHash,
		TransactionFee:   chainData.TransactionFee,
		BlockHeight:      chainData.BlockHeight,
		RecipientAddress: chainData.RecipientAddress,
	}
	if err := repository.FindOrCreate(newChainTransaction, &chainTransaction); err != nil {
		err := err.(appError.Err)
		return dto.TransactionReceipt{}, serviceError(err.ErrCode, err.ErrType, err)
	}
	transactionStatus := model.TransactionStatus.PENDING
	if chainTransaction.Status == true {
		transactionStatus = model.TransactionStatus.COMPLETED
	} else {
		transactionStatus = model.TransactionStatus.REJECTED
	}
	// update transaction object
	transaction.TransactionStatus = transactionStatus
	transaction.TransactionType = model.TransactionType.ONCHAIN
	transaction.TransactionTag = model.TransactionTag.DEPOSIT
	transaction.OnChainTxId = chainTransaction.ID

	tx := database.NewTx(repository.Db())
	if err := tx.Update(&assetDetails, model.UserAsset{AvailableBalance: newAssetBalance}).
		Create(&transaction).Commit(); err != nil {
		return dto.TransactionReceipt{}, serviceError(err.(appError.Err).ErrCode, err.(appError.Err).ErrType, errors.New(fmt.Sprintf("User asset account (%s) could not be credited :  %s", requestDetails.AssetID, err)))
	}

	return TxnReceipt(transaction, requestDetails.AssetID), nil
}

func (service *UserAssetService) InternalTransfer(requestDetails dto.CreditUserAssetRequest, initiatorAssetDetails model.UserAsset, recipientAssetDetails model.UserAsset) (dto.TransactionReceipt, error) {
	repository := service.Repository.(database.IUserAddressRepository)

	// Increment initiator asset balance and decrement recipient asset balance
	initiatorCurrentBalance := utility.Subtract(requestDetails.Value, initiatorAssetDetails.AvailableBalance, initiatorAssetDetails.Decimal)
	recipientCurrentBalance := utility.Add(requestDetails.Value, recipientAssetDetails.AvailableBalance, recipientAssetDetails.Decimal)

	transaction := BuildTxnObject(initiatorAssetDetails, requestDetails, initiatorCurrentBalance, initiatorAssetDetails.ID)
	transaction.InitiatorID = initiatorAssetDetails.ID
	transaction.RecipientID = recipientAssetDetails.ID
	transaction.TransactionTag = model.TransactionTag.TRANSFER

	tx := database.NewTx(repository.Db())
	if err := tx.Update(&model.UserAsset{BaseModel: model.BaseModel{ID: initiatorAssetDetails.ID}}, model.UserAsset{AvailableBalance: initiatorCurrentBalance}).
		Update(&model.UserAsset{BaseModel: model.BaseModel{ID: recipientAssetDetails.ID}}, model.UserAsset{AvailableBalance: recipientCurrentBalance}).
		Create(&transaction).Commit(); err != nil {
		return dto.TransactionReceipt{}, err
	}

	return TxnReceipt(transaction, requestDetails.AssetID), nil

}

func (service *UserAssetService) DebitAsset(requestDetails dto.CreditUserAssetRequest, assetDetails model.UserAsset, initiatorId uuid.UUID) (dto.TransactionReceipt, error) {
	repository := service.Repository.(database.IUserAddressRepository)
	// decrement user account by value
	newAssetBalance := utility.Subtract(requestDetails.Value, assetDetails.AvailableBalance, assetDetails.Decimal)
	transaction := BuildTxnObject(assetDetails, requestDetails, newAssetBalance, initiatorId)
	transaction.TransactionTag = model.TransactionTag.DEBIT

	tx := database.NewTx(repository.Db())
	if err := tx.Update(&assetDetails, model.UserAsset{AvailableBalance: newAssetBalance}).
		Create(&transaction).Commit(); err != nil {
		return dto.TransactionReceipt{}, serviceError(err.(appError.Err).ErrCode, err.(appError.Err).ErrType, errors.New(fmt.Sprintf("User asset account (%s) could not be debited :  %s", requestDetails.AssetID, err)))
	}

	return TxnReceipt(transaction, requestDetails.AssetID), nil
}

func BuildTxnObject(assetDetails model.UserAsset, requestDetails dto.CreditUserAssetRequest, newAssetBalance string, initiatorId uuid.UUID) model.Transaction {
	// Create transaction record
	paymentRef := utility.RandomString(16)
	value := strconv.FormatFloat(requestDetails.Value, 'g', utility.DigPrecision, 64)
	return model.Transaction{
		InitiatorID:          initiatorId, // serviceId
		RecipientID:          assetDetails.ID,
		TransactionReference: requestDetails.TransactionReference,
		PaymentReference:     paymentRef,
		Memo:                 requestDetails.Memo,
		TransactionType:      model.TransactionType.OFFCHAIN,
		TransactionStatus:    model.TransactionStatus.COMPLETED,
		TransactionTag:       model.TransactionTag.CREDIT,
		Value:                value,
		PreviousBalance:      assetDetails.AvailableBalance,
		AvailableBalance:     newAssetBalance,
		ProcessingType:       model.ProcessingType.SINGLE,
		TransactionStartDate: time.Now(),
		TransactionEndDate:   time.Now(),
		AssetSymbol:          assetDetails.AssetSymbol,
	}
}

func (service *UserAssetService) GetAssetBy(id uuid.UUID) (model.UserAsset, error) {
	repository := service.Repository.(database.IUserAddressRepository)
	userAsset := model.UserAsset{}
	if err := repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: id}}, &userAsset); err != nil {
		if err.Error() == errorcode.SQL_404 {
			return userAsset, appError.Err{
				ErrCode: err.(appError.Err).ErrCode,
				ErrType: errorcode.RECORD_NOT_FOUND,
				Err:     errors.New(fmt.Sprintf("Asset not found for assetId > %v", id)),
			}
		}
		return userAsset, err
	}

	return userAsset, nil
}

func (service *UserAssetService) Normalize(userAssetmodel model.UserAsset) dto.Asset {
	userAsset := dto.Asset{}
	userAsset.ID = userAssetmodel.ID
	userAsset.UserID = userAssetmodel.UserID
	userAsset.AssetSymbol = userAssetmodel.AssetSymbol
	userAsset.AvailableBalance = userAssetmodel.AvailableBalance
	userAsset.Decimal = userAssetmodel.Decimal
	return userAsset
}

func TxnReceipt(transaction model.Transaction, assetId uuid.UUID) dto.TransactionReceipt {
	return dto.TransactionReceipt{
		AssetID:              assetId,
		Value:                transaction.Value,
		TransactionReference: transaction.TransactionReference,
		PaymentReference:     transaction.PaymentReference,
		TransactionStatus:    transaction.TransactionStatus,
	}
}

func serviceError(status int, errType string, err error) error {
	return appError.Err{
		ErrCode: status,
		ErrType: errType,
		Err:     err,
	}
}
