package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
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
	"wallet-adapter/tasks/sweep"
	"wallet-adapter/utility"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/constants"
	"wallet-adapter/utility/errorcode"
	"wallet-adapter/utility/jwt"
	"wallet-adapter/utility/logger"
	Response "wallet-adapter/utility/response"
	"wallet-adapter/utility/variables"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
)

type TransactionProccessor struct {
	Cache          *cache.Memory
	Config         config.Data
	Repository     database.ITransactionRepository
	SweepTriggered bool
}

// GetTransaction ... Retrieves the transaction details of the reference sent
func (controller TransactionController) GetTransaction(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData dto.TransactionResponse
	apiResponse := Response.New()

	routeParams := mux.Vars(requestReader)
	transactionRef := routeParams["reference"]
	logger.Info("Incoming request details for GetTransaction : transaction reference : %+v", transactionRef)

	// Get from trabsa service
	TransactionService := services.NewTransactionService(controller.Cache, controller.Config, controller.Repository, nil)
	transaction, err := TransactionService.GetTransaction(transactionRef)
	if err != nil {
		err := err.(appError.Err)
		ReturnError(responseWriter, "GetTransaction", err, apiResponse.PlainError(err.ErrType, err.Error()))
		return
	}
	responseData = TransactionService.Normalize(transaction)
	TransactionService.PopulateChainData(&responseData, transaction.OnChainTxId)

	logger.Info("Outgoing response to GetTransaction request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// GetTransactionsByAssetId ... Retrieves all transactions relating to an asset
func (controller TransactionController) GetTransactionsByAssetId(responseWriter http.ResponseWriter, requestReader *http.Request) {

	var responseData dto.TransactionListResponse
	var initiatorTransactions []model.Transaction
	var recipientTransactions []model.Transaction
	apiResponse := Response.New()

	TransactionService := services.NewTransactionService(controller.Cache, controller.Config, controller.Repository, nil)

	routeParams := mux.Vars(requestReader)
	assetID, err := uuid.FromString(routeParams["assetId"])
	if err != nil {
		ReturnError(responseWriter, "GetTransactionsByAssetId", err, apiResponse.PlainError(errorcode.INPUT_ERR_CODE, errorcode.UUID_CAST_ERR))
		return
	}
	logger.Info("Incoming request details for GetTransactionsByAssetId : assetID : %+v", assetID)
	if err := controller.Repository.FetchByFieldName(&model.Transaction{InitiatorID: assetID}, &initiatorTransactions); err != nil {
		ReturnError(responseWriter, "GetTransactionsByAssetId", err, apiResponse.PlainError(errorcode.INPUT_ERR_CODE, appError.GetSQLErr(err)))
		return
	}
	if err := controller.Repository.FetchByFieldName(&model.Transaction{RecipientID: assetID}, &recipientTransactions); err != nil {
		ReturnError(responseWriter, "GetTransactionsByAssetId", err, apiResponse.PlainError(errorcode.INPUT_ERR_CODE, appError.GetSQLErr(err)))
		return
	}

	for i := 0; i < len(initiatorTransactions); i++ {
		transaction := initiatorTransactions[i]
		tx := dto.TransactionResponse{}
		tx = TransactionService.Normalize(transaction)
		TransactionService.PopulateChainData(&tx, transaction.OnChainTxId)
		responseData.Transactions = append(responseData.Transactions, tx)
	}
	for i := 0; i < len(recipientTransactions); i++ {
		receipientTransaction := recipientTransactions[i]
		txRecipient := dto.TransactionResponse{}
		txRecipient = TransactionService.Normalize(receipientTransaction)
		TransactionService.PopulateChainData(&txRecipient, receipientTransaction.OnChainTxId)
		responseData.Transactions = append(responseData.Transactions, txRecipient)
	}

	if len(responseData.Transactions) <= 0 {
		responseData.Transactions = []dto.TransactionResponse{}
	}

	logger.Info("Outgoing response to GetTransactionsByAssetId request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

func (controller TransactionController) updateTransactions(transaction model.TransactionQueue, status string, chainTransaction model.ChainTransaction) error {

	BatchService := services.NewBatchService(controller.Cache, controller.Config, controller.Repository)
	batchExist, batch, err := BatchService.CheckBatchExistAndReturn(transaction.BatchID)
	if err != nil {
		return err
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		logger.Error("Error response from updateTransactions : %+v while creating db transaction", err)
		return err
	}

	if batchExist {
		if err := tx.Model(&model.Transaction{}).Where("batch_id = ?", transaction.BatchID).Updates(model.Transaction{TransactionStatus: status}).Error; err != nil {
			tx.Rollback()
			logger.Error("Error response from updateTransactions : %+v while updating transactions with batchId : %+v", err, transaction.BatchID)
			return err
		}
		if err := tx.Model(&model.TransactionQueue{}).Where("batch_id = ?", transaction.BatchID).Updates(model.TransactionQueue{TransactionStatus: status}).Error; err != nil {
			tx.Rollback()
			logger.Error("Error response from updateTransactions : %+v while updating queued transactions with batchId  : %+v", err, transaction.ID)
			return err
		}
		dateCompleted := time.Now()
		if err := tx.Model(&batch).Updates(model.BatchRequest{Status: status, DateCompleted: &dateCompleted}).Error; err != nil {
			return err
		}
	} else {
		if err := tx.Model(&model.Transaction{}).Where("id = ?", transaction.TransactionId).Updates(model.Transaction{TransactionStatus: status}).Error; err != nil {
			tx.Rollback()
			logger.Error("Error response from updateTransactions : %+v while updating transaction with id : %+v", err, transaction.TransactionId)
			return err
		}
		if err := tx.Model(&model.TransactionQueue{}).Where("id = ?", transaction.ID).Updates(model.TransactionQueue{TransactionStatus: status}).Error; err != nil {
			tx.Rollback()
			logger.Error("Error response from updateTransactions : %+v while updating queued transaction with id  : %v", err, transaction.ID)
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		logger.Error("Error response from updateTransactions : %+v while commiting db transaction", err)
		return err
	}
	return nil

}

// ExternalTransfer ...
func (controller TransactionController) ExternalTransfer(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := Response.New()
	requestData := dto.ExternalTransferRequest{}
	responseData := dto.ExternalTransferResponse{}
	paymentRef := utility.RandomString(16)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	logger.Info("Incoming request details for ExternalTransfer : %+v", requestData)

	// Validate request
	if err := ValidateRequest(controller.Validator, requestData); len(err.(appError.Err).ErrData.([]map[string]string)) > 0 {
		appErr := err.(appError.Err)
		ReturnError(responseWriter, "CreateUserAssets", err, apiResponse.Error(appErr.ErrType, err.Error(), appErr.ErrData))
		return
	}

	authToken := requestReader.Header.Get(jwt.X_AUTH_TOKEN)
	decodedToken := dto.TokenClaims{}
	_ = jwt.DecodeToken(authToken, controller.Config, &decodedToken)

	// A check is done to ensure the debitReference points to an actual previous debit
	debitReferenceTransaction := model.Transaction{}
	if err := controller.Repository.FetchByFieldName(&model.Transaction{TransactionReference: requestData.DebitReference}, &debitReferenceTransaction); err != nil {
		ReturnError(responseWriter, "ExternalTransfer", err, apiResponse.PlainError("INPUT_ERR", appError.GetSQLErr(err)))
		return
	}

	// Check if withdrawal is ACTIVE on this asset
	DenominationServices := services.NewDenominationServices(controller.Cache, controller.Config, controller.Repository, nil)
	isActive, err := DenominationServices.IsWithdrawalActive(debitReferenceTransaction.AssetSymbol)
	if err != nil {
		ReturnError(responseWriter, "ExternalTransfer", err, apiResponse.PlainError("SERVER_ERR", appError.GetSQLErr(err)))
		return
	}
	if !isActive {
		ReturnError(responseWriter, "ExternalTransfer", errorcode.WITHDRAWAL_NOT_ACTIVE, apiResponse.PlainError("INPUT_ERR", errorcode.WITHDRAWAL_NOT_ACTIVE))
		return
	}

	// Checks to ensure the transaction status of debitReference is completed
	if debitReferenceTransaction.TransactionStatus != model.TransactionStatus.COMPLETED {
		ReturnError(responseWriter, "ExternalTransfer", errorcode.INVALID_DEBIT, apiResponse.PlainError("INVALID_DEBIT", errorcode.INVALID_DEBIT))
		return
	}

	// Checks also that the value matches the value that was initially debited
	value := decimal.NewFromFloat(requestData.Value)
	debitValue, err := decimal.NewFromString(debitReferenceTransaction.Value)
	if err != nil {
		ReturnError(responseWriter, "ExternalTransfer", err, apiResponse.PlainError("SERVER_ERR", errorcode.SERVER_ERR))
		return
	}
	if value.GreaterThan(debitValue) {
		ReturnError(responseWriter, "ExternalTransfer", errorcode.INVALID_DEBIT_AMOUNT, apiResponse.PlainError("INVALID_DEBIT_AMOUNT", errorcode.INVALID_DEBIT_AMOUNT))
		return
	}

	// Get asset associated with the debit reference
	debitReferenceAsset := model.UserAsset{}
	if err := controller.Repository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: debitReferenceTransaction.RecipientID}}, &debitReferenceAsset); err != nil {
		ReturnError(responseWriter, "ExternalTransfer", err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get debitReferenceAsset with id = %s", appError.GetSQLErr(err), debitReferenceTransaction.RecipientID)))
		return
	}

	// Ensure transaction value is above minimum send to chain
	minimumSpendable := decimal.NewFromFloat(variables.MINIMUM_SPENDABLE[debitReferenceAsset.AssetSymbol])
	if value.Cmp(minimumSpendable) <= 0 {
		ReturnError(responseWriter, "ExternalTransfer", errorcode.MINIMUM_SPENDABLE_ERR, apiResponse.PlainError("MINIMUM_SPENDABLE_ERR", fmt.Sprintf("%s : %v", errorcode.MINIMUM_SPENDABLE_ERR, variables.MINIMUM_SPENDABLE[debitReferenceAsset.AssetSymbol])))
		return
	}

	// Batch transaction, if asset is BTC
	var activeBatchId uuid.UUID
	if debitReferenceAsset.AssetSymbol == constants.COIN_BTC {
		BatchService := services.NewBatchService(controller.Cache, controller.Config, controller.Repository)
		activeBatchId, err = BatchService.GetWaitingBatchId(constants.COIN_BTC)
		if err != nil {
			ReturnError(responseWriter, "ExternalTransfer", err, apiResponse.PlainError("SERVER_ERR", errorcode.SERVER_ERR))
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
		ReturnError(responseWriter, "ExternalTransfer", err, apiResponse.PlainError("SERVER_ERR", errorcode.SERVER_ERR))
		return
	}

	// Create a transaction entry
	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "ExternalTransfer", err, apiResponse.PlainError("SERVER_ERR", appError.GetSQLErr(err)))
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
	if !strings.EqualFold(debitReferenceTransaction.Memo, constants.NO_MEMO) {
		queue.Memo = debitReferenceTransaction.Memo
	}

	if err := tx.Create(&queue).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "ExternalTransfer", err, apiResponse.PlainError("SERVER_ERR", appError.GetSQLErr(err)))
		return
	}

	if err := tx.Commit().Error; err != nil {
		ReturnError(responseWriter, "ExternalTransfer", err, apiResponse.PlainError("SERVER_ERR", appError.GetSQLErr(err)))
		return
	}

	// Send acknowledgement to the calling service
	responseData.TransactionReference = transaction.TransactionReference
	responseData.DebitReference = requestData.DebitReference
	responseData.TransactionStatus = transaction.TransactionStatus

	logger.Info("Outgoing response to ExternalTransfer request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// ConfirmTransaction ...
func (controller TransactionController) ConfirmTransaction(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := Response.New()
	requestData := dto.ChainData{}
	serviceErr := dto.ExternalServicesRequestErr{}

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	logger.Info("Incoming request details for ConfirmTransaction : %+v", requestData)

	// Validate request
	if err := ValidateRequest(controller.Validator, requestData); len(err.(appError.Err).ErrData.([]map[string]string)) > 0 {
		appErr := err.(appError.Err)
		ReturnError(responseWriter, "CreateUserAssets", err, apiResponse.Error(appErr.ErrType, err.Error(), appErr.ErrData))
		return
	}

	// Get the chain transaction for the request hash
	chainTransaction := model.ChainTransaction{}
	err := controller.Repository.Get(&model.ChainTransaction{TransactionHash: requestData.TransactionHash}, &chainTransaction)
	if err != nil {
		ReturnError(responseWriter, "ConfirmTransaction", err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get chainTransaction with transactionHash = %s", appError.GetSQLErr(err), requestData.TransactionHash)))
		return
	}

	// Calls TransactionStatus on crypto adapter to verify the transaction status of the hash
	transactionStatusRequest := dto.TransactionStatusRequest{
		TransactionHash: requestData.TransactionHash,
		AssetSymbol:     chainTransaction.AssetSymbol,
	}
	transactionStatusResponse := dto.TransactionStatusResponse{}
	CryptoAdapterService := services.NewCryptoAdapterService(controller.Cache, controller.Config, controller.Repository, &serviceErr)
	if err := CryptoAdapterService.TransactionStatus(transactionStatusRequest, &transactionStatusResponse); err != nil {
		if serviceErr.Code != "" {
			ReturnError(responseWriter, "ConfirmTransaction", err, apiResponse.PlainError(constants.SVCS_CRYPTOADAPTER_ERR, serviceErr.Message))
			return
		}
		ReturnError(responseWriter, "ConfirmTransaction", err, apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("%s : %s", errorcode.SERVER_ERR, err.Error())))
		return
	}

	// update the chain transaction with details of the on-chain TXN,
	chainTransactionUpdate := model.ChainTransaction{Status: *requestData.Status, TransactionFee: requestData.TransactionFee, BlockHeight: requestData.BlockHeight}
	if err := controller.Repository.Update(&chainTransaction, chainTransactionUpdate); err != nil {
		ReturnError(responseWriter, "ConfirmTransaction", err, apiResponse.PlainError("SERVER_ERR", appError.GetSQLErr(err)))
		return
	}

	// Update the transactions on the transaction table and on queue tied to the chain transaction as well as the batch status,if it is a batch transaction
	switch transactionStatusResponse.Status {
	case constants.SUCCESSFUL:
		if err := controller.confirmTransactions(chainTransaction, model.BatchStatus.COMPLETED); err != nil {
			ReturnError(responseWriter, "ConfirmTransaction", err, apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("error : %s while updating trnasactions tied to chain transaction with id %+v to COMPLETED", err.Error(), chainTransaction.ID)))
			return
		}
	case constants.FAILED:
		if err := controller.confirmTransactions(chainTransaction, model.BatchStatus.TERMINATED); err != nil {
			ReturnError(responseWriter, "ConfirmTransaction", err, apiResponse.PlainError("SERVER_ERR", fmt.Sprintf("error : %s while updating trnasactions tied to chain transaction with id %+v to TERMINATED", err.Error(), chainTransaction.ID)))
			return
		}
	default:
		break
	}

	logger.Info("Outgoing response to ConfirmTransaction request %+v", apiResponse.PlainSuccess(constants.SUCCESSFUL, constants.SUCCESS))
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess(constants.SUCCESSFUL, constants.SUCCESS))

}

// ProcessTransaction ...
func (controller TransactionController) ProcessTransactions(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := Response.New()

	// Endpoint spins up a go-routine to process queued transactions and sends back an acknowledgement to the scheduler
	done := make(chan bool)

	go func() {

		// Fetches all PENDING transactions from the transaction queue table for processing
		var transactionQueue []model.TransactionQueue
		var ETHTransactionCount int
		if err := controller.Repository.FetchByFieldName(&model.TransactionQueue{TransactionStatus: model.TransactionStatus.PENDING}, &transactionQueue); err != nil {
			logger.Error("Error response from ProcessTransactions job : %+v", err)
			done <- true
		}
		processor := &TransactionProccessor{Cache: controller.Cache, Config: controller.Config, Repository: controller.Repository}

		// Sort by asset symbol
		sort.Slice(transactionQueue, func(i, j int) bool {
			return transactionQueue[i].AssetSymbol < transactionQueue[j].AssetSymbol
		})

		for _, transaction := range transactionQueue {
			serviceErr := dto.ExternalServicesRequestErr{}

			// Check if the transaction belongs to a batch and return batch
			BatchService := services.NewBatchService(controller.Cache, controller.Config, controller.Repository)
			batchExist, _, err := BatchService.CheckBatchExistAndReturn(transaction.BatchID)
			if err != nil {
				logger.Error("Error occured while checking if transaction is batched : %s", err)
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
			LockerService := services.NewLockerService(controller.Cache, controller.Config, controller.Repository, &serviceErr)
			if err := LockerService.AcquireLock(lockerServiceRequest, &lockerServiceResponse); err != nil {
				logger.Error("Error occured while obtaining lock : %+v; %s", serviceErr, err)
				continue
			}

			// update transaction to processing
			if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PROCESSING, model.ChainTransaction{}); err != nil {
				logger.Error("Error occured while updating transaction %+v to processing : %+v; %s", transaction.TransactionId, serviceErr, err)
				_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
				continue
			}

			if transaction.AssetSymbol == constants.COIN_ETH {
				time.Sleep(time.Duration(utility.GetSingleTXProcessingIntervalTime(ETHTransactionCount)) * time.Second)
				ETHTransactionCount = ETHTransactionCount + 1
			}

			if err := processor.processSingleTxn(transaction); err != nil {
				logger.Error("The transaction '%+v' could not be processed : %s", transaction, err)
				// Checks status of the TXN broadcast to chain
				CryptoAdapterService := services.NewCryptoAdapterService(controller.Cache, controller.Config, controller.Repository, &serviceErr)
				txnExist, broadcastedTXNDetails, err := CryptoAdapterService.GetBroadcastedTXNDetailsByRefAndSymbol(transaction.DebitReference, transaction.AssetSymbol)
				if err != nil {
					logger.Error("Error checking if queued transaction (%+v) has been broadcasted already, leaving status as ONGOING : %s", transaction.ID, err)
					_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
					continue
				}

				if !txnExist {
					// Revert the transaction status back to pending, as transaction has not been broadcasted
					if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PENDING, model.ChainTransaction{}); err != nil {
						logger.Error("Error occured while updating transaction %+v to PENDING : %+v; %s", transaction.TransactionId, serviceErr, err)
						_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
						continue
					}
					_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
					continue
				}

				chainTransaction := model.ChainTransaction{
					TransactionHash:  broadcastedTXNDetails.TransactionHash,
					RecipientAddress: transaction.Recipient,
				}
				switch broadcastedTXNDetails.Status {
				case constants.FAILED:
					// Create chain transaction and update the transaction status to TERMINATED, as transaction broadcasted failed
					if broadcastedTXNDetails.TransactionHash != "" {
						if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}, &chainTransaction, model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}); err != nil {
							logger.Error("Error : %+v while creating chain transaction for the queued transaction", err, transaction.ID)
						}
					}
					if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.TERMINATED, chainTransaction); err != nil {
						logger.Error("Error occured while updating the queued transaction (%+v) to TERMINATED : %+v; %s", transaction.ID, serviceErr, err)
					}
					_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
					continue
				case constants.SUCCESSFUL:
					// Create chain transaction and update the transaction status to COMPLETED, as transaction is broadcasted successfully
					if broadcastedTXNDetails.TransactionHash != "" {
						if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}, &chainTransaction, model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}); err != nil {
							logger.Error("Error : %+v while creating chain transaction for the queued transaction", err, transaction.ID)
						}
					}
					if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.COMPLETED, chainTransaction); err != nil {
						logger.Error("Error occured while updating queued transaction %+v to COMPLETED : %+v; %s", transaction.ID, serviceErr, err)
					}
					_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
					continue
				default:
					// It creates a chain transaction for the broadcasted transaction
					if broadcastedTXNDetails.TransactionHash != "" {
						if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}, &chainTransaction, model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}); err != nil {
							logger.Error("Error : %+v while creating chain transaction for the queued transaction", err, transaction.ID)
						}
					}
					if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PROCESSING, chainTransaction); err != nil {
						logger.Error("Error occured while updating queued transaction %+v to PROCESSING : %+v; %s", transaction.ID, serviceErr, err)
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

	logger.Info("Outgoing response to ProcessTransactions request %+v", constants.SUCCESS)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess(constants.SUCCESSFUL, constants.SUCCESS))

	<-done
}

