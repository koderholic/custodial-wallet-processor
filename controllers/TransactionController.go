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
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/tasks"
	"wallet-adapter/utility"

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

// ExternalTransfer ...
func (controller UserAssetController) ExternalTransfer(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	batchService := services.BatchService{BaseService: services.BaseService{Config: controller.Config, Cache: controller.Cache, Logger: controller.Logger}}
	requestData := dto.ExternalTransferRequest{}
	responseData := dto.ExternalTransferResponse{}
	paymentRef := utility.GeneratePaymentRef()

	json.NewDecoder(requestReader.Body).Decode(&requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", errorcode.INPUT_ERR, validationErr), controller.Logger)
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

	if requestData.Network == "" {
		network, err := services.GetDefaultNetworkByAssetSymbol(controller.Repository, debitReferenceTransaction.AssetSymbol)
		if err != nil {
			ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
			return
		}
		requestData.Network = network
	}

	// Check if withdrawal is ACTIVE on this asset
	userAssetService := services.NewService(controller.Cache, controller.Logger, batchService.Config)
	isActive, err := userAssetService.IsWithdrawalActive(debitReferenceTransaction.AssetSymbol, requestData.Network, controller.Repository)
	if err != nil {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}
	if !isActive {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusBadRequest, errorcode.WITHDRAWAL_NOT_ACTIVE, apiResponse.PlainError("INPUT_ERR", errorcode.WITHDRAWAL_NOT_ACTIVE), controller.Logger)
		return
	}

	// Checks to ensure the transaction status of debitReference is completed
	if debitReferenceTransaction.TransactionStatus != model.TransactionStatus.COMPLETED {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusBadRequest, errorcode.INVALID_DEBIT, apiResponse.PlainError("INVALID_DEBIT", errorcode.INVALID_DEBIT), controller.Logger)
		return
	}

	// Checks also that the value matches the value that was initially debited
	value := decimal.NewFromFloat(requestData.Value)
	debitValue, err := decimal.NewFromString(debitReferenceTransaction.Value)
	if err != nil {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", errorcode.SYSTEM_ERR), controller.Logger)
		return
	}
	if value.GreaterThan(debitValue) {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusBadRequest, errorcode.INVALID_DEBIT_AMOUNT, apiResponse.PlainError("INVALID_DEBIT_AMOUNT", errorcode.INVALID_DEBIT_AMOUNT), controller.Logger)
		return
	}

	debitReferenceNetworkAsset, err := services.GetNetworkByAssetAndNetwork(controller.Repository, requestData.Network, debitReferenceTransaction.AssetSymbol)
	if err != nil {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get debitReferenceNetworkAsset with assetSymbol = %s and network : %s", utility.GetSQLErr(err), debitReferenceTransaction.AssetSymbol, debitReferenceTransaction.Network)), controller.Logger)
		return
	}

	// Batch transaction, if asset is batchable
	isBatchable, err := userAssetService.IsBatchable(debitReferenceTransaction.AssetSymbol, requestData.Network, controller.Repository)
	if err != nil {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}
	var activeBatchId uuid.UUID
	if isBatchable {
		activeBatchId, err = batchService.GetWaitingBatchId(controller.Repository, debitReferenceTransaction.AssetSymbol, requestData.Network)
		if err != nil {
			ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", errorcode.SYSTEM_ERR), controller.Logger)
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
		Network:          requestData.Network,
		BatchID:              activeBatchId,
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", errorcode.SYSTEM_ERR), controller.Logger)
		return
	}

	// Create a transaction entry
	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		ReturnError(responseWriter, "ExternalTransfer", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)), controller.Logger)
		return
	}

	// Convert transactionValue to bigInt
	value = utility.NativeValue(debitReferenceNetworkAsset.NativeDecimals, value)

	// Queue transaction up for processing
	queue := model.TransactionQueue{
		Recipient:      requestData.RecipientAddress,
		Value:          value,
		DebitReference: requestData.DebitReference,
		AssetSymbol:    debitReferenceNetworkAsset.AssetSymbol,
		Network:    requestData.Network,
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

	controller.Logger.Info("Outgoing response to ExternalTransfer request %v", http.StatusOK)
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

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusBadRequest, validationErr, apiResponse.Error("INPUT_ERR", errorcode.INPUT_ERR, validationErr), controller.Logger)
		return
	}

	// Get the chain transaction for the request hash
	chainTransaction := model.ChainTransaction{}
	err := controller.Repository.GetChainTransactionByHash(requestData.TransactionHash, &chainTransaction)
	if err != nil {
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("INPUT_ERR", fmt.Sprintf("%s, for get chainTransaction with transactionHash = %s", utility.GetSQLErr(err), requestData.TransactionHash)), controller.Logger)
		return
	}

	// Calls TransactionStatus on crypto adapter to verify the transaction status of the hash
	transactionStatusRequest := dto.TransactionStatusRequest{
		TransactionHash: requestData.TransactionHash,
		AssetSymbol:     chainTransaction.AssetSymbol,
		Network:     chainTransaction.Network,
	}
	transactionStatusResponse := dto.TransactionStatusResponse{}
	if err := services.TransactionStatus(controller.Cache, controller.Logger, controller.Config, transactionStatusRequest, &transactionStatusResponse, &serviceErr); err != nil {
		if serviceErr.Code != "" {
			ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError(utility.SVCS_CRYPTOADAPTER_ERR, serviceErr.Message), controller.Logger)
			return
		}
		ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("%s : %s", errorcode.SYSTEM_ERR, err.Error())), controller.Logger)
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
			ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("error : %s while updating trnasactions tied to chain transaction with id %+v to COMPLETED", err.Error(), chainTransaction.ID)), controller.Logger)
			return
		}
	case utility.FAILED:
		if err := controller.confirmTransactions(chainTransaction, model.BatchStatus.TERMINATED); err != nil {
			ReturnError(responseWriter, "ConfirmTransaction", http.StatusInternalServerError, err, apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("error : %s while updating trnasactions tied to chain transaction with id %+v to TERMINATED", err.Error(), chainTransaction.ID)), controller.Logger)
			return
		}
	default:
		break
	}

	controller.Logger.Info("Outgoing response to ConfirmTransaction request %v", http.StatusOK)
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
				continue
			}

			// update transaction to processing
			if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PROCESSING, model.ChainTransaction{}); err != nil {
				_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
				continue
			}

			if err := processor.processSingleTxn(transaction); err != nil {
				controller.Logger.Error("Transaction with id'%v' could not be processed, confirming broadcast state : %s", transaction.ID, err)
				// Checks status of the TXN broadcast to chain
				txnExist, broadcastedTXNDetails, err := services.GetBroadcastedTXNDetailsByRef(transaction.DebitReference, transaction.AssetSymbol, transaction.Network, processor.Cache, processor.Logger, processor.Config)
				if err != nil {
					_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
					continue
				}

				if !txnExist {
					// Revert the transaction status back to pending, as transaction has not been broadcasted
					if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PENDING, model.ChainTransaction{}); err != nil {
						_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
						continue
					}
					_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
					continue
				}

				chainTransaction := model.ChainTransaction{
					TransactionHash:  broadcastedTXNDetails.TransactionHash,
					RecipientAddress: transaction.Recipient,
					Network: transaction.Network,
				}
				switch broadcastedTXNDetails.Status {
				case utility.FAILED:
					// Create chain transaction and update the transaction status to TERMINATED, as transaction broadcasted failed
					if broadcastedTXNDetails.TransactionHash != "" {
						if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}, &chainTransaction, model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash, Network: transaction.Network }); err != nil {
							processor.Logger.Error("Error : %+v while creating chain transaction for the queued transaction", err, transaction.ID)
						}
					}
					if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.TERMINATED, chainTransaction); err != nil {
						controller.Logger.Error("Error occured while updating the queued transaction (%+v) to TERMINATED : %+v; %s", transaction.ID, serviceErr, err)
					}
					_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
					continue
				case utility.SUCCESSFUL:
					// Create chain transaction and update the transaction status to COMPLETED, as transaction is broadcasted successfully
					if broadcastedTXNDetails.TransactionHash != "" {
						if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}, &chainTransaction, model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash, Network: transaction.Network }); err != nil {
							processor.Logger.Error("Error : %+v while creating chain transaction for the queued transaction", err, transaction.ID)
						}
					}
					if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.COMPLETED, chainTransaction); err != nil {
						controller.Logger.Error("Error occured while updating queued transaction %+v to COMPLETED : %+v; %s", transaction.ID, serviceErr, err)
					}
					_ = processor.releaseLock(transaction.ID.String(), lockerServiceResponse.Token)
					continue
				default:
					// It creates a chain transaction for the broadcasted transaction
					if broadcastedTXNDetails.TransactionHash != "" {
						if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}, &chainTransaction, model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash, Network: transaction.Network }); err != nil {
							processor.Logger.Error("Error : %+v while creating chain transaction for the queued transaction", err, transaction.ID)
						}
					}
					if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PROCESSING, chainTransaction); err != nil {
						controller.Logger.Error("Error occured while updating queued transaction %+v to PROCESSING : %+v; %s", transaction.ID, serviceErr, err)
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

	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess(utility.SUCCESSFUL, utility.SUCCESS))

	<-done
}

