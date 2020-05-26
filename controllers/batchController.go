package controllers

import (
	"encoding/json"
	"fmt"
	"time"
	"errors"
	"net/http"
	"wallet-adapter/dto"
	"wallet-adapter/tasks"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"
	"wallet-adapter/database"
	"wallet-adapter/config"

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
	batchService := services.BatchService{BaseService : services.BaseService{Config: controller.Config, Cache : controller.Cache, Logger : controller.Logger}}
	serviceErr := dto.ServicesRequestErr{}
	done := make(chan bool)

	go func() {
		// Get all active batches
		activeBatches, err := batchService.GetAllActiveBatches(controller.Repository)
		if err != nil {
			controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v, while fetching active batches", err)
			done <- true
		}

		for _, batch := range activeBatches {
			
			var queuedBatchedTransactions []model.TransactionQueue

			// It calls the lock service to obtain a lock for the batch
			lockerServiceRequest := dto.LockerServiceRequest{
				Identifier:   fmt.Sprintf("%s%s", controller.Config.LockerPrefix, batch.ID),
				ExpiresAfter: 600000,
			}
			lockerServiceResponse := dto.LockerServiceResponse{}
			if err := services.AcquireLock(controller.Cache, controller.Logger, controller.Config, lockerServiceRequest, &lockerServiceResponse, &serviceErr); err != nil {
				controller.Logger.Error("Error occured while obtaining lock : %+v; %s", serviceErr, err)
				continue
			}
			processor := &BatchTransactionProcessor{Logger: controller.Logger, Cache: controller.Cache, Config: controller.Config, Repository: controller.Repository}

			// If batch is in RETRY_MODE
			if batch.Status == model.BatchStatus.RETRY_MODE {
				if err := processor.retryBatchProcessing(batch); err != nil {
					controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v. Batch with id %+v could not be reprocessed", err, batch.ID)
					continue
				}
			} else {
				if err := controller.Repository.Update(&batch, &model.BatchRequest{Status: model.BatchStatus.START_MODE, DateOfProcessing : time.Now()}); err != nil {
					controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating active batch status to PROCESSING", err)
					continue
				}

				if err := controller.Repository.FetchByFieldName(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING, BatchID : batch.ID, AssetSymbol: batch.AssetSymbol}, &queuedBatchedTransactions); err != nil {
					controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v, while fetching batched transactions from the queue", err)
					continue
				}

				if err := processor.processBatch(batch, queuedBatchedTransactions); err != nil {
					controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %s, while proccessing batch with id %+v", err, batch.ID)
					continue
				}
			}
			
			if err := controller.Repository.Update(&batch, &model.BatchRequest{Status: model.BatchStatus.PROCESSING, NoOfRecords: len(queuedBatchedTransactions)}); err != nil {
				controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating active batch status to PROCESSING", err)
				continue
			}

			// The routine returns the lock to the lock service and terminates
			lockReleaseRequest := dto.LockReleaseRequest{
				Identifier: fmt.Sprintf("%s%s", controller.Config.LockerPrefix, batch.ID),
				Token:      lockerServiceResponse.Token,
			}
			lockReleaseResponse := dto.ServicesRequestSuccess{}
			if err := services.ReleaseLock(controller.Cache, controller.Logger, controller.Config, lockReleaseRequest, &lockReleaseResponse, &serviceErr); err != nil || !lockReleaseResponse.Success {
				controller.Logger.Error("Error occured while releasing lock : %+v; %s", serviceErr, err)
			}

		}
		
		done <- true
	}()

	controller.Logger.Info("Outgoing response to ProcessBatchBTCTransactions request %+v", utility.SUCCESS)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess("SUCCESS", utility.SUCCESS))

	<-done
}