func (processor *TransactionProccessor) processSingleTxn(transaction model.TransactionQueue) error {
	serviceErr := dto.ExternalServicesRequestErr{}

	// The routine fetches the float account info from the db and sets the floatAddress as the fromAddress
	var floatAccount model.HotWalletAsset
	if err := processor.Repository.GetByFieldName(&model.HotWalletAsset{AssetSymbol: transaction.AssetSymbol}, &floatAccount); err != nil {
		if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PENDING, model.ChainTransaction{}); err != nil {
			logger.Error("Error occured while updating queued transaction %+v to PENDING : %+v; %s", transaction.ID, serviceErr, err)
			return err
		}
		return nil
	}

	// Get the transaction fee estimate by calling key-management to sign transaction
	signTransactionAndBroadcastRequest := dto.SignTransactionRequest{
		FromAddress: floatAccount.Address,
		ToAddress:   transaction.Recipient,
		Amount:      transaction.Value.BigInt(),
		Memo:        transaction.Memo,
		AssetSymbol: transaction.AssetSymbol,
		ProcessType: constants.WITHDRAWALPROCESS,
		Reference:   transaction.DebitReference,
	}
	signTransactionAndBroadcastResponse := dto.SignAndBroadcastResponse{}
	KeyManagementService := services.NewKeyManagementService(processor.Cache, processor.Config, processor.Repository, &serviceErr)
	if err := KeyManagementService.SignTransactionAndBroadcast(signTransactionAndBroadcastRequest, &signTransactionAndBroadcastResponse); err != nil {
		logger.Error("Error occured while signing and broadcast queued transaction %+v : %+v", transaction.ID, serviceErr)
		switch serviceErr.Code {
		case errorcode.INSUFFICIENT_FUNDS:
			_ = processor.ProcessTxnWithInsufficientFloat(transaction.AssetSymbol, *signTransactionAndBroadcastRequest.Amount)
			if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PENDING, model.ChainTransaction{}); err != nil {
				logger.Error("Error occured while updating queued transaction %+v to PENDING : %+v; %s", transaction.ID, serviceErr, err)
				return err
			}
			return nil
		case errorcode.BROADCAST_FAILED_ERR, errorcode.BROADCAST_REJECTED_ERR:
			if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.TERMINATED, model.ChainTransaction{}); err != nil {
				logger.Error("Error occured while updating queued transaction %+v to TERMINATED : %+v; %s", transaction.ID, serviceErr, err)
				return err
			}
			return nil
		default:
			return err
		}
	}

	// It creates a chain transaction for the transaction with the transaction hash returned by crypto adapter
	chainTransaction := model.ChainTransaction{
		TransactionHash:  signTransactionAndBroadcastResponse.TransactionHash,
		RecipientAddress: transaction.Recipient,
	}
	if err := processor.Repository.Create(&chainTransaction); err != nil {
		return err
	}
	// Update transaction with onChainTransactionId
	if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PROCESSING, chainTransaction); err != nil {
		logger.Error("Error occured while updating queued transaction %+v to PROCESSING : %+v; %s", transaction.ID, serviceErr, err)
		return err
	}

	return nil
}

