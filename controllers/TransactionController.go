package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/tasks"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
)

type TransactionProccessor struct {
	Cache          *utility.MemoryCache
	Logger         *utility.Logger
	Config         config.Data
	Repository     database.IUserAssetRepository
	SweepTriggered bool
}

// GetTransaction ... Retrieves the transaction details of the reference sent
func (controller BaseController) GetTransaction(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData dto.TransactionResponse
	var transaction model.Transaction
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	transactionRef := routeParams["reference"]
	controller.Logger.Info("Incoming request details for GetTransaction : transaction reference : %+v", transactionRef)

	if err := controller.Repository.GetByFieldName(&model.Transaction{TransactionReference: transactionRef}, &transaction); err != nil {
		controller.Logger.Error("Outgoing response to GetTransaction request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get transaction with transactionReference = %s", utility.GetSQLErr(err), transactionRef)))
		return
	}

	transaction.Map(&responseData)
	controller.populateChainData(transaction, &responseData, apiResponse, responseWriter)
	controller.Logger.Info("Outgoing response to GetTransaction request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetTransactionsByAssetId ... Retrieves all transactions relating to an asset
func (controller BaseController) GetTransactionsByAssetId(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData dto.TransactionListResponse
	var initiatorTransactions []model.Transaction
	var recipientTransactions []model.Transaction
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		ReturnError(responseWriter, "GetTransactionsByAssetId", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR), controller.Logger)
		return
	}
	controller.Logger.Info("Incoming request details for GetTransactionsByAssetId : assetID : %+v", assetID)
	if err := controller.Repository.FetchByFieldName(&model.Transaction{InitiatorID: assetID}, &initiatorTransactions); err != nil {
		ReturnError(responseWriter, "GetTransactionsByAssetId", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}
	if err := controller.Repository.FetchByFieldName(&model.Transaction{RecipientID: assetID}, &recipientTransactions); err != nil {
		ReturnError(responseWriter, "GetTransactionsByAssetId", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	for i := 0; i < len(initiatorTransactions); i++ {
		transaction := initiatorTransactions[i]
		tx := dto.TransactionResponse{}
		transaction.Map(&tx)
		controller.populateChainData(transaction, &tx, apiResponse, responseWriter)
		responseData.Transactions = append(responseData.Transactions, tx)
	}
	for i := 0; i < len(recipientTransactions); i++ {
		receipientTransaction := recipientTransactions[i]
		txRecipient := dto.TransactionResponse{}
		receipientTransaction.Map(&txRecipient)
		controller.populateChainData(receipientTransaction, &txRecipient, apiResponse, responseWriter)
		responseData.Transactions = append(responseData.Transactions, txRecipient)
	}

	if len(responseData.Transactions) <= 0 {
		responseData.Transactions = []dto.TransactionResponse{}
	}

	controller.Logger.Info("Outgoing response to GetTransactionsByAssetId request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

func (controller BaseController) populateChainData(transaction model.Transaction, txResponse *dto.TransactionResponse, apiResponse utility.ResponseResultObj, responseWriter http.ResponseWriter) {
	//get and populate chain transaction if exists, if this call fails, log error but proceed on
	chainTransaction := model.ChainTransaction{}
	chainData := dto.ChainData{}
	if transaction.TransactionType == "ONCHAIN" && transaction.OnChainTxId != uuid.Nil {
		err := controller.Repository.Get(&model.ChainTransaction{BaseModel: model.BaseModel{ID: transaction.OnChainTxId}}, &chainTransaction)
		if err != nil {
			ReturnError(responseWriter, "GetTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
			txResponse.ChainData = nil
		} else {
			chainTransaction.MaptoDto(&chainData)
			txResponse.ChainData = &chainData
		}
	} else {
		txResponse.ChainData = nil
	}

}

// ExternalTransfer ...
func (controller UserAssetController) ExternalTransfer(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := dto.ExternalTransferRequest{}
	responseData := dto.ExternalTransferResponse{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for ExternalTransfer : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr), controller.Logger)
		return
	}

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := dto.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// A check is done to ensure the debitReference points to an actual previous debit
	debitReferenceTransaction := model.Transaction{}
	if err := controller.Repository.FetchByFieldName(&model.Transaction{TransactionReference: requestData.DebitReference}, &debitReferenceTransaction); err != nil {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	// Checks to ensure the transaction status of debitReference is completed
	if debitReferenceTransaction.TransactionStatus != model.TransactionStatus.COMPLETED {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusBadRequest, utility.INVALID_DEBIT, apiResponse.PlainError("INVALID_DEBIT", utility.INVALID_DEBIT), controller.Logger)
		return
	}

	// Checks also that the value matches the value that was initially debited
	value := decimal.NewFromFloat(requestData.Value)
	debitValue, err := decimal.NewFromString(debitReferenceTransaction.Value)
	if err != nil {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR), controller.Logger)
		return
	}
	if value.GreaterThan(debitValue) {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusBadRequest, utility.INVALID_DEBIT_AMOUNT, apiResponse.PlainError("INVALID_DEBIT_AMOUNT", utility.INVALID_DEBIT_AMOUNT), controller.Logger)
		return
	}

	// Get asset associated with the debit reference
	debitReferenceAsset := model.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: debitReferenceTransaction.RecipientID}}, &debitReferenceAsset); err != nil {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get debitReferenceAsset with id = %s", utility.GetSQLErr(err), debitReferenceTransaction.RecipientID)), controller.Logger)
		return
	}

	// Ensure transaction value is above minimum send to chain
	denominationDecimal := decimal.NewFromInt(int64(debitReferenceAsset.Decimal))
	baseExp := decimal.NewFromInt(10)
	transactionValue, err := strconv.ParseInt(value.Mul(baseExp.Pow(denominationDecimal)).String(), 10, 64)
	if err != nil {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR), controller.Logger)
		return
	}

	if transactionValue <= utility.MINIMUM_SPENDABLE[debitReferenceAsset.AssetSymbol] {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusBadRequest, utility.MINIMUM_SPENDABLE_ERR, apiResponse.PlainError("MINIMUM_SPENDABLE_ERR", fmt.Sprintf("%s : %d", utility.MINIMUM_SPENDABLE_ERR, utility.MINIMUM_SPENDABLE[debitReferenceAsset.AssetSymbol])), controller.Logger)
		return
	}

	// Batch transaction, if asset is BTC
	var activeBatchId uuid.UUID
	if debitReferenceAsset.AssetSymbol == utility.BTC {
		activeBatchId, err = services.GetActiveBTCBatchId(controller.Repository, controller.Logger)
		if err != nil {
			ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR), controller.Logger)
			return
		}
		
	}

	// Build transaction object
	transaction := model.Transaction{
		InitiatorID:          decodedToken.ServiceID,
		RecipientID:          debitReferenceTransaction.RecipientID,
		TransactionReference: requestData.TransactionReference,
		PaymentReference:     paymentRef,
		DebitReference:       requestData.DebitReference,
		Memo:                 debitReferenceTransaction.Memo,
		TransactionType:      model.TransactionType.ONCHAIN,
		TransactionTag:       model.TransactionTag.WITHDRAW,
		Value:                value.String(),
		PreviousBalance:      debitReferenceTransaction.PreviousBalance,
		AvailableBalance:     debitReferenceTransaction.AvailableBalance,
		ProcessingType:       model.ProcessingType.SINGLE,
		TransactionStartDate: time.Now(),
		TransactionEndDate:   time.Now(),
		AssetSymbol:          debitReferenceTransaction.AssetSymbol,
		BatchID : activeBatchId,
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR), controller.Logger)
		return
	}

	// Create a transaction entry
	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	// Queue transaction up for processing
	queue := model.TransactionQueue{
		Recipient:      requestData.RecipientAddress,
		Value:          transactionValue,
		DebitReference: requestData.DebitReference,
		AssetSymbol:    debitReferenceAsset.AssetSymbol,
		TransactionId:  transaction.ID,
		BatchID : activeBatchId,
	}
	if !strings.EqualFold(debitReferenceTransaction.Memo, utility.NO_MEMO) {
		queue.Memo = debitReferenceTransaction.Memo
	}

	if err := tx.Create(&queue).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	if err := tx.Commit().Error; err != nil {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	// Send acknowledgement to the calling service
	responseData.TransactionReference = transaction.TransactionReference
	responseData.DebitReference = requestData.DebitReference
	responseData.TransactionStatus = transaction.TransactionStatus

	controller.Logger.Info("Outgoing response to ExternalTransfer request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// ConfirmTransaction ...
func (controller UserAssetController) ConfirmTransaction(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := dto.ChainData{}
	serviceErr := dto.ServicesRequestErr{}

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for ConfirmTransaction : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr), controller.Logger)
		return
	}

	// Get the asset denomination associated with the transaction
	chainTransaction := model.ChainTransaction{}
	transactionDetails := model.Transaction{}
	transactionQueueDetails := model.TransactionQueue{}
	err := controller.Repository.Get(&model.ChainTransaction{TransactionHash: requestData.TransactionHash}, &chainTransaction)
	if err != nil {
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get chainTransaction with transactionHash = %s", utility.GetSQLErr(err), requestData.TransactionHash)), controller.Logger)
		return
	}
	err = controller.Repository.Get(&model.Transaction{OnChainTxId: chainTransaction.ID}, &transactionDetails)
	if err != nil {
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get transactionDetails with onChainTxId = %s", utility.GetSQLErr(err), chainTransaction.ID)), controller.Logger)
		return
	}
	err = controller.Repository.GetByFieldName(&model.TransactionQueue{TransactionId: transactionDetails.ID}, &transactionQueueDetails)
	if err != nil {
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get transactionQueueDetails with transactionId = %s", utility.GetSQLErr(err), transactionDetails.ID)), controller.Logger)
		return
	}

	// Calls TransactionStatus on crypto adapter to verify the transaction status
	transactionStatusRequest := dto.TransactionStatusRequest{
		TransactionHash: requestData.TransactionHash,
		AssetSymbol:     transactionQueueDetails.AssetSymbol,
	}
	transactionStatusResponse := dto.TransactionStatusResponse{}
	if err := services.TransactionStatus(controller.Cache, controller.Logger, controller.Config, transactionStatusRequest, &transactionStatusResponse, &serviceErr); err != nil {
		if serviceErr.Code != "" {
			ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError(utility.SVCS_CRYPTOADAPTER_ERR, serviceErr.Message), controller.Logger)
			return
		}
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("%s : %s", utility.SYSTEM_ERR, err.Error())), controller.Logger)
		return
	}
	
	// Check if transaction belongs to a batch and return batch
	batchExist, batchDetails, err := services.CheckBatchExistAndReturn(controller.Repository, controller.Logger, chainTransaction.BatchID)
	if err != nil {
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("%s : %s", utility.SYSTEM_ERR, err.Error())), controller.Logger)
		return
	}

	chainTransactionUpdate := model.ChainTransaction{Status: *requestData.Status, TransactionFee: requestData.TransactionFee, BlockHeight: requestData.BlockHeight}
	var transactionUpdate model.Transaction
	var transactionQueueUpdate model.TransactionQueue
	switch transactionStatusResponse.Status {
	case "SUCCESS":
		if batchExist {
			processor := &TransactionProccessor{Logger: controller.Logger, Cache: controller.Cache, Config: controller.Config, Repository: controller.Repository}
			if err := processor.confirmBatchTransactions(batchDetails, chainTransaction, model.BatchStatus.COMPLETED); err != nil {
				ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("%s : %s", "Error while updating batch transactions and batch with id %+v to COMPLETED", err.Error(), batchDetails.ID)), controller.Logger)
				return
			}
		} else {
			transactionUpdate = model.Transaction{TransactionStatus: model.TransactionStatus.COMPLETED}
			transactionQueueUpdate = model.TransactionQueue{TransactionStatus: model.TransactionStatus.COMPLETED}
		}
	case "FAILED":
		if batchExist {
			processor := &TransactionProccessor{Logger: controller.Logger, Cache: controller.Cache, Config: controller.Config, Repository: controller.Repository}
			if err := processor.confirmBatchTransactions(batchDetails, chainTransaction, model.BatchStatus.TERMINATED); err != nil {
				ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("%s : %s", "Error while updating batch transactions and batch with id %+v to TERMINATED", err.Error(), batchDetails.ID)), controller.Logger)
				return
			}
		} else {
			transactionUpdate = model.Transaction{TransactionStatus: model.TransactionStatus.TERMINATED}
			transactionQueueUpdate = model.TransactionQueue{TransactionStatus: model.TransactionStatus.TERMINATED}
		}
	default:
		transactionUpdate = model.Transaction{TransactionStatus: model.TransactionStatus.PROCESSING}
		transactionQueueUpdate = model.TransactionQueue{TransactionStatus: model.TransactionStatus.PROCESSING}
	}

	if !batchExist {
		tx := controller.Repository.Db().Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()
		if err := tx.Error; err != nil {
			ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR), controller.Logger)
			return
		}

		// Goes to chain transaction table, update the status of the chain transaction,
		if err := tx.Model(&chainTransaction).Updates(&chainTransactionUpdate).Error; err != nil {
			tx.Rollback()
			ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
			return
		}
		// With the chainTransactionUpdateId it goes to the transactions table, fetches the transaction mapped to the chainId and updates the status
		if err := tx.Model(&transactionDetails).Updates(&transactionUpdate).Error; err != nil {
			tx.Rollback()
			ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
			return
		}
		// It goes to the queue table and fetches the queue matching the transactionId and updates the status to either TERMINATED or COMPLETED
		if err := tx.Model(&transactionQueueDetails).Updates(&transactionQueueUpdate).Error; err != nil {
			tx.Rollback()
			ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
			return
		}

		if err := tx.Commit().Error; err != nil {
			ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
			return
		}
	}
	

	controller.Logger.Info("Outgoing response to ConfirmTransaction request %+v", apiResponse.PlainSuccess("SUCCESS", utility.SUCCESS))
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess("SUCCESS", utility.SUCCESS))

}

