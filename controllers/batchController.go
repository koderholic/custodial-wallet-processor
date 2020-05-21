package controllers

import (
	"encoding/json"
	"fmt"
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
		// Checks if there is an active batch yet to be processed and return
		batchExist, batchId, err := services.CheckActiveBatchExistAndReturn(controller.Repository, controller.Logger, controller.Config)
		if err != nil {
			controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v, while checking for active batch", err)
			done <- true
		}

		if !batchExist {
			controller.Logger.Info("Outgoing response to ProcessBatchBTCTransactions request %+v", "There is no active BTC batch to process")
			done <- true
		}

		// Fetches all PENDING transactions from the transaction queue table for the given BatchID
		var transactionQueue []model.TransactionQueue
		if err := controller.Repository.FetchByFieldName(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING, BatchID : batchId, AssetSymbol: utility.BTC}, &transactionQueue); err != nil {
			controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v, while fetching batched transactions from the queue", err)
			done <- true
		}

		// It calls the lock service to obtain a lock for the batch transactions
		lockerServiceRequest := dto.LockerServiceRequest{
			Identifier:   fmt.Sprintf("%s%s", controller.Config.LockerPrefix, batchId),
			ExpiresAfter: 600000,
		}
		lockerServiceResponse := dto.LockerServiceResponse{}
		if err := services.AcquireLock(controller.Cache, controller.Logger, controller.Config, lockerServiceRequest, &lockerServiceResponse, &serviceErr); err != nil {
			controller.Logger.Error("Error occured while obtaining lock : %+v; %s", serviceErr, err)
			done <- true
		}

		// Prepare batched transactions for signing
		batchedRecipients := []dto.BatchRecipients{}
		batchedTransactionsIds := []uuid.UUID{}
		queuedBatchedTransactionsIds := []uuid.UUID{}
		floatAccount, err := services.GetHotWalletAddressFor(controller.Cache, controller.Repository.Db(), controller.Logger, controller.Config, utility.BTC)
		if err != nil {
			controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v", err)
			done <- true
		}

		for _, transaction := range transactionQueue {
			recipient := dto.BatchRecipients{
				Address: transaction.Recipient,
				Value:   transaction.Value,
			}
			batchedRecipients = append(batchedRecipients, recipient)
			batchedTransactionsId = append(batchedTransactionsId, transaction.TransactionId)
			queuedBatchedTransactionsId = append(queuedBatchedTransactionsId, transaction.ID)
		}

		signTransactionRequest := dto.BatchBTCRequest{
			AssetSymbol:   utility.BTC,
			ChangeAddress: floatAccount,
			Origins:       []string{floatAccount},
			Recipients:    batchedRecipients,
		}

		// Calls key-management to sign batched transactions
		signTransactionResponse := dto.SignTransactionResponse{}
		if err := services.SignBatchBTCTransaction(nil, controller.Cache, controller.Logger, controller.Config, signTransactionRequest, &signTransactionResponse, serviceErr); err != nil {
			controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v", err)
			if serviceErr.Code == "INSUFFICIENT_BALANCE" {
				processor := &TransactionProccessor{Logger: controller.Logger, Cache: controller.Cache, Config: controller.Config, Repository: controller.Repository}
				if err := processor.ProcessTxnWithInsufficientFloat(utility.BTC); err != nil {
					controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while calling ProcessTxnWithInsufficientFloat", err)
				}
			}
			done <- true
		}

		// Send the signed data to crypto adapter to send to chain
		broadcastToChainRequest := dto.BroadcastToChainRequest{
			SignedData:  signTransactionResponse.SignedData,
			AssetSymbol: utility.BTC,
			ProcessType: utility.WITHDRAWALPROCESS,
		}
		broadcastToChainResponse := dto.BroadcastToChainResponse{}
		if err := services.BroadcastToChain(controller.Cache, controller.Logger, controller.Config, broadcastToChainRequest, &broadcastToChainResponse, serviceErr); err != nil {
			controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while broadcasting to chain", err)
			done <- true
		}

		// It creates a chain transaction for the batch with the transaction hash returned by crypto adapter
		chainTransaction := model.ChainTransaction{
			TransactionHash: broadcastToChainResponse.TransactionHash,
			BatchID: batchId,
		}
		if err := controller.Repository.Create(&chainTransaction); err != nil {
			controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while creating chain transaction", err)
			done <- true
		}

		batchDetails := model.BatchRequest{}
		if err := controller.Repository.GetByFieldName(&model.BatchRequest{BaseModel :model.BaseModel{ID : batchId}}, &batchDetails); err != nil {
			controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while getting active batch details", err)
			done <- true
		}
		if err := controller.Repository.Update(&batchDetails, &model.BatchRequest{Status: model.BatchStatus.PROCESSING}); err != nil {
			controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while updating active batch status to PROCESSING", err)
			done <- true
		}

		tx := controller.Repository.Db().Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()
		if err := tx.Error; err != nil {
			controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while creating db transaction", err)
			done <- true
		}

		if err := tx.Model(model.Transaction{}).Where(batchedTransactionsId).Updates(model.Transaction{TransactionStatus: model.TransactionStatus.PROCESSING, OnChainTxId: chainTransaction.ID}).Error; err != nil {
			tx.Rollback()
			controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while creating db transaction", err)
			done <- true
		}

		if err := tx.Model(model.TransactionQueue{}).Where(queuedBatchedTransactionsId).Updates(model.TransactionQueue{TransactionStatus: model.TransactionStatus.PROCESSING}).Error; err != nil {
			tx.Rollback()
			controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while creating db transaction", err)
			done <- true
		}

		if err := tx.Commit().Error; err != nil {
			controller.Logger.Error("Error response from ProcessBatchBTCTransactions : %+v while creating db transaction", err)
			done <- true
		}

		// The routine returns the lock to the lock service and terminates
		lockReleaseRequest := dto.LockReleaseRequest{
			Identifier: fmt.Sprintf("%s%s", controller.Config.LockerPrefix, batchId),
			Token:      lockerServiceResponse.Token,
		}
		lockReleaseResponse := dto.ServicesRequestSuccess{}
		if err := services.ReleaseLock(controller.Cache, controller.Logger, controller.Config, lockReleaseRequest, &lockReleaseResponse, &serviceErr); err != nil || !lockReleaseResponse.Success {
			controller.Logger.Error("Error occured while releasing lock : %+v; %s", serviceErr, err)
		}
		
		done <- true
	}()

	controller.Logger.Info("Outgoing response to ProcessTransactions request %+v", utility.SUCCESS)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess("SUCCESS", utility.SUCCESS))

	<-done
}