func (processor *TransactionProccessor) processSingleTxn(transaction model.TransactionQueue) error {
	serviceErr := dto.ServicesRequestErr{}

	// The routine fetches the float account info from the db and sets the floatAddress as the fromAddress
	var floatAccount model.HotWalletAsset
	if err := processor.Repository.GetByFieldName(&model.HotWalletAsset{AssetSymbol: transaction.AssetSymbol, Network: transaction.Network}, &floatAccount); err != nil {
		if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PENDING, model.ChainTransaction{}); err != nil {
			return err
		}
		return nil
	}

	sendSingleTransactionRequest := dto.SendSingleTransactionRequest{
		FromAddress: floatAccount.Address,
		ToAddress:   transaction.Recipient,
		Amount:      transaction.Value.BigInt(),
		Memo:        transaction.Memo,
		AssetSymbol: transaction.AssetSymbol,
		Network: transaction.Network,
		ProcessType: utility.WITHDRAWALPROCESS,
		Reference:   transaction.DebitReference,
	}
	sendSingleTransactionResponse := dto.SendTransactionResponse{}
	if err := services.SendSingleTransaction(processor.Cache, processor.Logger, processor.Config,
		sendSingleTransactionRequest, &sendSingleTransactionResponse, &serviceErr); err != nil {
		switch serviceErr.Code {
		case errorcode.INSUFFICIENT_FUNDS:
			_ = processor.ProcessTxnWithInsufficientFloat(transaction.AssetSymbol, transaction.Network, *sendSingleTransactionRequest.Amount)
			if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PENDING, model.ChainTransaction{}); err != nil {
				return err
			}
			return nil
		case errorcode.BROADCAST_FAILED_ERR, errorcode.BROADCAST_REJECTED_ERR:
			if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.TERMINATED, model.ChainTransaction{}); err != nil {
				return err
			}
			return nil
		default:
			return err
		}
	}

	// It creates a chain transaction for the transaction with the transaction hash returned by crypto adapter
	chainTransaction := model.ChainTransaction{
		TransactionHash:  sendSingleTransactionResponse.TransactionHash,
		RecipientAddress: transaction.Recipient,
		Network: transaction.Network,
	}
	if err := processor.Repository.Create(&chainTransaction); err != nil {
		return err
	}
	// Update transaction with onChainTransactionId
	if err := processor.updateTransactions(transaction.TransactionId, model.TransactionStatus.PROCESSING, chainTransaction); err != nil {
		return err
	}

	return nil
}

