package services

import (
	"errors"
	"fmt"
	"strconv"
	"time"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/utility"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/constants"
	"wallet-adapter/utility/logger"

	uuid "github.com/satori/go.uuid"
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
		status, _ := service.VerifyTransactionStatus(transaction)
		if status != "" {
			transaction.TransactionStatus = status
		}
	}
	return transaction, nil
}

func (service *TransactionService) GetTransactionByRef(reference string) (model.Transaction, error) {

	transaction := model.Transaction{}
	if err := service.Repository.GetByFieldName(&model.Transaction{TransactionReference: reference}, &transaction); err != nil {
		appErr := err.(appError.Err)
		return model.Transaction{}, serviceError(appErr.ErrCode, appErr.ErrType, errors.New(fmt.Sprintf(`Error fetching transaction record for reference : %s, additional context : %s`, reference, appErr)))
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

func normalize() {

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

func (service *TransactionService) VerifyTransactionStatus(transaction model.Transaction) (string, error) {

	// Get queued transaction for transactionId
	transactionQueue, err := service.GetQueuedTransactionById(transaction.ID)
	if err != nil {
		logger.Error("verifyTransactionStatus logs : Error fetching queued transaction for transaction (%v) : %s", transaction.ID, err)
		return "", err
	}

	// Check if the transaction belongs to a batch and return batch
	broadcastTXRef := transactionQueue.DebitReference
	BatchService := NewBatchService(service.Cache, service.Config, service.Repository)
	CryptoAdapterService := NewCryptoAdapterService(service.Cache, service.Config, service.Repository)
	batchExist, _, err := BatchService.CheckBatchExistAndReturn(transactionQueue.BatchID)
	if err != nil {
		return "", err
	}
	if batchExist {
		broadcastTXRef = transactionQueue.BatchID.String()
	}

	// Get status of the TXN
	txnExist, broadcastedTX, err := CryptoAdapterService.GetBroadcastedTXNDetailsByRefAndSymbol(broadcastTXRef, transactionQueue.AssetSymbol)
	if err != nil {
		logger.Error("verifyTransactionStatus logs : Error checking the broadcasted state for queued transaction (%v) : %s", transactionQueue.ID, err)
		return "", err
	}

	status, err := service.GetTransactionStatus(txnExist, broadcastedTX, transaction, transactionQueue)
	if err != nil {
		logger.Error("verifyTransactionStatus logs : Error checking the broadcasted state for queued transaction (%v) : %s", transactionQueue.ID, err)
		return "", err
	}

	return status, nil
}

func (service *TransactionService) GetTransactionStatus(txnExist bool, broadcastedTX dto.TransactionStatusResponse, transaction model.Transaction, transactionQueue model.TransactionQueue) (string, error) {

	tx := database.NewTx(service.Repository.Db())
	if !txnExist {
		if utility.IsExceedWaitTime(time.Since(transactionQueue.CreatedAt), time.Duration(constants.MIN_WAIT_TIME_IN_PROCESSING)) {
			// Revert the transaction status back to pending, as transaction has not been broadcasted
			if err := tx.Update(&transaction, &model.Transaction{TransactionStatus: model.TransactionStatus.PENDING}).Update(&transactionQueue, &model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING}).Commit(); err != nil {
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
		if err := tx.Update(&chainTransaction, chainTransactionUpdate).Update(&transaction, &model.Transaction{TransactionStatus: model.TransactionStatus.COMPLETED, OnChainTxId: chainTransaction.ID}).Update(&transactionQueue, &model.TransactionQueue{TransactionStatus: model.TransactionStatus.COMPLETED}).Commit(); err != nil {
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
