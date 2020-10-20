package services

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/constants"
	"wallet-adapter/utility/errorcode"
	"wallet-adapter/utility/logger"

	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
)

//TransactionService object
type TransactionService struct {
	Cache      *cache.Memory
	Config     Config.Data
	Error      *dto.ExternalServicesRequestErr
	Repository database.ITransactionRepository
}

func NewTransactionService(cache *cache.Memory, config Config.Data, repository database.ITransactionRepository) *TransactionService {
	baseService := TransactionService{
		Cache:      cache,
		Config:     config,
		Repository: repository,
		Error:      &dto.ExternalServicesRequestErr{},
	}
	return &baseService
}

func (service *TransactionService) GetTransaction(reference string) (model.Transaction, error) {
	transaction, err := service.GetTransactionByRef(reference)
	if err != nil {
		return model.Transaction{}, err
	}
	if transaction.TransactionStatus == model.TransactionStatus.PROCESSING && transaction.TransactionType == model.TransactionType.ONCHAIN {
		transactionQueue, err := service.GetQueuedTransactionByTxId(transaction.ID)
		if err != nil {
			return model.Transaction{}, err
		}
		txnExist, broadcastedTX, err := service.VerifyBroadcastedTx(transaction)
		if err != nil {
			logger.Error("verifyTransactionStatus logs : Error checking the broadcasted state for transaction (%v) : %s", transaction.ID, err)
			return model.Transaction{}, err
		}
		status, err := service.GetTransactionStatus(txnExist, broadcastedTX, transaction, transactionQueue)
		if status != "" {
			transaction.TransactionStatus = status
		}
	}
	return transaction, nil
}

func (service *TransactionService) GetAllAssetTransactions(assetID uuid.UUID) ([]model.Transaction, error) {
	var transactions []model.Transaction
	initiatorTransactions, err := service.GetTransactionByInitiatorID(assetID)
	if err != nil {
		return nil, err
	}
	recipientTransactions, err := service.GetTransactionByRecipientID(assetID)
	if err != nil {
		return nil, err
	}
	transactions = append(transactions, initiatorTransactions, recipientTransactions)

	return transactions, err
}

func (service *TransactionService) GetTransactionByRef(reference string) (model.Transaction, error) {

	transaction := model.Transaction{}
	if err := service.Repository.GetByFieldName(&model.Transaction{TransactionReference: reference}, &transaction); err != nil {
		appErr := err.(appError.Err)
		return model.Transaction{}, serviceError(appErr.ErrCode, appErr.ErrType, errors.New(fmt.Sprintf(`Error fetching transaction record for reference : %s, additional context : %s`, reference, appErr)))
	}
	return transaction, nil
}

func (service *TransactionService) GetTransactionByInitiatorID(assetID uuid.UUID) (model.Transaction, error) {

	transaction := model.Transaction{}
	if err := service.Repository.GetByFieldName(&model.Transaction{InitiatorID: assetID}, &transaction); err != nil {
		appErr := err.(appError.Err)
		return model.Transaction{}, serviceError(appErr.ErrCode, appErr.ErrType, errors.New(fmt.Sprintf(`Error fetching transaction record for id : %v, additional context : %s`, assetID, appErr)))
	}
	return transaction, nil
}

func (service *TransactionService) GetTransactionByRecipientID(assetID uuid.UUID) (model.Transaction, error) {

	transaction := model.Transaction{}
	if err := service.Repository.GetByFieldName(&model.Transaction{RecipientID: assetID}, &transaction); err != nil {
		appErr := err.(appError.Err)
		return model.Transaction{}, serviceError(appErr.ErrCode, appErr.ErrType, errors.New(fmt.Sprintf(`Error fetching transaction record for id : %v, additional context : %s`, assetID, appErr)))
	}
	return transaction, nil
}

func (service *TransactionService) GetTransactionById(id uuid.UUID) (model.Transaction, error) {

	transaction := model.Transaction{}
	if err := service.Repository.GetByFieldName(&model.Transaction{BaseModel: model.BaseModel{ID: id}}, &transaction); err != nil {
		appErr := err.(appError.Err)
		return model.Transaction{}, serviceError(appErr.ErrCode, appErr.ErrType, errors.New(fmt.Sprintf(`Error fetching transaction record for id : %v, additional context : %s`, id, appErr)))
	}
	return transaction, nil
}

