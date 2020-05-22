package controllers

import (
	"encoding/json"
	"fmt"
	"time"
	"net/http"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"

	uuid "github.com/satori/go.uuid"
)


// ProcessTransaction ...
func (controller UserAssetController) ProcessBatchBTCTransactions(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	serviceErr := dto.ServicesRequestErr{}
	done := make(chan bool)

	go func() {
		// Get all active batches
		activeBatches, err := services.GetAllActiveBatches(controller.Repository, controller.Logger, controller.Config)
		if err != nil {
			controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v, while fetching active batches", err)
			done <- true
		}

		for _, batch := range activeBatches {
			
			// It calls the lock service to obtain a lock for the batch transactions
			lockerServiceRequest := dto.LockerServiceRequest{
				Identifier:   fmt.Sprintf("%s%s", controller.Config.LockerPrefix, batch.ID),
				ExpiresAfter: 600000,
			}
			lockerServiceResponse := dto.LockerServiceResponse{}
			if err := services.AcquireLock(controller.Cache, controller.Logger, controller.Config, lockerServiceRequest, &lockerServiceResponse, &serviceErr); err != nil {
				controller.Logger.Error("Error occured while obtaining lock : %+v; %s", serviceErr, err)
				done <- true
			}
			processor := &TransactionProccessor{Logger: controller.Logger, Cache: controller.Cache, Config: controller.Config, Repository: controller.Repository}
			
			// If batch is in RETRY_MODE
			if batch.Status == model.BatchStatus.RETRY_MODE {
				if err := processor.retryBatchProcessing(batch); err != nil {
					controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v. Batch with id %+v could not be reprocessed", err, batch.ID)
					done <- true
				}
			}

			// If batch is in WAIT_MODE
			shouldProcess := services.CanProcess(batch, controller.Config)
			if !shouldProcess {
				controller.Logger.Error("Error response from ProcessBatchBTCTransactions : Batch with id %+v cannont be processed now, still open for transactions", batch.ID)
				done <- true
			}

			var queuedBatchedTransactions []model.TransactionQueue
			if err := controller.Repository.FetchByFieldName(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING, BatchID : batch.ID, AssetSymbol: utility.BTC}, &queuedBatchedTransactions); err != nil {
				controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v, while fetching batched transactions from the queue", err)
				done <- true
			}
			if err := processor.processBatch(batch, queuedBatchedTransactions); err != nil {
				controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %s, while proccessing batch with id %+v", err, batch.ID)
				done <- true
			}

			batchDetails := model.BatchRequest{}
			if err := controller.Repository.GetByFieldName(&model.BatchRequest{BaseModel :model.BaseModel{ID : batch.ID}}, &batchDetails); err != nil {
				controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while getting active batch details", err)
				done <- true
			}
			if err := controller.Repository.Update(&batchDetails, &model.BatchRequest{Status: model.BatchStatus.PROCESSING, NoOfRecords: len(queuedBatchedTransactions), DateOfprocessing : time.Now()}); err != nil {
				controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating active batch status to PROCESSING", err)
				done <- true
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

func (processor *TransactionProccessor) processBatch(batch model.BatchRequest, queuedBatchedTransactions []model.TransactionQueue) error {

	// Prepare batched transactions for signing
	batchedRecipients := []dto.BatchRecipients{}
	batchedTransactionsIds := []uuid.UUID{}
	queuedBatchedTransactionsIds := []uuid.UUID{}
	floatAccount, err := services.GetHotWalletAddressFor(processor.Cache, processor.Repository.Db(), processor.Logger, processor.Config, utility.BTC)
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
		AssetSymbol:   utility.BTC,
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
			if err := processor.ProcessTxnWithInsufficientFloat(utility.BTC); err != nil {
				processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while calling ProcessTxnWithInsufficientFloat", err)
			}
		}
		return err
	}

	batchDetails := model.BatchRequest{}
	if err := processor.Repository.GetByFieldName(&model.BatchRequest{BaseModel :model.BaseModel{ID : batch.ID}}, &batchDetails); err != nil {
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while getting active batch details", err)
		return err
	}
	if err := processor.Repository.Update(&batchDetails, &model.BatchRequest{Status: model.BatchStatus.RETRY_MODE, NoOfRecords: len(queuedBatchedTransactions), DateOfprocessing : time.Now()}); err != nil {
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating active batch status to PROCESSING", err)
		return err
	}

	// Send the signed data to crypto adapter to send to chain
	broadcastToChainRequest := dto.BroadcastToChainRequest{
		SignedData:  signTransactionResponse.SignedData,
		AssetSymbol: utility.BTC,
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

func (processor *TransactionProccessor) retryBatchProcessing(batch model.BatchRequest) error {

	chainTransaction := model.ChainTransaction{}
	if err := processor.Repository.GetByFieldName(&model.ChainTransaction{BatchID : batch.ID}, &chainTransaction); err != nil {
		processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while checking if chain transaction exist for batch with id %+v ", err, batch.ID)
		if err.Error() != utility.SQL_404 {
			return err
		}

		// Checks status of the TXN broadcast to chain
		broadcastedTXNDetails, err := services.GetBroadcastedTXNDetails(batch.ID.String(), processor.Cache, processor.Logger, processor.Config)
		if err != nil {				
			processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while fetching broadcasted transaction status for batch with id %+v", err, batch.ID)
			return err
		}
		if broadcastedTXNDetails.Status == "SUCCESS" {
			// It creates a chain transaction for the batch with the transaction hash returned by crypto adapter
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

		// Fetches all PENDING transactions from the transaction queue table for the given BatchID
		var queuedBatchedTransactions []model.TransactionQueue
		if err := processor.Repository.FetchByFieldName(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING, BatchID : batch.ID, AssetSymbol: utility.BTC}, &queuedBatchedTransactions); err != nil {
			processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v, while fetching batched transactions from the queue", err)
			return err
		}

		if err := processor.processBatch(batch, queuedBatchedTransactions); err != nil {
			processor.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while processing batched transactions with batch id %+v", err, batch.ID)
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

func (processor *TransactionProccessor) UpdateBatchedTransactionsStatus(batch model.BatchRequest, chainTransaction model.ChainTransaction, status string) error  {
	// Fetches all transactions for the given BatchID
	var queuedBatchedTransactions []model.TransactionQueue
	if err := processor.Repository.FetchByFieldName(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING, BatchID : batch.ID, AssetSymbol: utility.BTC}, &queuedBatchedTransactions); err != nil {
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

func (processor TransactionProccessor) confirmBatchTransactions(batchDetails model.BatchRequest, chainTransaction model.ChainTransaction, status string) error {
	
	if err := processor.Repository.Update(&batchDetails, &model.BatchRequest{Status: status}); err != nil {
		return err
	}

	if err := processor.UpdateBatchedTransactionsStatus(batchDetails, chainTransaction, status); err != nil {
		return err
	}
	return nil
}