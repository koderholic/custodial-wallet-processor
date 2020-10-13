package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"time"
	"wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/tasks"
	"wallet-adapter/tasks/sweep"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/constants"
	"wallet-adapter/utility/errorcode"
	"wallet-adapter/utility/logger"
	Response "wallet-adapter/utility/response"

	uuid "github.com/satori/go.uuid"
)

type BatchTransactionProcessor struct {
	Cache          *cache.Memory
	Config         config.Data
	Repository     database.IBatchRepository
	SweepTriggered bool
}

// ProcessBatchBTCTransactions ...
func (controller BatchController) ProcessBatchBTCTransactions(responseWriter http.ResponseWriter, requestReader *http.Request) {
	BatchService := services.NewBatchService(controller.Cache, controller.Config, controller.Repository)
	done := make(chan bool)

	go func() {
		// Get all active batches
		activeBatches, err := BatchService.GetAllActiveBatches()
		if err != nil {
			logger.Error("Error response from ProcessBatchBTCTransactions : %+v, while fetching active batches", err)
			done <- true
		}

		for _, batch := range activeBatches {
			// It calls the lock service to obtain a lock for the batch
			LockerService := services.NewLockerService(controller.Cache, controller.Config, controller.Repository)
			lockerServiceResponse, err := LockerService.AcquireLock(batch.ID.String(), constants.SIX_HUNDRED_MILLISECONDS)
			lockerServiceToken := lockerServiceResponse.Token
			if err != nil {
				continue
			}
			processor := &BatchTransactionProcessor{Cache: controller.Cache, Config: controller.Config, Repository: controller.Repository}
			// If batch is in RETRY_MODE
			if batch.Status == model.BatchStatus.RETRY_MODE {
				if err := processor.retryBatchProcessing(batch); err != nil {
					logger.Error("Error response from ProcessBatchBTCTransactions : %+v. Batch with id %+v could not be reprocessed", err, batch.ID)
					_ = LockerService.ReleaseLock(batch.ID.String(), lockerServiceToken)
					continue
				}
			} else {
				if err := processor.UpdateBatchedTransactionsStatus(batch, model.ChainTransaction{}, model.BatchStatus.START_MODE); err != nil {
					logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating active batch status to PROCESSING", err)
					_ = LockerService.ReleaseLock(batch.ID.String(), lockerServiceToken)
					continue
				}

				queuedBatchedTransactions := []model.TransactionQueue{}
				if err := controller.Repository.FetchByFieldName(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING, BatchID: batch.ID, AssetSymbol: batch.AssetSymbol}, &queuedBatchedTransactions); err != nil {
					logger.Error("Error response from ProcessBatchBTCTransactions : %+v, while fetching batched transactions from the queue", err)
					_ = LockerService.ReleaseLock(batch.ID.String(), lockerServiceToken)
					continue
				}

				if err := processor.processBatch(batch, queuedBatchedTransactions); err != nil {
					_ = LockerService.ReleaseLock(batch.ID.String(), lockerServiceToken)
					continue
				}
			}
			// The routine returns the lock to the lock service and terminates
			_ = LockerService.ReleaseLock(batch.ID.String(), lockerServiceToken)
		}

		done <- true
	}()

	logger.Info("Outgoing response to ProcessBatchBTCTransactions request %+v", constants.SUCCESS)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(Response.New().PlainSuccess(constants.SUCCESSFUL, constants.SUCCESS))

	<-done
}