func (service *TransactionService) GetQueuedTransactionById(id uuid.UUID) (model.TransactionQueue, error) {

	transaction := model.TransactionQueue{}
	if err := service.Repository.GetByFieldName(&model.TransactionQueue{BaseModel: model.BaseModel{ID: id}}, &transaction); err != nil {
		appErr := err.(appError.Err)
		return model.TransactionQueue{}, serviceError(appErr.ErrCode, appErr.ErrType, errors.New(fmt.Sprintf(`Error fetching transaction record for id : %v, additional context : %s`, id, appErr)))
	}
	return transaction, nil
}

func (service *TransactionService) GetQueuedTransactionByTxId(txID uuid.UUID) (model.TransactionQueue, error) {

	transaction := model.TransactionQueue{}
	if err := service.Repository.GetByFieldName(&model.TransactionQueue{TransactionId: txID}, &transaction); err != nil {
		appErr := err.(appError.Err)
		return model.TransactionQueue{}, serviceError(appErr.ErrCode, appErr.ErrType, errors.New(fmt.Sprintf(`Error fetching transaction record for transaction id : %v, additional context : %s`, txID, appErr)))
	}
	return transaction, nil
}

func (service *TransactionService) PopulateChainData(transaction *dto.TransactionResponse, chainTxnId uuid.UUID) {

	//get and populate chain transaction if exists, if this call fails, log error but proceed on
	chainTransaction := model.ChainTransaction{}
	chainData := dto.ChainData{}
	if transaction.TransactionType == "ONCHAIN" && chainTxnId != uuid.Nil {
		if err := service.Repository.Get(&model.ChainTransaction{BaseModel: model.BaseModel{ID: chainTxnId}}, &chainTransaction); err != nil {
			logger.Error("Failed to populate chain record for transaction, id : %v and reference : %s", transaction.ID, transaction.TransactionReference)
			transaction.ChainData = nil
			return
		}
		chainTransaction.MaptoDto(&chainData)
		transaction.ChainData = &chainData
		return
	}
	return
}

func (service *TransactionService) VerifyBroadcastedTx(transaction model.Transaction) (bool, dto.TransactionStatusResponse, error) {

	// Get queued transaction for transactionId
	transactionQueue, err := service.GetQueuedTransactionByTxId(transaction.ID)
	if err != nil {
		logger.Error("verifyTransactionStatus logs : Error fetching queued transaction for transaction (%v) : %s", transaction.ID, err)
		return false, dto.TransactionStatusResponse{}, err
	}

	// Check if the transaction belongs to a batch and return batch
	broadcastTXRef := transactionQueue.DebitReference
	BatchService := NewBatchService(service.Cache, service.Config, service.Repository)
	CryptoAdapterService := NewCryptoAdapterService(service.Cache, service.Config, service.Repository)
	batchExist, _, err := BatchService.CheckBatchExistAndReturn(transactionQueue.BatchID)
	if err != nil {
		return false, dto.TransactionStatusResponse{}, err
	}
	if batchExist {
		broadcastTXRef = transactionQueue.BatchID.String()
	}
	// Get status of the TXN
	txnExist, broadcastedTX, err := CryptoAdapterService.GetBroadcastedTXNDetailsByRefAndSymbol(broadcastTXRef, transactionQueue.AssetSymbol)
	if err != nil {
		logger.Error("verifyTransactionStatus logs : Error checking the broadcasted state for queued transaction (%v) : %s", transactionQueue.ID, err)
		return false, dto.TransactionStatusResponse{}, err
	}
	return txnExist, broadcastedTX, err
}

