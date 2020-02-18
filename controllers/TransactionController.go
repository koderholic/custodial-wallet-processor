package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
)

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
		controller.Logger.Error("Outgoing response to GetTransactionsByAssetId request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.UUID_CAST_ERR))
		return
	}
	controller.Logger.Info("Incoming request details for GetTransactionsByAssetId : assetID : %+v", assetID)

	if err := controller.Repository.FetchByFieldName(&dto.Transaction{InitiatorID: assetID}, &initiatorTransactions); err != nil {
		if err.Error() != utility.SQL_404 {
			controller.Logger.Error("Outgoing response to GetTransactionsByAssetId request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
			return
		}
	}

	if err := controller.Repository.FetchByFieldName(&dto.Transaction{RecipientID: assetID}, &recipientTransactions); err != nil {
		if err.Error() != utility.SQL_404 {
			controller.Logger.Error("Outgoing response to GetTransactionsByAssetId request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
			return
		}
	}

	for i := 0; i < len(initiatorTransactions); i++ {
		transaction := initiatorTransactions[i]
		tx := model.TransactionResponse{}

		transaction.Map(&tx)

		responseData.Transactions = append(responseData.Transactions, tx)

	}
	for i := 0; i < len(recipientTransactions); i++ {
		receipientTransaction := recipientTransactions[i]
		txRecipient := model.TransactionResponse{}

		receipientTransaction.Map(&txRecipient)

		responseData.Transactions = append(responseData.Transactions, txRecipient)

	}

	if len(responseData.Transactions) <= 0 {
		responseData.Transactions = []model.TransactionResponse{}
	}

	controller.Logger.Info("Outgoing response to GetTransactionsByAssetId request %+v", responseData)
	responseWriter.Header().Set("Content-Type", "application/json")
	json.NewEncoder(responseWriter).Encode(responseData)

}

// ExternalTransfer ...
func (controller UserAssetController) ExternalTransfer(responseWriter http.ResponseWriter, requestReader *http.Request) {

	apiResponse := utility.NewResponse()
	requestData := model.ExternalTransferRequest{}
	responseData := model.ExternalTransferResponse{}
	paymentRef := utility.RandomString(16)

	authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
	decodedToken := model.TokenClaims{}
	_ = utility.DecodeAuthToken(authToken, controller.Config, &decodedToken)

	json.NewDecoder(requestReader.Body).Decode(&requestData)
	controller.Logger.Info("Incoming request details for ExternalTransfer : %+v", requestData)

	// Validate request
	if validationErr := ValidateRequest(controller.Validator, requestData, controller.Logger); len(validationErr) > 0 {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", validationErr)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr))
		return
	}

	// A check is done to ensure the debitReference points to an actual previous debit
	debitReferenceTransaction := dto.Transaction{}
	if err := controller.Repository.FetchByFieldName(&dto.Transaction{TransactionReference: requestData.DebitReference}, &debitReferenceTransaction); err != nil {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// Checks to ensure the transaction status of debitReference is completed
	if debitReferenceTransaction.TransactionStatus != dto.TransactionStatus.COMPLETED {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", utility.INVALID_DEBIT)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INVALID_DEBIT", utility.INVALID_DEBIT))
		return
	}

	// Checks also that the value matches the value that was initially debited
	value := decimal.NewFromFloat(requestData.Value)
	debitValue, err := decimal.NewFromString(debitReferenceTransaction.Value)
	if err != nil {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
		return
	}
	if value.GreaterThan(debitValue) {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INVALID_DEBIT_AMOUNT", utility.INVALID_DEBIT_AMOUNT))
		return
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("Failed to complete external transafer on %s : %s", requestData.DebitReference, err)))
		return
	}
	// Create a transaction record for the transaction on the db for the request
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
	}
	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// If recipient address is on platform, credit recipient asset and complete transaction
	recipientInternalAddress := dto.UserAddress{}
	if err := controller.Repository.FetchByFieldName(&dto.UserAddress{Address: requestData.RecipientAddress}, &recipientInternalAddress); err == nil {
		recipientAsset := dto.UserAssetBalance{}
		if err := controller.Repository.GetAssetsByID(&dto.UserAssetBalance{BaseDTO: dto.BaseDTO{ID: recipientInternalAddress.AssetID}}, &recipientAsset); err != nil {
			controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			if err.Error() == utility.SQL_404 {
				responseWriter.WriteHeader(http.StatusNotFound)
			} else {
				responseWriter.WriteHeader(http.StatusInternalServerError)
			}
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
			return
		}

		recipientBalance, err := decimal.NewFromString(recipientAsset.AvailableBalance)
		if err != nil {
			controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SERVER_ERR", err.Error()))
			return
		}
		recipientNewBalance := (recipientBalance.Add(value)).String()

		if err := tx.Model(&recipientAsset).Updates(dto.UserBalance{AvailableBalance: recipientNewBalance}).Error; err != nil {
			tx.Rollback()
			controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SERVER_ERR", err.Error()))
			return
		}
		if err := tx.Model(&transaction).Updates(dto.Transaction{TransactionStatus: dto.TransactionStatus.COMPLETED}).Error; err != nil {
			tx.Rollback()
			controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SERVER_ERR", err.Error()))
			return
		}

		responseData.TransactionReference = transaction.TransactionReference
		responseData.DebitReference = requestData.DebitReference
		responseData.TransactionStatus = transaction.TransactionStatus

		controller.Logger.Info("Outgoing response to ExternalTransfer request %+v", responseData)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusOK)
		json.NewEncoder(responseWriter).Encode(responseData)
		return
	}

	// Get asset associated with the debit reference
	debitReferenceAsset := dto.UserAssetBalance{}
	if err := controller.Repository.GetAssetsByID(&dto.UserAssetBalance{BaseDTO: dto.BaseDTO{ID: debitReferenceTransaction.RecipientID}}, &debitReferenceAsset); err != nil {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		if err.Error() == utility.SQL_404 {
			responseWriter.WriteHeader(http.StatusNotFound)
		} else {
			responseWriter.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	// Convert value to crypto smallest unit
	denominationDecimal := decimal.NewFromInt(int64(debitReferenceAsset.Decimal))
	baseExp := decimal.NewFromInt(10)
	transactionValue, err := strconv.ParseInt(value.Mul(baseExp.Pow(denominationDecimal)).String(), 10, 64)
	if err != nil {
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
		return
	}

	// Queue transaction up for processing
	queue := dto.TransactionQueue{
		Recipient:      requestData.RecipientAddress,
		Value:          transactionValue,
		DebitReference: requestData.DebitReference,
		Memo:           debitReferenceTransaction.Memo,
		Denomination:   debitReferenceAsset.Symbol,
		TransactionId:  transaction.ID,
	}
	if err := tx.Create(&queue).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)))
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to ExternalTransfer request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
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
		controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", validationErr)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.Error("INPUT_ERR", utility.INPUT_ERR, validationErr))
		return
	}

	tx := controller.Repository.Db().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if err := tx.Error; err != nil {
		controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
		return
	}

	// Get the asset denomination associated with the transaction
	chainTransaction := dto.ChainTransaction{}
	transactionDetails := dto.Transaction{}
	transactionQueueDetails := dto.TransactionQueue{}
	err := controller.Repository.Get(&dto.ChainTransaction{TransactionHash: requestData.TransactionHash}, &chainTransaction)
	if err != nil {
		controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)))
		return
	}
	err = controller.Repository.Get(&dto.Transaction{OnChainTxId: chainTransaction.ID}, &transactionDetails)
	if err != nil {
		controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)))
		return
	}
	err = controller.Repository.GetByFieldName(&dto.TransactionQueue{TransactionId: transactionDetails.ID}, &transactionQueueDetails)
	if err != nil {
		controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)))
		return
	}

	// Calls TransactionStatus on crypto adapter to verify the transaction status
	transactionStatusRequest := model.TransactionStatusRequest{
		TransactionHash: requestData.TransactionHash,
		AssetSymbol:     transactionQueueDetails.Denomination,
	}
	transactionStatusResponse := model.TransactionStatusResponse{}
	if err := services.TransactionStatus(controller.Cache, controller.Logger, controller.Config, transactionStatusRequest, &transactionStatusResponse, &serviceErr); err != nil {
		controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v : %+v", serviceErr, err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		if serviceErr.Code != "" {
			_ = json.NewEncoder(responseWriter).Encode(apiResponse.PlainError(utility.SVCS_CRYPTOADAPTER_ERR, serviceErr.Message))
			return
		}
		_ = json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", fmt.Sprintf("%s : %s", utility.SYSTEM_ERR, err.Error())))
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

	// Goes to chain transaction table, update the status of the chain transaction,
	if err := tx.Model(&chainTransaction).Updates(&chainTransactionUpdate).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.GetSQLErr(err)))
		return
	}
	// With the chainTransactionUpdateId it goes to the transactions table, fetches the transaction mapped to the chainId and updates the status
	if err := tx.Model(&transactionDetails).Updates(&transactionUpdate).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}
	// It goes to the queue table and fetches the queue matching the transactionId and updates the status to either TERMINATED or COMPLETED
	if err := tx.Model(&transactionQueueDetails).Updates(&transactionQueueUpdate).Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("INPUT_ERR", utility.GetSQLErr(err)))
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		controller.Logger.Error("Outgoing response to ConfirmTransaction request %+v", err)
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(responseWriter).Encode(apiResponse.PlainError("SYSTEM_ERR", utility.SYSTEM_ERR))
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

			err := controller.processSingleTxn(transaction)
			if err != nil {
				controller.Logger.Error("The transaction '%+v' could not be processed : ", transaction, err)

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
			fmt.Printf("xlockReleaseRequest >>> ", lockReleaseRequest)
		}
		done <- true
	}()

	controller.Logger.Info("Outgoing response to ProcessTransactions request %+v", utility.SUCCESS)
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(apiResponse.PlainSuccess("SUCCESS", utility.SUCCESS))

	<-done
}

