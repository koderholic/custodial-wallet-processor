package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
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
	batchService := services.BatchService{BaseService: services.BaseService{Config: controller.Config, Cache: controller.Cache, Logger: controller.Logger}}
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
	minimumSpendable := decimal.NewFromFloat(utility.MINIMUM_SPENDABLE[debitReferenceAsset.AssetSymbol])
	if value.Cmp(minimumSpendable) <= 0 {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusBadRequest, utility.MINIMUM_SPENDABLE_ERR, apiResponse.PlainError("MINIMUM_SPENDABLE_ERR", fmt.Sprintf("%s : %d", utility.MINIMUM_SPENDABLE_ERR, utility.MINIMUM_SPENDABLE[debitReferenceAsset.AssetSymbol])), controller.Logger)
		return
	}

	// Batch transaction, if asset is BTC
	var activeBatchId uuid.UUID
	if debitReferenceAsset.AssetSymbol == utility.COIN_BTC {
		activeBatchId, err = batchService.GetWaitingBTCBatchId(controller.Repository, utility.COIN_BTC)
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
		BatchID:              activeBatchId,
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
	value = utility.NativeValue(debitReferenceAsset.Decimal, value)

	// Queue transaction up for processing
	queue := model.TransactionQueue{
		Recipient:      requestData.RecipientAddress,
		Value:          value,
		DebitReference: requestData.DebitReference,
		AssetSymbol:    debitReferenceAsset.AssetSymbol,
		TransactionId:  transaction.ID,
		BatchID:        activeBatchId,
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

	// Get the chain transaction for the request hash
	chainTransaction := model.ChainTransaction{}
	err := controller.Repository.Get(&model.ChainTransaction{TransactionHash: requestData.TransactionHash}, &chainTransaction)
	if err != nil {
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get chainTransaction with transactionHash = %s", utility.GetSQLErr(err), requestData.TransactionHash)), controller.Logger)
		return
	}

	// Calls TransactionStatus on crypto adapter to verify the transaction status of the hash
	transactionStatusRequest := dto.TransactionStatusRequest{
		TransactionHash: requestData.TransactionHash,
		AssetSymbol:     chainTransaction.AssetSymbol,
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

	// update the chain transaction with details of the on-chain TXN,
	chainTransactionUpdate := model.ChainTransaction{Status: *requestData.Status, TransactionFee: requestData.TransactionFee, BlockHeight: requestData.BlockHeight}
	if err := controller.Repository.Update(&chainTransaction, chainTransactionUpdate); err != nil {
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	// Update the transactions on the transaction table and on queue tied to the chain transaction as well as the batch status,if it is a batch transaction
	switch transactionStatusResponse.Status {
	case utility.SUCCESSFUL:
		if err := controller.confirmTransactions(chainTransaction, model.BatchStatus.COMPLETED); err != nil {
			ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("%s : %s", "Error while updating trnasactions tied to chain transaction with id %+v to COMPLETED", err.Error(), chainTransaction.ID)), controller.Logger)
			return
		}
	case utility.FAILED:
		if err := controller.confirmTransactions(chainTransaction, model.BatchStatus.TERMINATED); err != nil {
			ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("%s : %s", "Error while updating trnasactions tied to chain transaction with id %+v to TERMINATED", err.Error(), chainTransaction.ID)), controller.Logger)
			return
		}
	default:
		break
	}

	controller.Logger.Info("Outgoing response to ConfirmTransaction request %+v", apiResponse.PlainSuccess(utility.SUCCESSFUL, utility.SUCCESS))
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess(utility.SUCCESSFUL, utility.SUCCESS))

}