func (service *TransactionService) GetTransactionStatus(txnExist bool, broadcastedTX dto.TransactionStatusResponse, transaction model.Transaction, transactionQueue model.TransactionQueue) (string, error) {

	tx := database.NewTx(service.Repository.Db())
	if !txnExist {
		if utility.IsExceedWaitTime(time.Now(), transactionQueue.CreatedAt.Add(time.Duration(constants.MIN_WAIT_TIME_IN_PROCESSING)*time.Second)) {
			// Revert the transaction status back to pending, as transaction has not been broadcasted
			if err := tx.Update(&transaction, &model.Transaction{TransactionStatus: model.TransactionStatus.PENDING}).UpdateWhere(&model.TransactionQueue{}, transactionQueue, &model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING}).Commit(); err != nil {
				logger.Error("GetTransactionStatus logs : Error occured while updating transaction %v to PENDING : %+v; %s", transaction.ID, service.Error, err)
				return "", err
			}
			return model.TransactionStatus.PENDING, nil
		}
		return "", nil
	}

	// Get the chain transaction for the broadcasted txn hash
	chainTransaction := model.ChainTransaction{}
	err := service.Repository.Get(&model.ChainTransaction{BaseModel: model.BaseModel{ID: transaction.OnChainTxId}}, &chainTransaction)
	if err != nil {
		logger.Error("GetTransactionStatus logs : Error fetching chain transaction for transaction (%+v) : %s", transactionQueue.ID, err)
		return "", err
	}
	blockHeight, err := strconv.Atoi(broadcastedTX.BlockHeight)

	// Update the transactions on the transaction table and on queue tied to the chain transaction as well as the batch status,if it is a batch transaction
	switch broadcastedTX.Status {
	case constants.SUCCESSFUL:
		chainTransactionUpdate := model.ChainTransaction{Status: true, TransactionFee: broadcastedTX.TransactionFee, BlockHeight: int64(blockHeight)}
		if err := tx.Update(&chainTransaction, chainTransactionUpdate).Update(&transaction, &model.Transaction{TransactionStatus: model.TransactionStatus.COMPLETED, OnChainTxId: chainTransaction.ID}).UpdateWhere(&model.TransactionQueue{}, transactionQueue, &model.TransactionQueue{TransactionStatus: model.TransactionStatus.COMPLETED}).Commit(); err != nil {
			logger.Error("GetTransactionStatus logs : Error occured while updating transaction %v to COMPLETED : %+v; %s", transaction.ID, service.Error, err)
			return "", err
		}
		return model.TransactionStatus.COMPLETED, err
	case constants.FAILED:
		if err := tx.Update(&transaction, &model.Transaction{TransactionStatus: model.TransactionStatus.TERMINATED, OnChainTxId: chainTransaction.ID}).Update(&transactionQueue, &model.TransactionQueue{TransactionStatus: model.TransactionStatus.TERMINATED}).Commit(); err != nil {
			logger.Error("GetTransactionStatus logs : Error occured while updating transaction %v to TERMINATED : %+v; %s", transaction.ID, service.Error, err)
			return "", err
		}
		return model.TransactionStatus.TERMINATED, err
	}

	return "", nil
}

func (service *TransactionService) Normalize(transaction model.Transaction) dto.TransactionResponse {
	tx := dto.TransactionResponse{}

	tx.ID = transaction.ID
	tx.InitiatorID = transaction.InitiatorID
	tx.RecipientID = transaction.RecipientID
	tx.Value = transaction.Value
	tx.TransactionStatus = transaction.TransactionStatus
	tx.TransactionReference = transaction.TransactionReference
	tx.PaymentReference = transaction.PaymentReference
	tx.PreviousBalance = transaction.PreviousBalance
	tx.AvailableBalance = transaction.AvailableBalance
	tx.TransactionType = transaction.TransactionType
	tx.TransactionEndDate = transaction.TransactionEndDate
	tx.TransactionStartDate = transaction.TransactionStartDate
	tx.CreatedDate = transaction.CreatedAt
	tx.UpdatedDate = transaction.UpdatedAt
	tx.TransactionTag = transaction.TransactionTag

	return tx
}

func (service *TransactionService) FetchPendingTransactionsInQueue() ([]model.TransactionQueue, error) {
	var transactionQueue []model.TransactionQueue
	if err := service.Repository.FetchByFieldName(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING}, &transactionQueue); err != nil {
		return transactionQueue, err
	}
	return transactionQueue, nil
}