func (processor *BatchTransactionProcessor) processBatch(batch model.BatchRequest, queuedBatchedTransactions []model.TransactionQueue) error {

	// Prepare batched transactions for signing
	batchedRecipients := []dto.BatchRecipients{}
	batchedTransactionsIds := []uuid.UUID{}
	queuedBatchedTransactionsIds := []uuid.UUID{}
	HotWalletService := services.NewHotWalletService(processor.Cache, processor.Config, processor.Repository)
	floatAccount, err := HotWalletService.GetHotWalletAddressFor(processor.Repository.Db(), batch.AssetSymbol)
	if err != nil {
		logger.Error("Error response from ProcessBatchBTCTransactions : %+v", err)
		return err
	}

	for _, transaction := range queuedBatchedTransactions {
		recipient := dto.BatchRecipients{
			Address: transaction.Recipient,
			Value:   transaction.Value.BigInt().Int64(),
		}
		batchedRecipients = append(batchedRecipients, recipient)
		batchedTransactionsIds = append(batchedTransactionsIds, transaction.TransactionId)
		queuedBatchedTransactionsIds = append(queuedBatchedTransactionsIds, transaction.ID)
	}

	signTransactionRequest := dto.BatchRequest{
		AssetSymbol:   batch.AssetSymbol,
		ChangeAddress: floatAccount,
		Origins:       []string{floatAccount},
		Recipients:    batchedRecipients,
		ProcessType:   constants.WITHDRAWALPROCESS,
		Reference:     batch.ID.String(),
	}

	// Calls key-management to sign batched transactions
	SignBatchTransactionAndBroadcastResponse := dto.SignAndBroadcastResponse{}
	KeyManagementService := services.NewKeyManagementService(processor.Cache, processor.Config, processor.Repository)
	if err := KeyManagementService.SignBatchTransactionAndBroadcast(nil, signTransactionRequest, &SignBatchTransactionAndBroadcastResponse); err != nil {
		logger.Error("Error response from ProcessBatchBTCTransactions : %+v ", err)
		if err.(appError.Err).ErrCode == http.StatusBadRequest {
			if err.(appError.Err).ErrType == errorcode.INSUFFICIENT_FUNDS {
				total := int64(0)
				for _, value := range signTransactionRequest.Recipients {
					total += value.Value
				}
				if err := processor.ProcessBatchTxnWithInsufficientFloat(batch.AssetSymbol, *big.NewInt(total)); err != nil {
					logger.Error("Error response from ProcessBatchBTCTransactions : %+v while calling ProcessBatchTxnWithInsufficientFloat", err)
				}
				return err
			}
			if err := processor.UpdateBatchedTransactionsStatus(batch, model.ChainTransaction{}, model.BatchStatus.TERMINATED); err != nil {
				return err
			}
			return err
		}
		if err := processor.retryBatchProcessing(batch); err != nil {
			return err
		}
		return err
	}

	// It creates a chain transaction for the batch with the transaction hash returned by crypto adapter
	chainTransaction := model.ChainTransaction{
		TransactionHash: SignBatchTransactionAndBroadcastResponse.TransactionHash,
		BatchID:         batch.ID,
	}
	if err := processor.Repository.Create(&chainTransaction); err != nil {
		logger.Error("Error response from ProcessBatchBTCTransactions : %+v while creating chain transaction", err)
		if err := processor.UpdateBatchedTransactionsStatus(batch, chainTransaction, model.BatchStatus.RETRY_MODE); err != nil {
			return err
		}
		return err
	}

	if err := processor.UpdateBatchedTransactionsStatus(batch, chainTransaction, model.BatchStatus.PROCESSING); err != nil {
		return err
	}

	return nil
}

func (processor *BatchTransactionProcessor) retryBatchProcessing(batch model.BatchRequest) error {
	// Checks status of the TXN broadcast to chain
	CryptoAdapterService := services.NewCryptoAdapterService(processor.Cache, processor.Config, processor.Repository)
	txnExist, broadcastedTXNDetails, err := CryptoAdapterService.GetBroadcastedTXNDetailsByRefAndSymbol(batch.ID.String(), batch.AssetSymbol)
	if err != nil {
		logger.Error("Error response from retryBatchProcessing : %+v while fetching broadcasted transaction status for batch with id %+v", err, batch.ID)
		return err
	}

	if !txnExist {
		// Fetches all PENDING transactions from the transaction queue table for the given BatchID
		var queuedBatchedTransactions []model.TransactionQueue
		if err := processor.Repository.FetchByFieldName(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING, BatchID: batch.ID, AssetSymbol: batch.AssetSymbol}, &queuedBatchedTransactions); err != nil {
			logger.Error("Error response from retryBatchProcessing : %+v, while fetching batched transactions from the queue", err)
			return err
		}

		if err := processor.processBatch(batch, queuedBatchedTransactions); err != nil {
			logger.Error("Error response from retryBatchProcessing : %+v while processing batched transactions with batch id %+v", err, batch.ID)
			return err
		}

		return nil
	}

	chainTransaction := model.ChainTransaction{
		TransactionHash: broadcastedTXNDetails.TransactionHash,
		BatchID:         batch.ID,
	}
	switch broadcastedTXNDetails.Status {
	case constants.FAILED:
		if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{BatchID: batch.ID}, &chainTransaction, model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}); err != nil {
			logger.Error("Error response from ProcessBatchBTCTransactions : %+v while creating chain transaction", err)
			return err
		}
		// Update batch transactions status
		if err := processor.UpdateBatchedTransactionsStatus(batch, chainTransaction, model.BatchStatus.TERMINATED); err != nil {
			logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating batched transaction status for batch with id %+v", err, batch.ID)
			return err
		}
		return nil
	case constants.SUCCESSFUL:
		chainTransaction.Status = true
		if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{BatchID: batch.ID}, &chainTransaction, model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash, Status: true}); err != nil {
			logger.Error("Error response from ProcessBatchBTCTransactions : %+v while creating chain transaction", err)
			return err
		}
		// Update batch transactions status
		if err := processor.UpdateBatchedTransactionsStatus(batch, chainTransaction, model.BatchStatus.COMPLETED); err != nil {
			logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating batched transaction status for batch with id %+v", err, batch.ID)
			return err
		}
		return nil
	default:
		// It creates a chain transaction for the batch with the transaction hash returned by crypto adapter if exist
		if broadcastedTXNDetails.TransactionHash != "" {
			if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{BatchID: batch.ID}, &chainTransaction, model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}); err != nil {
				logger.Error("Error response from ProcessBatchBTCTransactions : %+v while creating chain transaction", err)
				return err
			}
			// Update batch transactions status
			if err := processor.UpdateBatchedTransactionsStatus(batch, chainTransaction, model.BatchStatus.PROCESSING); err != nil {
				logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating batched transaction status for batch with id %+v", err, batch.ID)
				return err
			}
			return nil
		}
	}

	return nil
}