// ProcessTransaction ...
func (controller UserAssetController) ProcessTransactions(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()

	// Endpoint spins up a go-routine to process queued transactions and sends back an acknowledgement to the scheduler
	done := make(chan bool)

	go func() {

		// Fetches all PENDING transactions from the transaction queue table for processing
		var transactionQueue []model.TransactionQueue
		if err := controller.Repository.FetchByFieldName(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING}, &transactionQueue); err != nil {
			controller.Logger.Error("Error response from ProcessTransactions job : %+v", err)
			done <- true
		}
		processor := &TransactionProccessor{Logger: controller.Logger, Cache: controller.Cache, Config: controller.Config, Repository: controller.Repository}

		for _, transaction := range transactionQueue {
			serviceErr := dto.ServicesRequestErr{}

			// It calls the lock service to obtain a lock for the transaction
			lockerServiceRequest := dto.LockerServiceRequest{
				Identifier:   fmt.Sprintf("%s%s", controller.Config.LockerPrefix, transaction.ID),
				ExpiresAfter: 600000,
			}
			lockerServiceResponse := dto.LockerServiceResponse{}
			if err := services.AcquireLock(controller.Cache, controller.Logger, controller.Config, lockerServiceRequest, &lockerServiceResponse, &serviceErr); err != nil {
				controller.Logger.Error("Error occured while obtaining lock : %+v; %s", serviceErr, err)
				continue
			}

			transactionQueueDetails := model.TransactionQueue{}
			if err := controller.Repository.GetByFieldName(&model.TransactionQueue{TransactionId: transaction.TransactionId}, &transactionQueueDetails); err != nil {
				controller.Logger.Error("Error occured while reverting transaction (%s) to pending : %s", transaction.TransactionId, err)
				continue
			}
			if err := controller.Repository.Update(&transactionQueueDetails, &model.TransactionQueue{TransactionStatus: model.TransactionStatus.PROCESSING}); err != nil {
				controller.Logger.Error("Error occured while updating transaction (%s) to On-going : %s", transaction.TransactionId, err)
				continue
			}

			err := processor.processSingleTxn(transaction)
			if err != nil {
				controller.Logger.Error("The transaction '%+v' could not be processed : %s", transaction, err)

				// Revert the transaction status back to pending
				if err := controller.Repository.Update(&transactionQueueDetails, &model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING}); err != nil {
					controller.Logger.Error("Error occured while reverting transaction (%s) to pending : %s", transaction.TransactionId, err)
					continue
				}
			}

			// The routine returns the lock to the lock service and terminates
			lockReleaseRequest := dto.LockReleaseRequest{
				Identifier: fmt.Sprintf("%s%s", controller.Config.LockerPrefix, transaction.ID),
				Token:      lockerServiceResponse.Token,
			}
			lockReleaseResponse := dto.ServicesRequestSuccess{}
			if err := services.ReleaseLock(controller.Cache, controller.Logger, controller.Config, lockReleaseRequest, &lockReleaseResponse, &serviceErr); err != nil || !lockReleaseResponse.Success {
				controller.Logger.Error("Error occured while releasing lock : %+v; %s", serviceErr, err)
			}
		}
		done <- true
	}()

	controller.Logger.Info("Outgoing response to ProcessTransactions request %+v", utility.SUCCESS)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess("SUCCESS", utility.SUCCESS))

	<-done
}