// ProcessTransaction ...
func (controller UserAssetController) ProcessTransactions(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()

	// Endpoint spins up a go-routine to process queued transactions and sends back an acknowledgement to the scheduler
	done := make(chan bool)

	go func() {

		// Fetches all PENDING transactions from the transaction queue table for processing
		var transactionQueue []model.TransactionQueue
		var ETHTransactionCount int
		if err := controller.Repository.FetchByFieldName(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING}, &transactionQueue); err != nil {
			controller.Logger.Error("Error response from ProcessTransactions job : %+v", err)
			done <- true
		}
		processor := &TransactionProccessor{Logger: controller.Logger, Cache: controller.Cache, Config: controller.Config, Repository: controller.Repository}

		// Sort by asset symbol
		sort.Slice(transactionQueue, func(i, j int) bool {
			return transactionQueue[i].AssetSymbol < transactionQueue[j].AssetSymbol
		})

		for _, transaction := range transactionQueue {
			serviceErr := dto.ServicesRequestErr{}

			// Check if the transaction belongs to a batch and return batch
			batchService := services.BatchService{BaseService: services.BaseService{Config: controller.Config, Cache: controller.Cache, Logger: controller.Logger}}
			batchExist, _, err := batchService.CheckBatchExistAndReturn(controller.Repository, transaction.BatchID)
			if err != nil {
				controller.Logger.Error("Error occured while checking if transaction is batched : %s", err)
				continue
			}
			if batchExist {
				continue
			}

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

			// update transaction to processing
			if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PROCESSING); err != nil {
				controller.Logger.Error("Error occured while updating transaction %+v to processing : %+v; %s", transaction.TransactionId, serviceErr, err)
				_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
				continue
			}

			if transaction.AssetSymbol == utility.COIN_ETH {
				time.Sleep(time.Duration(utility.GetSingleTXProcessingIntervalTime(ETHTransactionCount)) * time.Second)
				ETHTransactionCount = ETHTransactionCount + 1
			}

			if err := processor.processSingleTxn(transaction); err == nil {
				controller.Logger.Error("The transaction '%+v' could not be processed : %s", transaction, err)
				// Checks status of the TXN broadcast to chain
				txnExist, broadcastedTXNDetails, err := services.GetBroadcastedTXNDetailsByRef(transaction.DebitReference, transaction.AssetSymbol, processor.Cache, processor.Logger, processor.Config)
				if err != nil {
					processor.Logger.Error("Error checking if queued transaction (%+v) has been broadcasted already, leaving status as ONGOING : %s", transaction.ID, err)
					_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
					continue
				}

				if !txnExist {
					// Revert the transaction status back to pending, as transaction has not been broadcasted
					if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PENDING); err != nil {
						controller.Logger.Error("Error occured while updating transaction %+v to PENDING : %+v; %s", transaction.TransactionId, serviceErr, err)
						_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
						continue
					}
					_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
					continue
				}

				chainTransaction := model.ChainTransaction{
					TransactionHash: broadcastedTXNDetails.TransactionHash,
				}
				switch broadcastedTXNDetails.Status {
				case utility.FAILED:
					// Create chain transaction and update the transaction status to TERMINATED, as transaction broadcasted failed
					if broadcastedTXNDetails.TransactionHash != "" {
						if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}, &chainTransaction, model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}); err != nil {
							processor.Logger.Error("Error : %+v while creating chain transaction for the queued transaction", err, transaction.ID)
						}
					}
					if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.TERMINATED); err != nil {
						controller.Logger.Error("Error occured while updating the queued transaction (%+v) to TERMINATED : %+v; %s", transaction.ID, serviceErr, err)
					}
					_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
					continue
				case utility.SUCCESSFUL:
					// Create chain transaction and update the transaction status to COMPLETED, as transaction is broadcasted successfully
					if broadcastedTXNDetails.TransactionHash != "" {
						if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}, &chainTransaction, model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}); err != nil {
							processor.Logger.Error("Error : %+v while creating chain transaction for the queued transaction", err, transaction.ID)
						}
					}
					if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.COMPLETED); err != nil {
						controller.Logger.Error("Error occured while updating queued transaction %+v to COMPLETED : %+v; %s", transaction.ID, serviceErr, err)
					}
					_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
					continue
				default:
					// It creates a chain transaction for the broadcasted transaction
					if broadcastedTXNDetails.TransactionHash != "" {
						if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}, &chainTransaction, model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}); err != nil {
							processor.Logger.Error("Error : %+v while creating chain transaction for the queued transaction", err, transaction.ID)
						}
					}
					_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
					continue
				}
			}
			// The routine returns the lock to the lock service and terminates
			_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
		}
		done <- true
	}()

	controller.Logger.Info("Outgoing response to ProcessTransactions request %+v", utility.SUCCESS)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess(utility.SUCCESSFUL, utility.SUCCESS))

	<-done
}