func (processor *BatchTransactionProcessor) UpdateBatchedTransactionsStatus(batch model.BatchRequest, chainTransaction model.ChainTransaction, status string) error {

	// Fetches all transactions for the given BatchID
	var queuedBatchedTransactions []model.TransactionQueue
	if err := processor.Repository.FetchByFieldName(&model.TransactionQueue{BatchID: batch.ID, AssetSymbol: batch.AssetSymbol}, &queuedBatchedTransactions); err != nil {
		logger.Error("Error response from UpdateBatchedTransactionsStatus : %+v, while fetching batched transactions from the queue", err)
		return err
	}

	tx := processor.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		logger.Error("Error response from UpdateBatchedTransactionsStatus : %+v while creating db transaction", err)
		return err
	}
	if status == model.BatchStatus.PROCESSING || status == model.BatchStatus.COMPLETED || status == model.BatchStatus.TERMINATED {
		// Updates all transactions associated with the batch
		batchedTransactionsIds := []uuid.UUID{}
		queuedBatchedTransactionsIds := []uuid.UUID{}

		for _, transaction := range queuedBatchedTransactions {
			batchedTransactionsIds = append(batchedTransactionsIds, transaction.TransactionId)
			queuedBatchedTransactionsIds = append(queuedBatchedTransactionsIds, transaction.ID)
		}

		if err := tx.Model(&model.Transaction{}).Where("id IN (?)", batchedTransactionsIds).Updates(model.Transaction{TransactionStatus: status, OnChainTxId: chainTransaction.ID}).Error; err != nil {
			tx.Rollback()
			logger.Error("Error response from UpdateBatchedTransactionsStatus : %+v while updating the batchedTransactions status to %s", err, status)
			return err
		}

		if err := tx.Model(&model.TransactionQueue{}).Where("id IN (?)", queuedBatchedTransactionsIds).Updates(model.TransactionQueue{TransactionStatus: status}).Error; err != nil {
			tx.Rollback()
			logger.Error("Error response from UpdateBatchedTransactionsStatus : %+v while updating the queued batchedTransactions status to %s", err, status)
			return err
		}
	}

	dateCompleted := time.Now()
	if err := tx.Model(&batch).Updates(model.BatchRequest{Status: status, NoOfRecords: len(queuedBatchedTransactions), DateCompleted: &dateCompleted}).Error; err != nil {
		logger.Error("Error response from UpdateBatchedTransactionsStatus : %+v while updating active batch status to %s", err, status)
		return err
	}

	if err := tx.Commit().Error; err != nil {
		logger.Error("Error response from UpdateBatchedTransactionsStatus : %+v while commiting db transaction", err)
		return err
	}

	return nil

}

func (processor *BatchTransactionProcessor) ProcessBatchTxnWithInsufficientFloat(assetSymbol string, amount big.Int) error {

	DB := database.Database{Config: processor.Config, DB: processor.Repository.Db()}
	baseRepository := database.BaseRepository{Database: DB}
	tasks.NotifyColdWalletUsersViaSMS(amount, assetSymbol, processor.Config, processor.Cache, processor.Repository)
	if !processor.SweepTriggered {
		go sweep.SweepTransactions(processor.Cache, processor.Config, &baseRepository)
		processor.SweepTriggered = true
		return errors.New(fmt.Sprintf("Not enough balance in float for this transaction, triggering sweep operation."))
	}
	return errors.New(fmt.Sprintf("Not enough balance in float for this transaction, sweep operation in progress."))
}