func (controller UserAssetController) processSingleTxn(transaction dto.TransactionQueue) error {
	var floatAccount dto.HotWalletAsset
	serviceErr := model.ServicesRequestErr{}

	// The routine fetches the float account info from the db
	if err := controller.Repository.GetByFieldName(&dto.HotWalletAsset{AssetSymbol: transaction.Denomination}, &floatAccount); err != nil {
		return err
	}

	// It makes a call to crypto adapter to get the current float amount
	onchainBalanceRequest := model.OnchainBalanceRequest{
		Address:     floatAccount.Address,
		AssetSymbol: transaction.Denomination,
	}
	onchainBalanceResponse := model.OnchainBalanceResponse{}
	if err := services.GetOnchainBalance(controller.Cache, controller.Logger, controller.Config, onchainBalanceRequest, &onchainBalanceResponse, &serviceErr); err != nil {
		if serviceErr.Code != "" {
			controller.Logger.Error("Error occured while obtaining lock : %+v", serviceErr)
			return errors.New(serviceErr.Message)
		}
		return err
	}

	floatBalance, err := strconv.ParseInt(onchainBalanceResponse.Balance, 10, 64)
	if err != nil {
		return err
	}

	// If float amount is lesser than the amount to send, it triggers sweep operation
	if floatBalance < transaction.Value {
		// If BTC, trigger BTC sweep
		// Else, trigger sweep for other Asset
		return errors.New("Not enough balance in float for this transaction")
	}

	// Calls key-management to sign transaction
	var signTransactionRequest model.SignTransactionRequest
	signTransactionRequest = model.SignTransactionRequest{
		FromAddress: floatAccount.Address,
		ToAddress:   transaction.Recipient,
		Amount:      transaction.Value,
		Memo:        transaction.Memo,
		CoinType:    transaction.Denomination,
	}
	signTransactionResponse := model.SignTransactionResponse{}
	if err := services.SignTransaction(controller.Cache, controller.Logger, controller.Config, signTransactionRequest, &signTransactionResponse, serviceErr); err != nil {
		if serviceErr.Code != "" {
			controller.Logger.Error("Error occured while obtaining lock : %+v", serviceErr)
			return errors.New(serviceErr.Message)
		}
		return err
	}

	// Send the signed data to crypto adapter to send to chain
	broadcastToChainRequest := model.BroadcastToChainRequest{
		SignedData:  signTransactionResponse.SignedData,
		AssetSymbol: transaction.Denomination,
	}
	broadcastToChainResponse := model.BroadcastToChainResponse{}

	if err := services.BroadcastToChain(controller.Cache, controller.Logger, controller.Config, broadcastToChainRequest, &broadcastToChainResponse, serviceErr); err != nil {
		if serviceErr.Code != "" {
			controller.Logger.Error("Error occured while obtaining lock : %+v", serviceErr)
			return errors.New(serviceErr.Message)
		}
		return err
	}

	// It creates a chain transaction for the transaction with the transaction hash returned by crypto adapter
	chainTransaction := dto.ChainTransaction{
		TransactionHash: broadcastToChainResponse.TransactionHash,
	}
	if err := controller.Repository.Create(&chainTransaction); err != nil {
		return err
	}

	// Updates the transaction status to in progress
	transactionDetails := dto.Transaction{}
	if err = controller.Repository.Get(&dto.Transaction{BaseDTO: dto.BaseDTO{ID: transaction.TransactionId}}, &transactionDetails); err != nil {
		return err
	}
	if err := controller.Repository.Update(&transactionDetails, &dto.Transaction{TransactionStatus: dto.TransactionStatus.PROCESSING, OnChainTxId: chainTransaction.ID}); err != nil {
		return err
	}

	return nil
}
