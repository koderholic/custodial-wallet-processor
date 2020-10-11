package controllers

import (
	"encoding/json"
	"math/big"
	"net/http"
	"sort"
	"time"
	"wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/tasks"
	"wallet-adapter/tasks/float"
	"wallet-adapter/tasks/sweep"
	"wallet-adapter/utility"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/constants"
	"wallet-adapter/utility/errorcode"
	"wallet-adapter/utility/logger"
	Response "wallet-adapter/utility/response"

	"github.com/gorilla/mux"
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
	routeParams := mux.Vars(requestReader)
	transactionRef := routeParams["reference"]
	logger.Info("Incoming request details for GetTransaction : transaction reference : %+v", transactionRef)

	// Get from trabsa service
	TransactionService := services.NewTransactionService(controller.Cache, controller.Config, controller.Repository)
	transaction, err := TransactionService.GetTransaction(transactionRef)
	if err != nil {
		ReturnError(responseWriter, "GetTransaction", err, Response.New().PlainError(err.(appError.Err).ErrType, err.Error()))
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
	routeParams := mux.Vars(requestReader)
	assetID, err := utility.ToUUID(routeParams["assetID"])
	if err != nil {
		ReturnError(responseWriter, "GetTransactionsByAssetId", err, Response.New().PlainError(err.(appError.Err).ErrType, err.Error()))
		return
	}
	logger.Info("Incoming request details for GetTransactionsByAssetId : assetID : %+v", assetID)

	TransactionService := services.NewTransactionService(controller.Cache, controller.Config, controller.Repository)
	transactions, err := TransactionService.GetAllAssetTransactions(assetID)
	if err != nil {
		ReturnError(responseWriter, "GetTransactionsByAssetId", err, Response.New().PlainError(err.(appError.Err).ErrType, err.Error()))
		return
	}

	for i := 0; i < len(transactions); i++ {
		transaction := transactions[i]
		normalizedTx := TransactionService.Normalize(transaction)
		TransactionService.PopulateChainData(&normalizedTx, transaction.OnChainTxId)
		responseData.Transactions = append(responseData.Transactions, normalizedTx)
	}

	if len(responseData.Transactions) <= 0 {
		responseData.Transactions = []dto.TransactionResponse{}
	}

	logger.Info("Outgoing response to GetTransactionsByAssetId request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// ExternalTransfer ...
func (controller TransactionController) ExternalTransfer(responseWriter http.ResponseWriter, requestReader *http.Request) {

	requestData := dto.ExternalTransferRequest{}
	responseData := dto.ExternalTransferResponse{}
	json.NewDecoder(requestReader.Body).Decode(&requestData)
	logger.Info("Incoming request details for ExternalTransfer : %+v", requestData)

	// Validate request
	if err := ValidateRequest(controller.Validator, requestData); len(err.(appError.Err).ErrData.([]map[string]string)) > 0 {
		appErr := err.(appError.Err)
		ReturnError(responseWriter, "CreateUserAssets", err, Response.New().Error(appErr.ErrType, err.Error(), appErr.ErrData))
		return
	}

	// A check is done to ensure the debitReference points to an actual previous debit
	TransactionService := services.NewTransactionService(controller.Cache, controller.Config, controller.Repository)
	transactionStatus, err := TransactionService.ExternalTx(requestData, controller.GetInitiatingServiceId(requestReader))
	if err != nil {
		ReturnError(responseWriter, "ExternalTransfer", err, Response.New().PlainError(err.(appError.Err).ErrType, err.Error()))
		return
	}

	// Send acknowledgement to the calling service
	responseData.TransactionReference = requestData.TransactionReference
	responseData.DebitReference = requestData.DebitReference
	responseData.TransactionStatus = transactionStatus

	logger.Info("Outgoing response to ExternalTransfer request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(responseData)

}

// ConfirmTransaction ...
func (controller TransactionController) ConfirmTransaction(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := Response.New()
	requestData := dto.ChainData{}
	json.NewDecoder(requestReader.Body).Decode(&requestData)
	logger.Info("Incoming request details for ConfirmTransaction : %+v", requestData)

	// Validate request
	if err := ValidateRequest(controller.Validator, requestData); len(err.(appError.Err).ErrData.([]map[string]string)) > 0 {
		appErr := err.(appError.Err)
		ReturnError(responseWriter, "CreateUserAssets", err, apiResponse.Error(appErr.ErrType, err.Error(), appErr.ErrData))
		return
	}

	// A check is done to ensure the debitReference points to an actual previous debit
	TransactionService := services.NewTransactionService(controller.Cache, controller.Config, controller.Repository)
	if err := TransactionService.ConfirmTransaction(requestData.TransactionHash); err != nil {
		ReturnError(responseWriter, "ExternalTransfer", err, Response.New().PlainError(err.(appError.Err).ErrType, err.Error()))
		return
	}

	logger.Info("Outgoing response to ConfirmTransaction request %+v", apiResponse.PlainSuccess(constants.SUCCESSFUL, constants.SUCCESS))
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess(constants.SUCCESSFUL, constants.SUCCESS))

}

// ProcessTransaction ...
func (controller TransactionController) ProcessTransactions(responseWriter http.ResponseWriter, requestReader *http.Request) {
	// Endpoint spins up a go-routine to process queued transactions and sends back an acknowledgement to the scheduler
	done := make(chan bool)

	go func() {

		TransactionService := services.NewTransactionService(controller.Cache, controller.Config, controller.Repository)
		transactionQueue, err := TransactionService.FetchPendingTransactionsInQueue()
		if err != nil {
			done <- true
		}
		processor := &TransactionProccessor{Cache: controller.Cache, Config: controller.Config, Repository: controller.Repository}

		// Sort by asset symbol
		sort.Slice(transactionQueue, func(i, j int) bool {
			return transactionQueue[i].AssetSymbol < transactionQueue[j].AssetSymbol
		})

		for _, transaction := range transactionQueue {
			transactionLockToken, err := processor.PrepareSingleTxProcessing(transaction)
			if err != nil || transactionLockToken == "" {
				if err != nil {
					logger.Error("Error occured while preparing to process transaction %+v, error : %s", transaction, err)
				}
				continue
			}

			LockerService := services.NewLockerService(controller.Cache, controller.Config, controller.Repository)
			if err := processor.ProcessSingleTxn(transaction); err != nil {
				logger.Error("Error occured while processing transaction %+v, error : %s, proceeding to verify transaction broadcast status", transaction, err)

				CryptoAdapterService := services.NewCryptoAdapterService(controller.Cache, controller.Config, controller.Repository)
				txnExist, broadcastedTXNDetails, err := CryptoAdapterService.GetBroadcastedTXNDetailsByRefAndSymbol(transaction.DebitReference, transaction.AssetSymbol)
				if err != nil {
					logger.Error("Error checking if queued transaction (%+v) has been broadcasted already, leaving status as ONGOING : %s", transaction.ID, err)
					_ = LockerService.ReleaseLock(transaction.ID.String(), transactionLockToken)
					continue
				}

				if err := processor.ConfirmTxOnchainStatusAndUpdate(txnExist, broadcastedTXNDetails, transaction); err != nil {
					logger.Error("Error occured while preparing to process transaction %+v, error : %s", transaction, err)
					_ = LockerService.ReleaseLock(transaction.ID.String(), transactionLockToken)
					continue
				}
				_ = LockerService.ReleaseLock(transaction.ID.String(), transactionLockToken)
			}
			_ = LockerService.ReleaseLock(transaction.ID.String(), transactionLockToken)
		}
		done <- true
	}()

	logger.Info("Outgoing response to ProcessTransactions request %+v", constants.SUCCESS)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(Response.New().PlainSuccess(constants.SUCCESSFUL, constants.SUCCESS))
	<-done
}

func (processor *TransactionProccessor) PrepareSingleTxProcessing(transaction model.TransactionQueue) (string, error) {
	// Check if the transaction belongs to a batch and return batch
	BatchService := services.NewBatchService(processor.Cache, processor.Config, processor.Repository)
	batchExist, _, err := BatchService.CheckBatchExistAndReturn(transaction.BatchID)
	if err != nil {
		return "", err
	}
	if batchExist {
		return "", nil
	}

	LockerService := services.NewLockerService(processor.Cache, processor.Config, processor.Repository)
	lockerServiceResponse, err := LockerService.AcquireLock(transaction.ID.String(), constants.SIX_HUNDRED_MILLISECONDS)
	if err != nil {
		return "", err
	}

	// update transaction to processing
	TransactionService := services.NewTransactionService(processor.Cache, processor.Config, processor.Repository)
	if err := TransactionService.UpdateTransactionByTxID(transaction.TransactionId, model.TransactionStatus.PROCESSING, model.ChainTransaction{}); err != nil {
		_ = LockerService.ReleaseLock(transaction.ID.String(), lockerServiceResponse.Token)
		return "", err
	}

	var ETHTransactionCount int
	if transaction.AssetSymbol == constants.COIN_ETH {
		time.Sleep(time.Duration(utility.GetSingleTXProcessingIntervalTime(ETHTransactionCount)) * time.Second)
		ETHTransactionCount = ETHTransactionCount + 1
	}
	return lockerServiceResponse.Token, nil
}

func (processor *TransactionProccessor) ConfirmTxOnchainStatusAndUpdate(txnExist bool, broadcastedTXNDetails dto.TransactionStatusResponse, transaction model.TransactionQueue) error {
	TransactionService := services.NewTransactionService(processor.Cache, processor.Config, processor.Repository)
	if !txnExist {
		// Revert the transaction status back to pending, as transaction has not been broadcasted
		if err := TransactionService.UpdateTransactionByTxID(transaction.TransactionId, model.TransactionStatus.PENDING, model.ChainTransaction{}); err != nil {
			return err
		}
		return nil
	}

	chainTransaction := model.ChainTransaction{
		TransactionHash:  broadcastedTXNDetails.TransactionHash,
		RecipientAddress: transaction.Recipient,
	}
	switch broadcastedTXNDetails.Status {
	case constants.FAILED:
		if broadcastedTXNDetails.TransactionHash != "" {
			if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}, &chainTransaction, model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}); err != nil {
				return err
			}
		}
		if err := TransactionService.UpdateTransactionByTxID(transaction.TransactionId, model.TransactionStatus.TERMINATED, chainTransaction); err != nil {
			return err
		}
		return nil
	case constants.SUCCESSFUL:
		if broadcastedTXNDetails.TransactionHash != "" {
			if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}, &chainTransaction, model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}); err != nil {
				return err
			}
		}
		if err := TransactionService.UpdateTransactionByTxID(transaction.TransactionId, model.TransactionStatus.COMPLETED, chainTransaction); err != nil {
			return err
		}
		return nil
	default:
		// It creates a chain transaction for the broadcasted transaction
		if broadcastedTXNDetails.TransactionHash != "" {
			if err := processor.Repository.UpdateOrCreate(model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}, &chainTransaction, model.ChainTransaction{TransactionHash: broadcastedTXNDetails.TransactionHash}); err != nil {
				return err
			}
		}
		if err := TransactionService.UpdateTransactionByTxID(transaction.TransactionId, model.TransactionStatus.PROCESSING, chainTransaction); err != nil {
			return err
		}
		return nil
	}
}