func (processor *BatchTransactionProcessor) processBatch(batch model.BatchRequest, queuedBatchedTransactions []model.TransactionQueue) error {

	// Prepare batched transactions for signing
	batchedRecipients := []dto.BatchRecipients{}
	batchedTransactionsIds := []uuid.UUID{}
	queuedBatchedTransactionsIds := []uuid.UUID{}
	floatAccount, err := services.GetHotWalletAddressFor(processor.Cache, processor.Repository.Db(), processor.Logger, processor.Config, batch.AssetSymbol)
	if err != nil {
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v", err)
		return err
	}

	for _, transaction := range queuedBatchedTransactions {
		recipient := dto.BatchRecipients{
			Address: transaction.Recipient,
			Value:   transaction.Value,
		}
		batchedRecipients = append(batchedRecipients, recipient)
		batchedTransactionsIds = append(batchedTransactionsIds, transaction.TransactionId)
		queuedBatchedTransactionsIds = append(queuedBatchedTransactionsIds, transaction.ID)
	}

	signTransactionRequest := dto.BatchBTCRequest{
		AssetSymbol:   batch.AssetSymbol,
		ChangeAddress: floatAccount,
		Origins:       []string{floatAccount},
		Recipients:    batchedRecipients,
	}

	// Calls key-management to sign batched transactions
	signTransactionResponse := dto.SignTransactionResponse{}
	serviceErr := dto.ServicesRequestErr{}
	if err := services.SignBatchBTCTransaction(nil, processor.Cache, processor.Logger, processor.Config, signTransactionRequest, &signTransactionResponse, serviceErr); err != nil {
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v", err)
		if serviceErr.Code == "INSUFFICIENT_BALANCE" {
			if err := processor.ProcessBatchTxnWithInsufficientFloat(batch.AssetSymbol); err != nil {
				processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while calling ProcessBatchTxnWithInsufficientFloat", err)
			}
		}
		return err
	}

	if err := processor.Repository.Update(&batch, &model.BatchRequest{Status: model.BatchStatus.RETRY_MODE, NoOfRecords: len(queuedBatchedTransactions), DateOfProcessing : time.Now()}); err != nil {
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating active batch status to PROCESSING", err)
		return err
	}

	// Send the signed data to crypto adapter to send to chain
	broadcastToChainRequest := dto.BroadcastToChainRequest{
		SignedData:  signTransactionResponse.SignedData,
		AssetSymbol: batch.AssetSymbol,
		Reference:   batch.ID.String(),
		ProcessType: utility.WITHDRAWALPROCESS,
	}
	broadcastToChainResponse := dto.BroadcastToChainResponse{}
	if err := services.BroadcastToChain(processor.Cache, processor.Logger, processor.Config, broadcastToChainRequest, &broadcastToChainResponse, serviceErr); err != nil {
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while broadcasting to chain", err)
		return err
	}

	// It creates a chain transaction for the batch with the transaction hash returned by crypto adapter
	chainTransaction := model.ChainTransaction{
		TransactionHash: broadcastToChainResponse.TransactionHash,
		BatchID: batch.ID,
	}
	if err := processor.Repository.Create(&chainTransaction); err != nil {
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while creating chain transaction", err)
		return err
	}

	if err := processor.UpdateBatchedTransactionsStatus(batch, chainTransaction, model.TransactionStatus.PROCESSING); err != nil {
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating batched transaction status for batch with id %+v", err, batch.ID)
		return err
	}

	return nil
}