func (processor *TransactionProccessor) ProcessTxnWithInsufficientFloat(assetSymbol string, amount big.Int) error {

	DB := database.Database{Config: processor.Config, DB: processor.Repository.Db()}
	baseRepository := database.BaseRepository{Database: DB}

	serviceErr := dto.ExternalServicesRequestErr{}
	tasks.NotifyColdWalletUsersViaSMS(amount, assetSymbol, processor.Config, processor.Cache, serviceErr, processor.Repository)
	if !processor.SweepTriggered {
		go sweep.SweepTransactions(processor.Cache, processor.Config, &baseRepository)
		processor.SweepTriggered = true
		return errors.New(fmt.Sprintf("Not enough balance in float for this transaction, triggering sweep operation."))
	}
	return errors.New(fmt.Sprintf("Not enough balance in float for this transaction, sweep operation in progress."))
}

func (controller TransactionController) confirmTransactions(chainTransaction model.ChainTransaction, status string) error {

	BatchService := services.NewBatchService(controller.Cache, controller.Config, controller.Repository)

	// Check if chain transaction belongs to a batch and update batch
	batchExist, batch, err := BatchService.CheckBatchExistAndReturn(chainTransaction.BatchID)
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
		logger.Error("Error response from confirmTransactions : %+v while creating db transaction", err)
		return err
	}

	if err := tx.Model(&model.Transaction{}).Where("id IN (?)", transactionsIds).Updates(model.Transaction{TransactionStatus: status}).Error; err != nil {
		tx.Rollback()
		logger.Error("Error response from confirmTransactions : %+v while updating transaction records tied to chain transaction : %+v", err, chainTransaction.ID)
		return err
	}

	if err := tx.Model(&model.TransactionQueue{}).Where("transaction_id IN (?)", transactionsIds).Updates(model.TransactionQueue{TransactionStatus: status}).Error; err != nil {
		tx.Rollback()
		logger.Error("Error response from confirmTransactions : %+v while updating transaction queued records for chain transaction : %+v", err, chainTransaction.ID)
		return err
	}

	if batchExist {
		dateCompleted := time.Now()
		if err := tx.Model(&batch).Updates(model.BatchRequest{Status: status, DateCompleted: &dateCompleted}).Error; err != nil {
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		logger.Error("Error response from confirmTransactions : %+v while commiting db transaction", err)
		return err
	}
	return nil
}