func (processor *TransactionProccessor) ProcessTxnWithInsufficientFloat(assetSymbol, network string, amount big.Int) error {

	DB := database.Database{Logger: processor.Logger, Config: processor.Config, DB: processor.Repository.Db()}
	baseRepository := database.BaseRepository{Database: DB}

	serviceErr := dto.ServicesRequestErr{}
	tasks.NotifyColdWalletUsersViaSMS(amount, assetSymbol, network, processor.Config, processor.Cache, processor.Logger, serviceErr, baseRepository)
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
		return err
	}

	if err := tx.Model(&model.Transaction{}).Where("id IN (?)", transactionsIds).Updates(model.Transaction{TransactionStatus: status}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&model.TransactionQueue{}).Where("transaction_id IN (?)", transactionsIds).Updates(model.TransactionQueue{TransactionStatus: status}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if batchExist {
		dateCompleted := time.Now()
		if err := tx.Model(&batch).Updates(model.BatchRequest{Status: status, DateCompleted: &dateCompleted}).Error; err != nil {
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
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

	transactionQueueDetails := model.TransactionQueue{}
	if err := processor.Repository.Get(&model.TransactionQueue{TransactionId: transactionId}, &transactionQueueDetails); err != nil {
		return err
	}
	if err := tx.Model(&transactionQueueDetails).Updates(&model.TransactionQueue{TransactionStatus: status}).Error; err != nil {
		return err
	}
	transactionDetails := model.Transaction{}
	if err := processor.Repository.Get(&model.Transaction{BaseModel: model.BaseModel{ID: transactionId}}, &transactionDetails); err != nil {
		return err
	}
	if err := tx.Model(&transactionDetails).Updates(&model.Transaction{TransactionStatus: status, OnChainTxId: chainTransaction.ID, Network: transactionQueueDetails.Network}).Error; err != nil {
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
		return err
	}
	return nil
}
