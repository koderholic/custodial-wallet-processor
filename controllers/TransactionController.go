package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
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

	var responseData model.TransactionResponse
	var transaction dto.Transaction
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	transactionRef := routeParams["reference"]
	controller.Logger.Info("Incoming request details for GetTransaction : transaction reference : %+v", transactionRef)

	if err := controller.Repository.GetByFieldName(&dto.Transaction{TransactionReference: transactionRef}, &transaction); err != nil {
		controller.Logger.Error("Outgoing response to GetTransaction request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
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

	var responseData model.TransactionListResponse
	var initiatorTransactions []dto.Transaction
	var recipientTransactions []dto.Transaction
	apiResponse := utility.NewResponse()

	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		ReturnError(responseWriter, "GetTransactionsByAssetId", http.StatusBadRequest, err, apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR), controller.Logger)
		return
	}
	controller.Logger.Info("Incoming request details for GetTransactionsByAssetId : assetID : %+v", assetID)
	if err := controller.Repository.FetchByFieldName(&dto.Transaction{InitiatorID: assetID}, &initiatorTransactions); err != nil {
		ReturnError(responseWriter, "GetTransactionsByAssetId", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}
	if err := controller.Repository.FetchByFieldName(&dto.Transaction{RecipientID: assetID}, &recipientTransactions); err != nil {
		ReturnError(responseWriter, "GetTransactionsByAssetId", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	for i := 0; i < len(initiatorTransactions); i++ {
		transaction := initiatorTransactions[i]
		tx := model.TransactionResponse{}
		transaction.Map(&tx)
		controller.populateChainData(transaction, &tx, apiResponse, responseWriter)
		responseData.Transactions = append(responseData.Transactions, tx)
	}
	for i := 0; i < len(recipientTransactions); i++ {
		receipientTransaction := recipientTransactions[i]
		txRecipient := model.TransactionResponse{}
		receipientTransaction.Map(&txRecipient)
		controller.populateChainData(receipientTransaction, &txRecipient, apiResponse, responseWriter)
		responseData.Transactions = append(responseData.Transactions, txRecipient)
	}

	if len(responseData.Transactions) <= 0 {
		responseData.Transactions = []model.TransactionResponse{}
	}

	controller.Logger.Info("Outgoing response to GetTransactionsByAssetId request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

func (controller BaseController) populateChainData(transaction dto.Transaction, txResponse *model.TransactionResponse, apiResponse utility.ResponseResultObj, responseWriter http.ResponseWriter) {
	//get and populate chain transaction if exists, if this call fails, log error but proceed on
	chainTransaction := dto.ChainTransaction{}
	chainData := model.ChainData{}
	if transaction.TransactionType == "ONCHAIN" && transaction.OnChainTxId != uuid.Nil {
		err := controller.Repository.Get(&dto.ChainTransaction{BaseDTO: dto.BaseDTO{ID: transaction.OnChainTxId}}, &chainTransaction)
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
	requestData := model.ExternalTransferRequest{}
	responseData := model.ExternalTransferResponse{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for ExternalTransfer : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr), controller.Logger)
		return
	}

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := model.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	// A check is done to ensure the debitReference points to an actual previous debit
	debitReferenceTransaction := dto.Transaction{}
	if err := controller.Repository.FetchByFieldName(&dto.Transaction{TransactionReference: requestData.DebitReference}, &debitReferenceTransaction); err != nil {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	// Checks to ensure the transaction status of debitReference is completed
	if debitReferenceTransaction.TransactionStatus != dto.TransactionStatus.COMPLETED {
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
	debitReferenceAsset := dto.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: debitReferenceTransaction.RecipientID}}, &debitReferenceAsset); err != nil {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	// Ensure transaction value is above minimum send to chain
	minimumSpendable := decimal.NewFromFloat(utility.MINIMUM_SPENDABLE[debitReferenceAsset.AssetSymbol])
	if value.Cmp(minimumSpendable) <= 0 {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusBadRequest, utility.MINIMUM_SPENDABLE_ERR, apiResponse.PlainError("MINIMUM_SPENDABLE_ERR", fmt.Sprintf("%s : %d", utility.MINIMUM_SPENDABLE_ERR, utility.MINIMUM_SPENDABLE[debitReferenceAsset.AssetSymbol])), controller.Logger)
		return
	}

	// Build transaction object
	transaction := dto.Transaction{
		InitiatorID:          decodedToken.ServiceID,
		RecipientID:          debitReferenceTransaction.RecipientID,
		TransactionReference: requestData.TransactionReference,
		PaymentReference:     paymentRef,
		DebitReference:       requestData.DebitReference,
		Memo:                 debitReferenceTransaction.Memo,
		TransactionType:      dto.TransactionType.ONCHAIN,
		TransactionTag:       dto.TransactionTag.WITHDRAW,
		Value:                value.String(),
		PreviousBalance:      debitReferenceTransaction.PreviousBalance,
		AvailableBalance:     debitReferenceTransaction.AvailableBalance,
		ProcessingType:       dto.ProcessingType.SINGLE,
		TransactionStartDate: time.Now(),
		TransactionEndDate:   time.Now(),
		AssetSymbol:          debitReferenceTransaction.AssetSymbol,
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

	// Convert transactionValue to bigInt
	denominationDecimal := decimal.NewFromInt(int64(debitReferenceAsset.Decimal))
	baseExp := decimal.NewFromInt(10)
	value = value.Mul(baseExp.Pow(denominationDecimal))

	// Queue transaction up for processing
	queue := dto.TransactionQueue{
		Recipient:      requestData.RecipientAddress,
		Value:          value,
		DebitReference: requestData.DebitReference,
		AssetSymbol:    debitReferenceAsset.AssetSymbol,
		TransactionId:  transaction.ID,
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
	requestData := model.ChainData{}
	serviceErr := model.ServicesRequestErr{}

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for ConfirmTransaction : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr), controller.Logger)
		return
	}

	// Get the asset denomination associated with the transaction
	chainTransaction := dto.ChainTransaction{}
	transactionDetails := dto.Transaction{}
	transactionQueueDetails := dto.TransactionQueue{}
	err := controller.Repository.Get(&dto.ChainTransaction{TransactionHash: requestData.TransactionHash}, &chainTransaction)
	if err != nil {
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}
	err = controller.Repository.Get(&dto.Transaction{OnChainTxId: chainTransaction.ID}, &transactionDetails)
	if err != nil {
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}
	err = controller.Repository.GetByFieldName(&dto.TransactionQueue{TransactionId: transactionDetails.ID}, &transactionQueueDetails)
	if err != nil {
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	// Calls TransactionStatus on crypto adapter to verify the transaction status
	transactionStatusRequest := model.TransactionStatusRequest{
		TransactionHash: requestData.TransactionHash,
		AssetSymbol:     transactionQueueDetails.AssetSymbol,
	}
	transactionStatusResponse := model.TransactionStatusResponse{}
	if err := services.TransactionStatus(controller.Cache, controller.Logger, controller.Config, transactionStatusRequest, &transactionStatusResponse, &serviceErr); err != nil {
		if serviceErr.Code != "" {
			ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError(utility.SVCS_CRYPTOADAPTER_ERR, serviceErr.Message), controller.Logger)
			return
		}
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("%s : %s", utility.SYSTEM_ERR, err.Error())), controller.Logger)
		return
	}

	chainTransactionUpdate := dto.ChainTransaction{Status: *requestData.Status, TransactionFee: requestData.TransactionFee, BlockHeight: requestData.BlockHeight}
	var transactionUpdate dto.Transaction
	var transactionQueueUpdate dto.TransactionQueue
	switch transactionStatusResponse.Status {
	case "SUCCESS":
		transactionUpdate = dto.Transaction{TransactionStatus: dto.TransactionStatus.COMPLETED}
		transactionQueueUpdate = dto.TransactionQueue{TransactionStatus: dto.TransactionStatus.COMPLETED}
	case "FAILED":
		transactionUpdate = dto.Transaction{TransactionStatus: dto.TransactionStatus.TERMINATED}
		transactionQueueUpdate = dto.TransactionQueue{TransactionStatus: dto.TransactionStatus.TERMINATED}
	default:
		transactionUpdate = dto.Transaction{TransactionStatus: dto.TransactionStatus.PROCESSING}
		transactionQueueUpdate = dto.TransactionQueue{TransactionStatus: dto.TransactionStatus.PROCESSING}
	}

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
		var transactionQueue []dto.TransactionQueue
		if err := controller.Repository.FetchByFieldName(&dto.TransactionQueue{TransactionStatus: dto.TransactionStatus.PENDING}, &transactionQueue); err != nil {
			controller.Logger.Error("Error response from ProcessTransactions job : %+v", err)
			done <- true
		}
		processor := &TransactionProccessor{Logger: controller.Logger, Cache: controller.Cache, Config: controller.Config, Repository: controller.Repository}

		for _, transaction := range transactionQueue {
			serviceErr := model.ServicesRequestErr{}

			// It calls the lock service to obtain a lock for the transaction
			lockerServiceRequest := model.LockerServiceRequest{
				Identifier:   fmt.Sprintf("%s%s", controller.Config.LockerPrefix, transaction.ID),
				ExpiresAfter: 600000,
			}
			lockerServiceResponse := model.LockerServiceResponse{}
			if err := services.AcquireLock(controller.Cache, controller.Logger, controller.Config, lockerServiceRequest, &lockerServiceResponse, &serviceErr); err != nil {
				controller.Logger.Error("Error occured while obtaining lock : %+v; %s", serviceErr, err)
				continue
			}

			transactionQueueDetails := dto.TransactionQueue{}
			if err := controller.Repository.GetByFieldName(&dto.TransactionQueue{TransactionId: transaction.TransactionId}, &transactionQueueDetails); err != nil {
				controller.Logger.Error("Error occured while reverting transaction (%s) to pending : %s", transaction.TransactionId, err)
				continue
			}
			if err := controller.Repository.Update(&transactionQueueDetails, &dto.TransactionQueue{TransactionStatus: dto.TransactionStatus.PROCESSING}); err != nil {
				controller.Logger.Error("Error occured while updating transaction (%s) to On-going : %s", transaction.TransactionId, err)
				continue
			}

			err := processor.processSingleTxn(transaction)
			if err != nil {
				controller.Logger.Error("The transaction '%+v' could not be processed : %s", transaction, err)

				// Revert the transaction status back to pending
				if err := controller.Repository.Update(&transactionQueueDetails, &dto.TransactionQueue{TransactionStatus: dto.TransactionStatus.PENDING}); err != nil {
					controller.Logger.Error("Error occured while reverting transaction (%s) to pending : %s", transaction.TransactionId, err)
					continue
				}
			}

			// The routine returns the lock to the lock service and terminates
			lockReleaseRequest := model.LockReleaseRequest{
				Identifier: fmt.Sprintf("%s%s", controller.Config.LockerPrefix, transaction.ID),
				Token:      lockerServiceResponse.Token,
			}
			lockReleaseResponse := model.ServicesRequestSuccess{}
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

func (processor *TransactionProccessor) processSingleTxn(transaction dto.TransactionQueue) error {
	serviceErr := model.ServicesRequestErr{}

	// The routine fetches the float account info from the db and sets the floatAddress as the fromAddress
	var floatAccount dto.HotWalletAsset
	if err := processor.Repository.GetByFieldName(&dto.HotWalletAsset{AssetSymbol: transaction.AssetSymbol}, &floatAccount); err != nil {
		return err
	}

	// Get the transaction fee estimate by calling key-management to sign transaction

	// Get transaction denomination
	denomination := dto.Denomination{}
	if err := processor.Repository.GetByFieldName(&dto.Denomination{AssetSymbol: transaction.AssetSymbol, IsEnabled: true}, &denomination); err != nil {
		return err
	}

	// Convert transactionValue to bigInt
	denominationDecimal := decimal.NewFromInt(int64(denomination.Decimal))
	baseExp := decimal.NewFromInt(10)
	transactionValue := new(big.Int)
	_, setStringValidity := transactionValue.SetString(transaction.Value.Mul(baseExp.Pow(denominationDecimal)).String(), 10)
	if !setStringValidity {
		return errors.New("Could not convert transaction value from decimal to bigInt")
	}

	signTransactionRequest := model.SignTransactionRequest{
		FromAddress: floatAccount.Address,
		ToAddress:   transaction.Recipient,
		Amount:      transactionValue,
		Memo:        transaction.Memo,
		AssetSymbol: transaction.AssetSymbol,
	}
	signTransactionResponse := model.SignTransactionResponse{}
	if err := services.SignTransaction(processor.Cache, processor.Logger, processor.Config, signTransactionRequest, &signTransactionResponse, &serviceErr); err != nil {
		if serviceErr.Code == "INSUFFICIENT_BALANCE" {
			if err := processor.ProcessTxnWithInsufficientFloat(transaction.AssetSymbol); err != nil {
				return errors.New(serviceErr.Message)
			}
		}
		return err
	}

	// Send the signed data to crypto adapter to send to chain
	broadcastToChainRequest := model.BroadcastToChainRequest{
		SignedData:  signTransactionResponse.SignedData,
		AssetSymbol: transaction.AssetSymbol,
		Reference:   transaction.DebitReference,
		ProcessType: utility.WITHDRAWALPROCESS,
	}
	broadcastToChainResponse := model.BroadcastToChainResponse{}

	if err := services.BroadcastToChain(processor.Cache, processor.Logger, processor.Config, broadcastToChainRequest, &broadcastToChainResponse, &serviceErr); err != nil {
		processor.Logger.Error("Error occured while broadcasting transaction : %+v", serviceErr)
		if serviceErr.StatusCode == http.StatusBadRequest {
			tx := processor.Repository.Db().Begin()
			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
				}
			}()
			// Updates the transaction status to REJECTED
			transactionDetails := dto.Transaction{}
			_ = processor.Repository.Get(&dto.Transaction{BaseDTO: dto.BaseDTO{ID: transaction.TransactionId}}, &transactionDetails)
			_ = tx.Model(&transactionDetails).Updates(&dto.Transaction{TransactionStatus: dto.TransactionStatus.REJECTED})
			// Update transactionQueue to REJECTED
			transactionQueueDetails := dto.TransactionQueue{}
			_ = processor.Repository.Get(&dto.TransactionQueue{TransactionId: transaction.TransactionId}, &transactionQueueDetails)
			_ = tx.Model(&transactionQueueDetails).Updates(&dto.TransactionQueue{TransactionStatus: dto.TransactionStatus.REJECTED})
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
	chainTransaction := dto.ChainTransaction{
		TransactionHash: broadcastToChainResponse.TransactionHash,
	}
	if err := processor.Repository.Create(&chainTransaction); err != nil {
		return err
	}

	// Updates the transaction status to in progress
	transactionDetails := dto.Transaction{}
	if err := processor.Repository.Get(&dto.Transaction{BaseDTO: dto.BaseDTO{ID: transaction.TransactionId}}, &transactionDetails); err != nil {
		return err
	}
	if err := processor.Repository.Update(&transactionDetails, &dto.Transaction{TransactionStatus: dto.TransactionStatus.PROCESSING, OnChainTxId: chainTransaction.ID}); err != nil {
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