func (processor *BatchTransactionProcessor) retryBatchProcessing(batch model.BatchRequest) error {

	chainTransaction := model.ChainTransaction{}
	if err := processor.Repository.GetByFieldName(&model.ChainTransaction{BatchID : batch.ID}, &chainTransaction); err != nil {
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while checking if chain transaction exist for batch with id %+v ", err, batch.ID)
		if err.Error() != utility.SQL_404 {
			return err
		}

		// Checks status of the TXN broadcast to chain
		txnExist, broadcastedTXNDetails, err := services.GetBroadcastedTXNDetails(batch.ID.String(), processor.Cache, processor.Logger, processor.Config)
		if err != nil {	
			processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while fetching broadcasted transaction status for batch with id %+v", err, batch.ID)
			return err
		}

		if !txnExist {
			// Fetches all PENDING transactions from the transaction queue table for the given BatchID
			var queuedBatchedTransactions []model.TransactionQueue
			if err := processor.Repository.FetchByFieldName(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING, BatchID : batch.ID, AssetSymbol: batch.AssetSymbol}, &queuedBatchedTransactions); err != nil {
				processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v, while fetching batched transactions from the queue", err)
				return err
			}
	
			if err := processor.processBatch(batch, queuedBatchedTransactions); err != nil {
				processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while processing batched transactions with batch id %+v", err, batch.ID)
				return err
			}
			
		}

		switch broadcastedTXNDetails.Status {
		case "FAILED":
			// Update batch transactions status
			if err := processor.UpdateBatchedTransactionsStatus(batch, chainTransaction, model.TransactionStatus.TERMINATED); err != nil {
				processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating batched transaction status for batch with id %+v", err, batch.ID)
				return err
			}
			if err := processor.Repository.Update(&batch, &model.BatchRequest{Status: model.BatchStatus.TERMINATED}); err != nil {
				processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating active batch status to TERMINATED", err)
				return err
			}
			return err
		case "SUCCESS":
			fallthrough
		default:
			// It creates a chain transaction for the batch with the transaction hash returned by crypto adapter if exist
			if broadcastedTXNDetails.TransactionHash != "" {
				chainTransaction := model.ChainTransaction{
					TransactionHash: broadcastedTXNDetails.TransactionHash,
					BatchID: batch.ID,
				}
				if err := processor.Repository.Create(&chainTransaction); err != nil {
					processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while creating chain transaction", err)
					return err
				}
				// Update batch transactions status
				if err := processor.UpdateBatchedTransactionsStatus(batch, chainTransaction, model.TransactionStatus.PROCESSING); err != nil {
					processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating batched transaction status for batch with id %+v", err, batch.ID)
					return err
				}
				return nil
			}
			if err := processor.Repository.Update(&batch, &model.BatchRequest{Status: model.BatchStatus.START_MODE}); err != nil {
				processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating active batch status to PROCESSING", err)
				return err
			}
			return err
		}

	}

	// Update batch transactions status
	if err := processor.UpdateBatchedTransactionsStatus(batch, chainTransaction, model.TransactionStatus.PROCESSING); err != nil {
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating batched transaction status for batch with id %+v", err, batch.ID)
		return err
	}
	return nil
}

func (processor *BatchTransactionProcessor) UpdateBatchedTransactionsStatus(batch model.BatchRequest, chainTransaction model.ChainTransaction, status string) error  {
	
	// Fetches all transactions for the given BatchID
	var queuedBatchedTransactions []model.TransactionQueue
	if err := processor.Repository.FetchByFieldName(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING, BatchID : batch.ID, AssetSymbol: batch.AssetSymbol}, &queuedBatchedTransactions); err != nil {
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v, while fetching batched transactions from the queue", err)
		return err
	}

	batchedTransactionsIds := []uuid.UUID{}
	queuedBatchedTransactionsIds := []uuid.UUID{}

	for _, transaction := range queuedBatchedTransactions {
		batchedTransactionsIds = append(batchedTransactionsIds, transaction.TransactionId)
		queuedBatchedTransactionsIds = append(queuedBatchedTransactionsIds, transaction.ID)
	}

	tx := processor.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while creating db transaction", err)
		return err
	}

	if err := tx.Model(model.Transaction{}).Where(batchedTransactionsIds).Updates(model.Transaction{TransactionStatus: status, OnChainTxId: chainTransaction.ID}).Error; err != nil {
		tx.Rollback()
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating the batchedTransactions status", err)
		return err
	}

	if err := tx.Model(model.TransactionQueue{}).Where(queuedBatchedTransactionsIds).Updates(model.TransactionQueue{TransactionStatus: status}).Error; err != nil {
		tx.Rollback()
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating the queued batchedTransactions status", err)
		return err
	}

	if err := tx.Commit().Error; err != nil {
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while commiting db transaction", err)
		return err
	}
	return nil
}

func (processor *BatchTransactionProcessor) ProcessBatchTxnWithInsufficientFloat(assetSymbol string) error {

	DB := database.Database{Logger: processor.Logger, Config: processor.Config, DB: processor.Repository.Db()}
	baseRepository := database.BaseRepository{Database: DB}

	if !processor.SweepTriggered {
		go tasks.SweepTransactions(processor.Cache, processor.Logger, processor.Config, baseRepository)
		processor.SweepTriggered = true
		return errors.New(fmt.Sprintf("Not enough balance in float for this transaction, triggering sweep operation."))
	}

	return errors.New(fmt.Sprintf("Not enough balance in float for this transaction, sweep operation in progress."))
}
