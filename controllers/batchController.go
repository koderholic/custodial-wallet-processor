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
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/tasks"
	"wallet-adapter/utility"

	uuid "github.com/satori/go.uuid"
)

type BatchTransactionProcessor struct {
	Cache          *utility.MemoryCache
	Logger         *utility.Logger
	Config         config.Data
	Repository     database.IBatchRepository
	SweepTriggered bool
}

// ProcessBatchBTCTransactions ...
func (controller BatchController) ProcessBatchBTCTransactions(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	batchService := services.BatchService{BaseService: services.BaseService{Config: controller.Config, Cache: controller.Cache, Logger: controller.Logger}}
	done := make(chan bool)

	go func() {
		// Get all active batches
		activeBatches, err := batchService.GetAllActiveBatches(controller.Repository)
		if err != nil {
			controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v, while fetching active batches", err)
			done <- true
		}

		for _, batch := range activeBatches {

			// It calls the lock service to obtain a lock for the batch
			lockerServiceToken, err := controller.obtainLock(batch.ID.String())
			if err != nil {
				continue
			}
			processor := &BatchTransactionProcessor{Logger: controller.Logger, Cache: controller.Cache, Config: controller.Config, Repository: controller.Repository}

			// If batch is in RETRY_MODE
			if batch.Status == model.BatchStatus.RETRY_MODE {
				if err := processor.retryBatchProcessing(batch); err != nil {
					controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v. Batch with id %+v could not be reprocessed", err, batch.ID)
					_ = controller.releaseLock(batch.ID.String(), lockerServiceToken)
					continue
				}
			} else {
				if err := processor.UpdateBatchedTransactionsStatus(batch, model.ChainTransaction{}, model.BatchStatus.START_MODE); err != nil {
					controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating active batch status to PROCESSING", err)
					_ = controller.releaseLock(batch.ID.String(), lockerServiceToken)
					continue
				}

				queuedBatchedTransactions := []model.TransactionQueue{}
				if err := controller.Repository.FetchByFieldName(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING, BatchID: batch.ID,
					AssetSymbol: batch.AssetSymbol}, &queuedBatchedTransactions); err != nil {
					controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v, while fetching batched transactions from the queue", err)
					_ = controller.releaseLock(batch.ID.String(), lockerServiceToken)
					continue
				}

				if err := processor.processBatch(batch, queuedBatchedTransactions); err != nil {
					controller.Logger.Error("Error response from ProcessBatchBTCTransactions : could not process batch with id : %v, error : %s",batch.ID.String(), err)
					_ = controller.releaseLock(batch.ID.String(), lockerServiceToken)
					continue
				}
			}

			// The routine returns the lock to the lock service and terminates
			_ = controller.releaseLock(batch.ID.String(), lockerServiceToken)

		}

		done <- true
	}()

	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess(utility.SUCCESSFUL, utility.SUCCESS))

	<-done
}