func (service *TransactionService) ExternalTx(requestDetails dto.ExternalTransferRequest, initiatorID uuid.UUID) (string, error) {

	// A check is done to ensure the debitReference points to an actual previous debit
	debitTransaction, err := service.GetTransactionByRef(requestDetails.DebitReference)
	if err != nil {
		return "", err
	}
	// Check if withdrawal is ACTIVE on this asset
	DenominationServices := NewDenominationServices(service.Cache, service.Config, service.Repository)
	if err := DenominationServices.CheckWithdrawalIsActive(debitTransaction.AssetSymbol); err != nil {
		return "", err
	}
	// Checks to ensure the transaction status of debitReference is completed
	if debitTransaction.TransactionStatus != model.TransactionStatus.COMPLETED {
		return "", serviceError(http.StatusBadRequest, errorcode.INVALID_DEBIT_CODE, errors.New(errorcode.INVALID_DEBIT))
	}
	// Checks also that the value matches the value that was initially debited
	value := decimal.NewFromFloat(requestDetails.Value)
	debitValue, err := decimal.NewFromString(debitTransaction.Value)
	if err != nil {
		return "", serviceError(http.StatusInternalServerError, errorcode.SERVER_ERR_CODE, errors.New(errorcode.SERVER_ERR))
	}
	if value.GreaterThan(debitValue) {
		return "", serviceError(http.StatusBadRequest, errorcode.INVALID_DEBIT_AMOUNT, errors.New(errorcode.INVALID_DEBIT_AMOUNT))
	}

	// Get asset associated with the debit reference
	UserAssetService := NewUserAssetService(service.Cache, service.Config, service.Repository)
	debitAsset, err := UserAssetService.GetAssetBy(debitTransaction.RecipientID)
	if err != nil {
		return "", err
	}

	userAssetTXRequest := dto.UserAssetTXRequest{
		AssetID:              debitTransaction.RecipientID,
		TransactionReference: requestDetails.TransactionReference,
		Memo:                 debitTransaction.Memo,
		Value:                requestDetails.Value,
	}
	transaction := BuildTxnObject(debitAsset, userAssetTXRequest, debitTransaction.AvailableBalance, initiatorID)

	// Batch transaction, if asset is batchable
	isBatchable, err := DenominationServices.IsBatchable(debitAsset.AssetSymbol)
	if err != nil {
		return "", err
	}
	var activeBatchID uuid.UUID
	if isBatchable {
		BatchService := NewBatchService(service.Cache, service.Config, service.Repository)
		activeBatchID, err = BatchService.GetWaitingBatchId(debitTransaction.AssetSymbol)
		if err != nil {
			return "", err
		}
		transaction.BatchID = activeBatchID
		transaction.ProcessingType = model.ProcessingType.BATCH
	}
	transaction.TransactionType = model.TransactionType.ONCHAIN
	transaction.TransactionTag = model.TransactionTag.WITHDRAW
	transaction.DebitReference = requestDetails.DebitReference
	transaction.PreviousBalance = debitTransaction.PreviousBalance
	transaction.TransactionStatus = model.TransactionStatus.PENDING
	tx := database.NewTx(service.Repository.Db())
	tx = tx.Create(&transaction)

	// Queue transaction up for processing
	queue := model.TransactionQueue{
		Recipient:      requestDetails.RecipientAddress,
		Value:          utility.NativeValue(debitAsset.Decimal, value),
		DebitReference: requestDetails.DebitReference,
		AssetSymbol:    debitAsset.AssetSymbol,
		TransactionId:  transaction.ID,
		BatchID:        activeBatchID,
	}
	if !strings.EqualFold(debitTransaction.Memo, constants.NO_MEMO) {
		queue.Memo = debitTransaction.Memo
	}

	if err := tx.Create(&queue).Commit(); err != nil {
		appErr := err.(appError.Err)
		logger.Error(fmt.Sprintf("TransactionService logs : External transfer failed for asset %v. Error : %s", debitAsset.ID, err))
		return "", serviceError(appErr.ErrCode, appErr.ErrType, errors.New(fmt.Sprintf("External transfer failed for asset %v :  %s", debitAsset.ID, appErr)))
	}
	logger.Info(fmt.Sprintf("TransactionService logs : External transfer success for asset %v, ref:%s", debitAsset.ID, requestDetails.TransactionReference))
	return transaction.TransactionStatus, nil
}

func (service TransactionService) ConfirmTransaction(transactionHash string) error {

	// Get the chain transaction for the request hash
	chainTransaction := model.ChainTransaction{}
	err := service.Repository.Get(&model.ChainTransaction{TransactionHash: transactionHash}, &chainTransaction)
	if err != nil {
		return err
	}

	// Calls TransactionStatus to get the transaction status of the hash on-chain
	transactionStatusRequest := dto.TransactionStatusRequest{TransactionHash: transactionHash, AssetSymbol: chainTransaction.AssetSymbol}
	transactionStatusResponse := dto.TransactionStatusResponse{}
	CryptoAdapterService := NewCryptoAdapterService(service.Cache, service.Config, service.Repository)
	if err := CryptoAdapterService.TransactionStatus(transactionStatusRequest, &transactionStatusResponse); err != nil {
		return err
	}

	if err := service.confirmTransactionByStatus(chainTransaction, transactionStatusResponse); err != nil {
		return err
	}

	return nil
}