func (processor TransactionProccessor) updateTransactions(transactionId uuid.UUID, status string, chainTransaction model.ChainTransaction) error {

	tx := processor.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	transactionDetails := model.Transaction{}
	if err := processor.Repository.Get(&model.Transaction{BaseModel: model.BaseModel{ID: transactionId}}, &transactionDetails); err != nil {
		return err
	}
	if err := tx.Model(&transactionDetails).Updates(&model.Transaction{TransactionStatus: status, OnChainTxId: chainTransaction.ID}).Error; err != nil {
		return err
	}
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
	serviceErr := dto.ExternalServicesRequestErr{}

	lockReleaseRequest := dto.LockReleaseRequest{
		Identifier: fmt.Sprintf("%s%s", processor.Config.LockerPrefix, identifier),
		Token:      lockerserviceToken,
	}
	lockReleaseResponse := dto.ServicesRequestSuccess{}
	LockerService := services.NewLockerService(processor.Cache, processor.Config, processor.Repository, &serviceErr)
	if err := LockerService.ReleaseLock(lockReleaseRequest, &lockReleaseResponse); err != nil || !lockReleaseResponse.Success {
		logger.Error("verifyTransactionStatus logs :Error occured while releasing lock for (%+v) : %+v; %s", identifier, serviceErr, err)
		return err
	}
	return nil
}