func (processor *BatchTransactionProcessor) processBatch(batch model.BatchRequest, queuedBatchedTransactions []model.TransactionQueue) error {

	// Prepare batched transactions for signing
	batchedRecipients := []dto.BatchRecipients{}
	batchedTransactionsIds := []uuid.UUID{}
	queuedBatchedTransactionsIds := []uuid.UUID{}
	floatAccount, err := services.GetHotWalletAddressFor(processor.Cache, processor.Repository.Db(), processor.Logger, processor.Config, batch.AssetSymbol)
	if err != nil {
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

	sendBatchTransactionRequest := dto.BatchRequest{
		AssetSymbol:   batch.AssetSymbol,
		ChangeAddress: floatAccount,
		Origins:       []string{floatAccount},
		Recipients:    batchedRecipients,
		ProcessType:   utility.WITHDRAWALPROCESS,
		Reference:     batch.ID.String(),
	}

	sendBatchTransactionResponse := dto.SendTransactionResponse{}
	serviceErr := dto.ServicesRequestErr{}
	if err := services.SendBatchTransaction(nil, processor.Cache, processor.Logger, processor.Config, sendBatchTransactionRequest, &sendBatchTransactionResponse, &serviceErr); err != nil {
		if serviceErr.StatusCode == http.StatusBadRequest {
			if serviceErr.Code == errorcode.INSUFFICIENT_FUNDS {
				total := int64(0)
				for _, value := range sendBatchTransactionRequest.Recipients {
					total += value.Value
				}
				if err := processor.ProcessBatchTxnWithInsufficientFloat(batch.AssetSymbol, *big.NewInt(total)); err != nil {
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
		TransactionHash: sendBatchTransactionResponse.TransactionHash,
		BatchID:         batch.ID,
		AssetSymbol:     batch.AssetSymbol,
	}
	if err := processor.Repository.Create(&chainTransaction); err != nil {
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
	txnExist, broadcastedTXNDetails, err := services.GetBroadcastedTXNDetailsByRef(batch.ID.String(), batch.AssetSymbol, processor.Cache, processor.Logger, processor.Config)
	if err != nil {
		return err
	}

	if !txnExist {
		// Fetches all PENDING transactions from the transaction queue table for the given BatchID
		var queuedBatchedTransactions []model.TransactionQueue
		if err := processor.Repository.FetchByFieldName(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING, BatchID: batch.ID,
			AssetSymbol: batch.AssetSymbol}, &queuedBatchedTransactions); err != nil {
			return err
		}

		if err := processor.processBatch(batch, queuedBatchedTransactions); err != nil {
			return err
		}

		return nil
	}

	chainTransaction := model.ChainTransaction{
		TransactionHash: broadcastedTXNDetails.TransactionHash,
		BatchID:         batch.ID,
		AssetSymbol:     batch.AssetSymbol,
	}
	switch broadcastedTXNDetails.Status {
	case utility.FAILED:
		if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{BatchID: batch.ID}, &chainTransaction,
		model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}); err != nil {
			return err
		}
		// Update batch transactions status
		if err := processor.UpdateBatchedTransactionsStatus(batch, chainTransaction, model.BatchStatus.TERMINATED); err != nil {
			return err
		}
		return nil
	case utility.SUCCESSFUL:
		chainTransaction.Status = true
		if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{BatchID: batch.ID}, &chainTransaction,
		model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash, Status: true}); err != nil {
			return err
		}
		// Update batch transactions status
		if err := processor.UpdateBatchedTransactionsStatus(batch, chainTransaction, model.BatchStatus.COMPLETED); err != nil {
			return err
		}
		return nil
	default:
		// It creates a chain transaction for the batch with the transaction hash returned by crypto adapter if exist
		if broadcastedTXNDetails.TransactionHash != "" {
			if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{BatchID: batch.ID}, &chainTransaction,
			model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}); err != nil {
				return err
			}
			// Update batch transactions status
			if err := processor.UpdateBatchedTransactionsStatus(batch, chainTransaction, model.BatchStatus.PROCESSING); err != nil {
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
		return err
	}

	tx := processor.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
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

		if err := tx.Model(&model.Transaction{}).Where("id IN (?)", batchedTransactionsIds).
			Updates(model.Transaction{TransactionStatus: status, OnChainTxId: chainTransaction.ID}).Error; err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Model(&model.TransactionQueue{}).Where("id IN (?)", queuedBatchedTransactionsIds).
			Updates(model.TransactionQueue{TransactionStatus: status}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	dateCompleted := time.Now()
	if err := tx.Model(&batch).Updates(model.BatchRequest{Status: status, NoOfRecords: len(queuedBatchedTransactions), DateCompleted: &dateCompleted}).Error; err != nil {
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil

}

func (processor *BatchTransactionProcessor) ProcessBatchTxnWithInsufficientFloat(assetSymbol string, amount big.Int) error {

	DB := database.Database{Logger: processor.Logger, Config: processor.Config, DB: processor.Repository.Db()}
	baseRepository := database.BaseRepository{Database: DB}

	serviceErr := dto.ServicesRequestErr{}
	tasks.NotifyColdWalletUsersViaSMS(amount, assetSymbol, processor.Config, processor.Cache, processor.Logger, serviceErr, baseRepository)
	if !processor.SweepTriggered {
		go tasks.SweepTransactions(processor.Cache, processor.Logger, processor.Config, baseRepository)
		processor.SweepTriggered = true
		return errors.New(fmt.Sprintf("Not enough balance in float for this transaction, triggering sweep operation."))
	}
	return errors.New(fmt.Sprintf("Not enough balance in float for this transaction, sweep operation in progress."))
}

func (controller BatchController) obtainLock(identifier string) (string, error) {
	serviceErr := dto.ServicesRequestErr{}

	lockerServiceRequest := dto.LockerServiceRequest{
		Identifier:   fmt.Sprintf("%s%s", controller.Config.LockerPrefix, identifier),
		ExpiresAfter: 600000,
	}
	lockerServiceResponse := dto.LockerServiceResponse{}
	if err := services.AcquireLock(controller.Cache, controller.Logger, controller.Config, lockerServiceRequest, &lockerServiceResponse, &serviceErr); err != nil {
		return "", err
	}
	return lockerServiceResponse.Token, nil
}

func (controller BatchController) releaseLock(identifier string, lockerserviceToken string) error {
	serviceErr := dto.ServicesRequestErr{}

	lockReleaseRequest := dto.LockReleaseRequest{
		Identifier: fmt.Sprintf("%s%s", controller.Config.LockerPrefix, identifier),
		Token:      lockerserviceToken,
	}
	lockReleaseResponse := dto.ServicesRequestSuccess{}
	if err := services.ReleaseLock(controller.Cache, controller.Logger,
		controller.Config, lockReleaseRequest, &lockReleaseResponse, &serviceErr); err != nil || !lockReleaseResponse.Success {
		return err
	}
	return nil
}