func (processor *TransactionProccessor) ProcessSingleTxn(transaction model.TransactionQueue) error {
	TransactionService := services.NewTransactionService(processor.Cache, processor.Config, processor.Repository)
	floatAddress, err := float.GetFloatAddressFor(processor.Repository, transaction.AssetSymbol)
	if err != nil {
		if err := TransactionService.UpdateTransactionByTxID(transaction.TransactionId, model.TransactionStatus.PENDING, model.ChainTransaction{}); err != nil {
			return err
		}
		return nil
	}

	signTransactionAndBroadcastRequest := dto.SignTransactionRequest{
		FromAddress: floatAddress,
		ToAddress:   transaction.Recipient,
		Amount:      transaction.Value.BigInt(),
		Memo:        transaction.Memo,
		AssetSymbol: transaction.AssetSymbol,
		ProcessType: constants.WITHDRAWALPROCESS,
		Reference:   transaction.DebitReference,
	}
	signTransactionAndBroadcastResponse := dto.SignAndBroadcastResponse{}
	KeyManagementService := services.NewKeyManagementService(processor.Cache, processor.Config, processor.Repository)
	broadcastErr := KeyManagementService.SignTransactionAndBroadcast(signTransactionAndBroadcastRequest, &signTransactionAndBroadcastResponse)

	if err := processor.UpdateSingleTxBySignAdBroaqdcastResponse(transaction, signTransactionAndBroadcastResponse, broadcastErr); err != nil {
		return err
	}
	return nil
}