func (service *TransactionService) confirmTransactionByStatus(chainTransaction model.ChainTransaction, txStatusResponse dto.TransactionStatusResponse) error {

	// update the chain transaction with details of the on-chain TXN,
	blockHeight, err := strconv.Atoi(txStatusResponse.BlockHeight)
	if err != nil {
		return err
	}
	chainTransactionUpdate := model.ChainTransaction{Status: txStatusResponse.Status == constants.SUCCESSFUL, TransactionFee: txStatusResponse.TransactionFee, BlockHeight: int64(blockHeight)}
	if err := service.Repository.Update(&chainTransaction, chainTransactionUpdate); err != nil {
		return err
	}

	// Update the transactions on the transaction table and on queue tied to the chain transaction as well as the batch status,if it is a batch transaction
	switch txStatusResponse.Status {
	case constants.SUCCESSFUL:
		if err := service.UpdateTransactionStatusByChainID(chainTransaction, model.BatchStatus.COMPLETED); err != nil {
			return err
		}
	case constants.FAILED:
		if err := service.UpdateTransactionStatusByChainID(chainTransaction, model.BatchStatus.TERMINATED); err != nil {
			return err
		}
	default:
		break
	}
	return nil
}

func (service *TransactionService) UpdateTransactionStatusByChainID(chainTransaction model.ChainTransaction, status string) error {

	BatchService := NewBatchService(service.Cache, service.Config, service.Repository)
	batchExist, batch, err := BatchService.CheckBatchExistAndReturn(chainTransaction.BatchID)
	if err != nil {
		return err
	}

	tx := database.NewTx(service.Repository.Db())
	if batchExist {
		dateCompleted := time.Now()
		if err :=
			tx.Update(&batch, model.BatchRequest{Status: status, DateCompleted: &dateCompleted}).
				UpdateWhere(&model.Transaction{}, model.Transaction{BatchID: batch.ID}, model.Transaction{TransactionStatus: status}).
				UpdateWhere(&model.TransactionQueue{}, model.Transaction{BatchID: batch.ID}, model.Transaction{TransactionStatus: status}).
				Commit(); err != nil {
			return err
		}
	} else {
		transaction := model.Transaction{}
		if err := service.Repository.FetchByFieldName(&model.Transaction{OnChainTxId: chainTransaction.ID}, &transaction); err != nil {
			return err
		}
		if err := tx.
			Update(&transaction, model.Transaction{TransactionStatus: status}).
			UpdateWhere(&model.TransactionQueue{}, model.TransactionQueue{TransactionId: transaction.ID}, model.Transaction{TransactionStatus: status}).
			Commit(); err != nil {
			return err
		}

	}
	return nil
}

func (service *TransactionService) UpdateTransactionByTxID(transactionId uuid.UUID, status string, chainTransaction model.ChainTransaction) error {
	tx := database.NewTx(service.Repository.Db())
	if err := tx.
		Update(&model.Transaction{BaseModel: model.BaseModel{ID: transactionId}}, model.Transaction{TransactionStatus: status, OnChainTxId: chainTransaction.ID}).
		UpdateWhere(&model.TransactionQueue{}, model.TransactionQueue{TransactionId: transactionId}, model.Transaction{TransactionStatus: status}).
		Commit(); err != nil {
		return err
	}
	return nil
}

func BuildTxnObject(assetDetails model.UserAsset, requestDetails dto.UserAssetTXRequest, newAssetBalance string, initiatorId uuid.UUID) model.Transaction {
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

func (service *TransactionService) GetFloatAddressFor(symbol string) (string, error) {
	//Get the float address
	var floatAccount model.HotWalletAsset
	if err := service.Repository.Get(&model.HotWalletAsset{AssetSymbol: symbol}, &floatAccount); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id and trying to get float detials", err)
		return "", err
	}
	return floatAccount.Address, nil
}