func (processor *TransactionProccessor) processSingleTxn(transaction model.TransactionQueue) error {
	serviceErr := dto.ServicesRequestErr{}

	// The routine fetches the float account info from the db and sets the floatAddress as the fromAddress
	var floatAccount model.HotWalletAsset
	if err := processor.Repository.GetByFieldName(&model.HotWalletAsset{AssetSymbol: transaction.AssetSymbol}, &floatAccount); err != nil {
		if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PENDING); err != nil {
			processor.Logger.Error("Error occured while updating queued transaction %+v to PENDING : %+v; %s", transaction.ID, serviceErr, err)
			return err
		}
		return nil
	}

	// Get the transaction fee estimate by calling key-management to sign transaction
	signTransactionRequest := dto.SignTransactionRequest{
		FromAddress: floatAccount.Address,
		ToAddress:   transaction.Recipient,
		Amount:      transaction.Value.BigInt(),
		Memo:        transaction.Memo,
		AssetSymbol: transaction.AssetSymbol,
	}
	signTransactionResponse := dto.SignTransactionResponse{}
	if err := services.SignTransaction(processor.Cache, processor.Logger, processor.Config, signTransactionRequest, &signTransactionResponse, &serviceErr); err != nil {
		processor.Logger.Error("Error occured while signing transaction : %+v", serviceErr)
		if serviceErr.Code == "INSUFFICIENT_BALANCE" {
			_ = processor.ProcessTxnWithInsufficientFloat(transaction.AssetSymbol)
		}
		if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PENDING); err != nil {
			processor.Logger.Error("Error occured while updating queued transaction %+v to PENDING : %+v; %s", transaction.ID, serviceErr, err)
			return err
		}
		return nil
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
			if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.TERMINATED); err != nil {
				processor.Logger.Error("Error occured while updating queued transaction %+v to TERMINATED : %+v; %s", transaction.ID, serviceErr, err)
				return err
			}
			return nil
		}
		return err
	}

	// It creates a chain transaction for the transaction with the transaction hash returned by crypto adapter
	chainTransaction := model.ChainTransaction{
		TransactionHash:  broadcastToChainResponse.TransactionHash,
		RecipientAddress: transaction.Recipient,
	}
	if err := processor.Repository.Create(&chainTransaction); err != nil {
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

func (controller UserAssetController) confirmTransactions(chainTransaction model.ChainTransaction, status string) error {

	batchService := services.BatchService{BaseService: services.BaseService{Config: controller.Config, Cache: controller.Cache, Logger: controller.Logger}}

	// Check if chain transaction belongs to a batch and update batch
	batchExist, batch, err := batchService.CheckBatchExistAndReturn(controller.Repository, chainTransaction.BatchID)
	if err != nil {
		return err
	}

	transactions := []model.Transaction{}
	if err := controller.Repository.FetchByFieldName(&model.Transaction{OnChainTxId: chainTransaction.ID}, &transactions); err != nil {
		return err
	}

	transactionsIds := []uuid.UUID{}

	for _, transaction := range transactions {
		transactionsIds = append(transactionsIds, transaction.ID)
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		controller.Logger.Error("Error response from confirmTransactions : %+v while creating db transaction", err)
		return err
	}

	if err := tx.Model(&model.Transaction{}).Where("id IN (?)", transactionsIds).Updates(model.Transaction{TransactionStatus: status}).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Error response from confirmTransactions : %+v while updating transaction records tied to chain transaction : %+v", err, chainTransaction.ID)
		return err
	}

	if err := tx.Model(&model.TransactionQueue{}).Where("transaction_id IN (?)", transactionsIds).Updates(model.TransactionQueue{TransactionStatus: status}).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Error response from confirmTransactions : %+v while updating transaction queued records for chain transaction : %+v", err, chainTransaction.ID)
		return err
	}

	if batchExist {
		dateCompleted := time.Now()
		if err := tx.Model(&batch).Updates(model.BatchRequest{Status: status, DateCompleted: &dateCompleted}).Error; err != nil {
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		controller.Logger.Error("Error response from confirmTransactions : %+v while commiting db transaction", err)
		return err
	}
	return nil
}

func (processor TransactionProccessor) updateTransactions(transactionId uuid.UUID, status string) error {

	tx := processor.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	// Updates the transaction status to PROCESSING
	transactionDetails := model.Transaction{}
	if err := processor.Repository.Get(&model.Transaction{BaseModel: model.BaseModel{ID: transactionId}}, &transactionDetails); err != nil {
		return err
	}
	if err := tx.Model(&transactionDetails).Updates(&model.Transaction{TransactionStatus: status}).Error; err != nil {
		return err
	}
	// Update transactionQueue to PROCESSING
	transactionQueueDetails := model.TransactionQueue{}
	if err := processor.Repository.Get(&model.TransactionQueue{TransactionId: transactionId}, &transactionQueueDetails); err != nil {
		return err
	}
	if err := tx.Model(&transactionQueueDetails).Updates(&model.TransactionQueue{TransactionStatus: status}).Error; err != nil {
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil

}

func (processor TransactionProccessor) releaseLock(identifier string, lockerserviceToken string) error {
	serviceErr := dto.ServicesRequestErr{}

	lockReleaseRequest := dto.LockReleaseRequest{
		Identifier: fmt.Sprintf("%s%s", processor.Config.LockerPrefix, identifier),
		Token:      lockerserviceToken,
	}
	lockReleaseResponse := dto.ServicesRequestSuccess{}
	if err := services.ReleaseLock(processor.Cache, processor.Logger, processor.Config, lockReleaseRequest, &lockReleaseResponse, &serviceErr); err != nil || !lockReleaseResponse.Success {
		processor.Logger.Error("Error occured while releasing lock for (%+v) : %+v; %s", identifier, serviceErr, err)
		return err
	}
	return nil
}