func (processor *TransactionProccessor) UpdateSingleTxBySignAdBroaqdcastResponse(transaction model.TransactionQueue, broadcastResp dto.SignAndBroadcastResponse, broadcastErr error) error {
	TransactionService := services.NewTransactionService(processor.Cache, processor.Config, processor.Repository)
	if broadcastErr != nil {

		switch broadcastErr.(appError.Err).ErrType {
		case errorcode.INSUFFICIENT_FUNDS:
			processor.ProcessTxnWithInsufficientFloat(transaction.AssetSymbol, *transaction.Value.BigInt())
			if err := TransactionService.UpdateTransactionByTxID(transaction.TransactionId, model.TransactionStatus.PENDING, model.ChainTransaction{}); err != nil {
				return err
			}
			return nil
		case errorcode.BROADCAST_FAILED_ERR, errorcode.BROADCAST_REJECTED_ERR:
			if err := TransactionService.UpdateTransactionByTxID(transaction.TransactionId, model.TransactionStatus.TERMINATED, model.ChainTransaction{}); err != nil {
				return err
			}
			return nil
		default:
			return broadcastErr
		}
	}

	// It creates a chain transaction for the transaction with the transaction hash returned by crypto adapter
	chainTransaction := model.ChainTransaction{
		TransactionHash:  broadcastResp.TransactionHash,
		RecipientAddress: transaction.Recipient,
	}
	if err := processor.Repository.Create(&chainTransaction); err != nil {
		return err
	}
	// Update transaction with onChainTransactionId
	if err := TransactionService.UpdateTransactionByTxID(transaction.TransactionId, model.TransactionStatus.PROCESSING, chainTransaction); err != nil {
		logger.Error("Error occured while updating queued transaction %+v to PROCESSING, error : %s", transaction.ID, err)
		return err
	}
	return nil
}

func (processor *TransactionProccessor) ProcessTxnWithInsufficientFloat(assetSymbol string, amount big.Int) {

	DB := database.Database{Config: processor.Config, DB: processor.Repository.Db()}
	baseRepository := database.BaseRepository{Database: DB}
	tasks.NotifyColdWalletUsersViaSMS(amount, assetSymbol, processor.Config, processor.Cache, processor.Repository)
	if processor.SweepTriggered {
		go sweep.SweepTransactions(processor.Cache, processor.Config, &baseRepository)
		processor.SweepTriggered = true
	}
}