func (processor *TransactionProccessor) processSingleTxn(transaction model.TransactionQueue) error {
	serviceErr := dto.ServicesRequestErr{}

	// The routine fetches the float account info from the db and sets the floatAddress as the fromAddress
	var floatAccount model.HotWalletAsset
	if err := processor.Repository.GetByFieldName(&model.HotWalletAsset{AssetSymbol: transaction.AssetSymbol}, &floatAccount); err != nil {
		return err
	}

	// Get the transaction fee estimate by calling key-management to sign transaction
	signTransactionRequest := dto.SignTransactionRequest{
		FromAddress: floatAccount.Address,
		ToAddress:   transaction.Recipient,
		Amount:      transaction.Value,
		Memo:        transaction.Memo,
		AssetSymbol: transaction.AssetSymbol,
	}
	signTransactionResponse := dto.SignTransactionResponse{}
	if err := services.SignTransaction(processor.Cache, processor.Logger, processor.Config, signTransactionRequest, &signTransactionResponse, &serviceErr); err != nil {
		if serviceErr.Code == "INSUFFICIENT_BALANCE" {
			if err := processor.ProcessTxnWithInsufficientFloat(transaction.AssetSymbol); err != nil {
				return errors.New(serviceErr.Message)
			}
		}
		return err
	}

	// Send the signed data to crypto adapter to send to chain
	broadcastToChainRequest := dto.BroadcastToChainRequest{
		SignedData:  signTransactionResponse.SignedData,
		AssetSymbol: transaction.AssetSymbol,
		Reference:   transaction.DebitReference,
		ProcessType: utility.WITHDRAWALPROCESS,
	}
	broadcastToChainResponse := dto.BroadcastToChainResponse{}

	if err := services.BroadcastToChain(processor.Cache, processor.Logger, processor.Config, broadcastToChainRequest, &broadcastToChainResponse, &serviceErr); err != nil {
		processor.Logger.Error("Error occured while broadcasting transaction : %+v", serviceErr)
		if serviceErr.StatusCode == http.StatusBadRequest {
			tx := processor.Repository.Db().Begin()
			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
				}
			}()
			// Updates the transaction status to TERMINATED
			transactionDetails := model.Transaction{}
			_ = processor.Repository.Get(&model.Transaction{BaseModel: model.BaseModel{ID: transaction.TransactionId}}, &transactionDetails)
			_ = tx.Model(&transactionDetails).Updates(&model.Transaction{TransactionStatus: model.TransactionStatus.TERMINATED})
			// Update transactionQueue to TERMINATED
			transactionQueueDetails := model.TransactionQueue{}
			_ = processor.Repository.Get(&model.TransactionQueue{TransactionId: transaction.TransactionId}, &transactionQueueDetails)
			_ = tx.Model(&transactionQueueDetails).Updates(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.TERMINATED})
			err := tx.Commit().Error
			return err
		}

		// Checks status of the TXN broadcast to chain
		isBroadcastedSuccessfully := services.GetBroadcastedTXNStatusByRef(transaction.DebitReference, processor.Cache, processor.Logger, processor.Config)
		if isBroadcastedSuccessfully {
			return nil
		}

		if serviceErr.Message != "" {
			return errors.New(serviceErr.Message)
		}
		return err
	}

	// It creates a chain transaction for the transaction with the transaction hash returned by crypto adapter
	chainTransaction := model.ChainTransaction{
		TransactionHash: broadcastToChainResponse.TransactionHash,
		RecipientAddress : transaction.Recipient,
	}
	if err := processor.Repository.Create(&chainTransaction); err != nil {
		return err
	}

	// Updates the transaction status to in progress
	transactionDetails := model.Transaction{}
	if err := processor.Repository.Get(&model.Transaction{BaseModel: model.BaseModel{ID: transaction.TransactionId}}, &transactionDetails); err != nil {
		return err
	}
	if err := processor.Repository.Update(&transactionDetails, &model.Transaction{TransactionStatus: model.TransactionStatus.PROCESSING, OnChainTxId: chainTransaction.ID}); err != nil {
		return err
	}

	return nil
}

func (processor *TransactionProccessor) ProcessTxnWithInsufficientFloat(assetSymbol string) error {

	DB := database.Database{Logger: processor.Logger, Config: processor.Config, DB: processor.Repository.Db()}
	baseRepository := database.BaseRepository{Database: DB}

	if !processor.SweepTriggered {
		go tasks.SweepTransactions(processor.Cache, processor.Logger, processor.Config, baseRepository)
		processor.SweepTriggered = true
		return errors.New(fmt.Sprintf("Not enough balance in float for this transaction, triggering sweep operation."))
	}

	return errors.New(fmt.Sprintf("Not enough balance in float for this transaction, sweep operation in progress."))
}
