package tasks

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"
	"wallet-adapter/utility/constants"

	"github.com/robfig/cron/v3"
	uuid "github.com/satori/go.uuid"
)

// SweepParam ... Model definition for batch sweep
type (
	SweepParam struct {
		FloatAddress     string
		BrokerageAddress string
		FloatPercent     int64
		BrokeragePercent int64
	}
)

func SweepTransactions(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository) {
	logger.Info("Sweep operation begins")
	serviceErr := dto.ServicesRequestErr{}
	token, err := AcquireLock("sweep", utility.SIX_HUNDRED_MILLISECONDS, cache, logger, config, serviceErr)
	if err != nil {
		logger.Error("Could not acquire lock", err)
		return
	}

	var transactions []model.Transaction
	if err := repository.FetchSweepCandidates(&transactions); err != nil {
		logger.Error("Error response from Sweep job : could not fetch sweep candidates %+v", err)
		if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
			logger.Error("Could not release lock", err)
			return
		}
		return
	}

	var binanceDepositTransactions []model.Transaction
	if err := repository.FetchBinanceOnchainSweepCandidates(&binanceDepositTransactions); err != nil {
		logger.Error("Error response from Sweep job : could not fetch binance internal deposit sweep candidates %+v", err)
		if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
			logger.Error("Could not release lock", err)
			return
		}
		return
	}
	for _, transaction := range binanceDepositTransactions {
		if utility.IsValidUUID(transaction.TransactionReference) {
			transactions = append(transactions, transaction)
		}
	}

	logger.Info("Fetched %d sweep candidates", len(transactions))

	var batchAddresses []string
	var batchAssetTransactionsToSweep []model.Transaction
	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	for _, tx := range transactions {
		//Filter batchable assets, save in a seperate list for batch processing and skip individual processing
		//need recipient Asset to check assetSymbol
		recipientAsset := model.UserAsset{}
		//all the tx in assetTransactions have the same recipientId so just pass the 0th position
		if err := userAssetRepository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: tx.RecipientID}}, &recipientAsset); err != nil {
			logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
			if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
				logger.Error("Could not release lock", err)
				return
			}
			return
		}

		userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
		txNetworkAsset, err := services.GetNetworkByAssetAndNetwork(&userAssetRepository, tx.Network, tx.AssetSymbol)
		if err != nil {
			continue
		}


		if *txNetworkAsset.IsBatchable {
			//get recipient address for transaction
			chainTransaction := model.ChainTransaction{}
			err = getChainTransaction(repository, tx, &chainTransaction, logger)
			if err != nil {
				logger.Error("Error response from Sweep job, could not get chain transaction :"+
					" %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
				continue

			}
			batchAddresses = append(batchAddresses, chainTransaction.RecipientAddress)
			batchAssetTransactionsToSweep = append(batchAssetTransactionsToSweep, tx)
			//skip futher processing for this asset, will be included a part of batch processing
			continue
		}
	}
	batchAddresses = ToUniqueAddresses(batchAddresses)
	//remove btc transactions from list of remaining transactions
	transactions = RemoveBatchTransactions(transactions, batchAssetTransactionsToSweep)
	//Do other Coins apart from batchable assets
	transactionsPerAddressPerAssetSymbol, err := GroupTxByAddressByAssetSymbol(transactions, repository, logger)
	if err != nil {
		logger.Error("Error grouping By Address", err)
		return
	}
	for addressAndAssetSymbol, addressTransactions := range transactionsPerAddressPerAssetSymbol {
		stringSlice := strings.Split(addressAndAssetSymbol, utility.SWEEP_GROUPING_SEPERATOR)
		var address = stringSlice[0]
		sum := CalculateSum(addressTransactions)
		logger.Info("Sweeping %s with total of %d", address, sum)
		if err := sweepPerAddress(cache, logger, config, repository, userAssetRepository, serviceErr, addressTransactions, sum, address); err != nil {
			logger.Error("Error response from Sweep job : %+v while sweepPerAddress for address %s", err, address)
			continue
		}
	}
	//batch process btc
	if len(batchAddresses) > 0 {
		transactionsPerAssetSymbol, batchAddressesPerAssetSymbol, _ := GroupTxByAssetSymbol(batchAssetTransactionsToSweep, repository, logger)
		for assetSymbol, addressTransactions := range transactionsPerAssetSymbol{
			if err := sweepBatchTx(cache, logger, config, repository, userAssetRepository, serviceErr, batchAddressesPerAssetSymbol[assetSymbol], addressTransactions); err != nil {
				logger.Error("Error response from Sweep job : %+v while sweeping batch transactions", err)
				if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
					logger.Error("Could not release lock", err)
					return
				}
				return
			}
		}

	}
	if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
		logger.Error("Could not release lock", err)
		return
	}
	logger.Info("Sweep operation ends successfully, lock released")
}

func CalculateSum(addressTransactions []model.Transaction) float64 {
	//Get total sum to be swept for this assetId address
	var sum float64
	for _, tx := range addressTransactions {
		balance, _ := strconv.ParseFloat(tx.Value, 64)
		sum = sum + balance
	}
	return sum
}

func CalculateSumOfBatch(addressTransactions []model.Transaction) float64 {
	//Get total sum to be swept for this batch
	var sum = float64(0)
	for _, tx := range addressTransactions {
		value, _ := strconv.ParseFloat(tx.Value, 64)
		sum += value
	}
	return sum
}

func sweepBatchTx(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, userAssetRepository database.UserAssetRepository, serviceErr dto.ServicesRequestErr, batchAddresses []string, batchAssetTransactionsToSweep []model.Transaction) error {

	txNetworkAsset, err := services.GetNetworkByAssetAndNetwork(&userAssetRepository, batchAssetTransactionsToSweep[0].Network, batchAssetTransactionsToSweep[0].AssetSymbol)
	if err != nil {
		return err
	}

	recipientData := []dto.BatchRecipients{}
	//get float
	floatAccount, err := getFloatDetails(repository, txNetworkAsset.AssetSymbol, txNetworkAsset.Network, logger)
	if err != nil {
		return err
	}

	//check total sum threshold for this batch
	totalSweepSum := CalculateSumOfBatch(batchAssetTransactionsToSweep)
	if totalSweepSum < txNetworkAsset.MinimumSweepable {
		logger.Error("Error response from sweep job : Total sweep sum %v for asset (%s) is below the minimum sweep %v, so terminating sweep process", totalSweepSum, txNetworkAsset.AssetSymbol, txNetworkAsset.MinimumSweepable, err)
		return err
	}

	sweepParam, err := GetSweepParams(cache, logger, config, repository, floatAccount, txNetworkAsset, totalSweepSum)
	if err != nil {
		logger.Error("Error response from Sweep job : %+v while getting sweep params for %s", err, floatAccount.AssetSymbol)
		return err
	}

	if sweepParam.FloatPercent != int64(0) {
		floatRecipient := dto.BatchRecipients{
			Address: sweepParam.FloatAddress,
			Value:   sweepParam.FloatPercent,
		}
		recipientData = append(recipientData, floatRecipient)
	}
	if sweepParam.BrokeragePercent != int64(0) {
		brokerageRecipient := dto.BatchRecipients{
			Address: sweepParam.BrokerageAddress,
			Value:   sweepParam.BrokeragePercent,
		}
		recipientData = append(recipientData, brokerageRecipient)
	}
	sendBatchTransactionRequest := dto.BatchRequest{
		AssetSymbol:   txNetworkAsset.AssetSymbol,
		ChangeAddress: sweepParam.BrokerageAddress,
		IsSweep:       true,
		Origins:       batchAddresses,
		Recipients:    recipientData,
		ProcessType:   utility.SWEEPPROCESS,
		Network: txNetworkAsset.Network,
		Reference:     fmt.Sprintf("SWEEP-%s-%d", txNetworkAsset.AssetSymbol, time.Now().Unix()),
	}
	sendBatchTransactionResponse := dto.SendTransactionResponse{}
	if err := services.SendBatchTransaction(nil, cache, logger, config, sendBatchTransactionRequest, &sendBatchTransactionResponse, serviceErr); err != nil {
		logger.Error("Error response from SendBatchTransaction : %+v while sweeping batch transactions", err)
		return err
	}
	if err := updateSweptStatus(batchAssetTransactionsToSweep, repository, logger); err != nil {
		return err
	}
	return nil

}

func sweepPerAddress(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, userAssetRepository database.UserAssetRepository, serviceErr dto.ServicesRequestErr, addressTransactions []model.Transaction, sum float64, recipientAddress string) error {
	transactionListInfo, e := getTransactionListInfo(repository, addressTransactions, logger)
	if e != nil {
		return e
	}

	var userAddress model.UserAddress
	err := repository.GetByFieldName(&model.UserAddress{Address: recipientAddress}, &userAddress)
	if err != nil {
		logger.Error("Error getting address provider, defaulting to BUNDLE")
		userAddress.AddressProvider = model.AddressProvider.BUNDLE
	}

	if userAddress.AddressProvider == model.AddressProvider.BINANCE {
		//call Binance brokerage service
		service := services.BaseService{Config: config, Cache: cache, Logger: logger}
		_, sweepErr := service.SweepUserAddress(transactionListInfo.UserId, transactionListInfo.AssetSymbol, utility.FloatToString(sum))
		if sweepErr != nil {
			logger.Error("Error response from Binance Brokerage service : %+v while sweeping for address with id %+v", sweepErr, recipientAddress)
			return sweepErr
		}
		if err := updateSweptStatus(addressTransactions, repository, logger); err != nil {
			return err
		}
		return nil
	}

	txNetworkAsset, err := services.GetNetworkByAssetAndNetwork(&userAssetRepository, transactionListInfo.Network, transactionListInfo.AssetSymbol)
	if  err != nil {
		logger.Error("Error response from sweep job : %+v while trying to denomination of float asset, network is %+v and assetSymbol is %+v", err, transactionListInfo.Network, transactionListInfo.AssetSymbol)
		return err
	}

	if txNetworkAsset.CoinType == constants.TRX_COINTYPE {
		isExceededLimit := HasExceededTrxSweepLimit(userAddress, logger, transactionListInfo.AssetSymbol, repository)
		if isExceededLimit {
			return nil
		}
	}

	//Check that sweep amount is not below the minimum sweep amount
	isAmountSufficient, err := CheckSweepMinimum(txNetworkAsset, config, sum, logger)
	if !isAmountSufficient {
		return err
	}

	floatAccount, err := getFloatDetails(repository, transactionListInfo.AssetSymbol, transactionListInfo.Network, logger)
	if err != nil {
		return err
	}

	toAddress, addressMemo, err := GetSweepAddressAndMemo(cache, logger, config, repository, floatAccount, txNetworkAsset)
	if err != nil {
		logger.Error("Error response from Sweep job : %+v while getting sweep toAddress and memo for %s", err, floatAccount.AssetSymbol)
		return err
	}

	//Do this only for BEp-2 tokens and not for BNB itself
	if txNetworkAsset.RequiresMemo && *txNetworkAsset.IsToken {
		//send sweep fee to main address
		err, _ := fundSweepFee(floatAccount, txNetworkAsset, recipientAddress, userAddress.Network, cache, logger, config, serviceErr, addressTransactions, repository)
		if err != nil {
			return err
		}
		time.Sleep(time.Second * utility.FUND_SWEEP_FEE_WAIT_TIME)
	}

	sendSingleTransactionRequest := dto.SendSingleTransactionRequest{
		FromAddress: recipientAddress,
		ToAddress:   toAddress,
		Memo:        addressMemo,
		Amount:      big.NewInt(0),
		AssetSymbol: transactionListInfo.AssetSymbol,
		IsSweep:     true,
		ProcessType: utility.SWEEPPROCESS,
		Reference:   fmt.Sprintf("%s-%d", recipientAddress, time.Now().Unix()),
	}

	sendSingleTransactionResponse := dto.SendTransactionResponse{}
	if err := services.SendSingleTransaction(cache, logger, config, sendSingleTransactionRequest, &sendSingleTransactionResponse, &serviceErr); err != nil {
		logger.Error("Error response from SendSingleTransaction : %+v while sweeping for address with id %+v", err, recipientAddress)
		switch serviceErr.Code {
		case errorcode.INSUFFICIENT_FUNDS:
			if err := updateSweptStatus(addressTransactions, repository, logger); err != nil {
				return err
			}
			return nil
		default:
			return err
		}
	}

	if txNetworkAsset.CoinType == constants.TRX_COINTYPE {
		_ = incrementTRXSweepCount(repository, userAddress)
	}

	if err := updateSweptStatus(addressTransactions, repository, logger); err != nil {
		return err
	}
	return nil
}

func HasExceededTrxSweepLimit(userAddress model.UserAddress, logger *utility.Logger, assetSymbol string, repository database.BaseRepository) bool {
	if userAddress.NextSweepTime == nil {
		nextSweepTime := time.Now()
		userAddress.NextSweepTime = &nextSweepTime
	}
	if userAddress.SweepCount >= constants.DAILY_TRX_SWEEP_COUNT {
		logger.Error("Daily sweep limit exceeded for %s, postponing sweep to reset counter", assetSymbol)
		_ = ResetTRXSweepCount(repository, &userAddress)
		return true
	} else if userAddress.SweepCount == 0 && userAddress.NextSweepTime.UnixNano() > time.Now().UnixNano() {
		return true
	}
	return false
}

func ResetTRXSweepCount(repository database.BaseRepository, userAddress *model.UserAddress) error {

	userAddress.SweepCount = 0
	userAddress.NextSweepTime = utility.GetNextDayFromNow()

	if err := repository.Update(userAddress, model.UserAddress{SweepCount: userAddress.SweepCount, NextSweepTime: userAddress.NextSweepTime}); err != nil {
		return err
	}

	return nil
}

func incrementTRXSweepCount(repository database.BaseRepository, userAddress model.UserAddress) error {
	userAddress.SweepCount = userAddress.SweepCount + 1
	if err := repository.Update(&userAddress, userAddress); err != nil {
		return err
	}
	return nil
}

func CheckSweepMinimum(denomination model.Network, config Config.Data, sum float64, logger *utility.Logger) (bool, error) {
	if sum < denomination.MinimumSweepable {
		logger.Error("Error response from sweep job : Total sweep sum %v for asset (%s) is below the minimum sweep %v, so terminating sweep process", sum, denomination.AssetSymbol, denomination.MinimumSweepable)
		return false, utility.AppError{
			ErrType: utility.SWEEP_ERROR_INSUFFICIENT,
			Err:     errors.New(utility.SWEEP_ERROR_INSUFFICIENT),
		}
	}
	return true, nil
}

func getTransactionListInfo(repository database.BaseRepository, assetTransactions []model.Transaction, logger *utility.Logger) (dto.TransactionListInfo, error) {
	//need representative Asset to get common things about this list like symbol, Decimals etc
	var transactionListInfo = dto.TransactionListInfo{}

	recipientAsset := model.UserAsset{}
	//all the tx in assetTransactions have the same recipientId so get info from the 0th position
	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	if err := userAssetRepository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetTransactions[0].RecipientID}}, &recipientAsset); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
		return dto.TransactionListInfo{}, err
	}
	logger.Error("QA Log from Sweep job : now calling GetNetworkByAssetAndNetwork with network %+v and %+v while trying to getTransactionListInfo", transactionListInfo.Network, transactionListInfo.AssetSymbol)
	recipientNetworkAsset, err := services.GetNetworkByAssetAndNetwork(&userAssetRepository, transactionListInfo.Network, transactionListInfo.AssetSymbol)
	if err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
		return dto.TransactionListInfo{}, err
	}
	transactionListInfo.AssetSymbol = recipientAsset.AssetSymbol
	transactionListInfo.Decimal = recipientNetworkAsset.NativeDecimals
	transactionListInfo.UserId = recipientAsset.UserID
	transactionListInfo.AddressProvider = recipientNetworkAsset.AddressProvider
	transactionListInfo.Network = recipientNetworkAsset.Network
	return transactionListInfo, nil
}

func GroupTxByAddressByAssetSymbol(transactions []model.Transaction, repository database.BaseRepository, logger *utility.Logger) (map[string][]model.Transaction, error) {
	//loop over assetTransactions, get the chainTx and group by address
	//group transactions by addresses
	transactionsPerRecipientAddress := make(map[string][]model.Transaction)
	for _, tx := range transactions {
		logger.Info("GroupByTx - getting chain transaction for  %+v", tx.ID)
		chainTransaction := model.ChainTransaction{}
		if uuid.Nil != tx.OnChainTxId && tx.TransactionTag != "CREDIT" {
			e := getChainTransaction(repository, tx, &chainTransaction, logger)
			logger.Info("GroupByTx - chaintx is  %+v - %+v", chainTransaction.ID, chainTransaction.RecipientAddress)
			if e != nil {
				logger.Info("GroupByTx - getting chain transaction FAILED for  %+v", tx.ID)
				return nil, e
			}
		} else {
			logger.Info("GroupByTx - skipping getChainTransaction for Internal Deposit %s", tx.TransactionReference)
		}

		if chainTransaction.RecipientAddress != "" {
			key := chainTransaction.RecipientAddress + utility.SWEEP_GROUPING_SEPERATOR + tx.AssetSymbol
			transactionsPerRecipientAddress[key] = append(transactionsPerRecipientAddress[key], tx)
		} else {
			userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
			binanceAddress, err := services.GetBinanceProvidedAddressforAsset(&userAssetRepository, tx.RecipientID)
			if err != nil || binanceAddress == "" {
				//skip this asset
				logger.Info("GroupByTx - getting binance address FAILED for  %+v skipping this transaction", tx.ID)
				continue
			}
			key := binanceAddress + utility.SWEEP_GROUPING_SEPERATOR + tx.AssetSymbol
			transactionsPerRecipientAddress[key] = append(transactionsPerRecipientAddress[key], tx)
		}

	}

	return transactionsPerRecipientAddress, nil
}

func GroupTxByAssetSymbol(transactions []model.Transaction, repository database.BaseRepository, logger *utility.Logger) (map[string][]model.Transaction, map[string][]string, error) {
	//loop over assetTransactions, get the chainTx and group by symbol
	//group transactions by symbol
	transactionsPerSymbol := make(map[string][]model.Transaction)
	batchAddressesPerSymbol := make(map[string][]string)
	for _, tx := range transactions {
		logger.Info("GroupByTx - getting chain transaction for  %+v", tx.ID)
		chainTransaction := model.ChainTransaction{}
		if uuid.Nil != tx.OnChainTxId && tx.TransactionTag != "CREDIT" {
			e := getChainTransaction(repository, tx, &chainTransaction, logger)
			logger.Info("GroupByTx - chaintx is  %+v - %+v", chainTransaction.ID, chainTransaction.RecipientAddress)
			if e != nil {
				logger.Info("GroupByTx - getting chain transaction FAILED for  %+v", tx.ID)
				return nil, nil, e
			}

			key := tx.AssetSymbol
			transactionsPerSymbol[key] = append(transactionsPerSymbol[key], tx)
			batchAddressesPerSymbol[key] = append(batchAddressesPerSymbol[key], chainTransaction.RecipientAddress)

		} else {
			logger.Info("GroupByTx - skipping getChainTransaction for Internal Deposit %s", tx.TransactionReference)
		}
	}

	return transactionsPerSymbol, batchAddressesPerSymbol, nil
}

func getChainTransaction(repository database.BaseRepository, tx model.Transaction, chainTransaction *model.ChainTransaction, logger *utility.Logger) error {
	err := repository.Get(&model.ChainTransaction{BaseModel: model.BaseModel{ID: tx.OnChainTxId}}, chainTransaction)
	if err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v cant fetch chainTransaction for depsoit tx",
			err)
		return err
	}
	return nil
}

// RemoveBatchTransactions returns the elements in `a` that aren't in `b`.
func RemoveBatchTransactions(a []model.Transaction, b []model.Transaction) []model.Transaction {

	mb := make(map[uuid.UUID]struct{}, len(b))
	for _, x := range b {
		mb[x.ID] = struct{}{}
	}
	var diff []model.Transaction
	for _, x := range a {
		if _, found := mb[x.ID]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

func ToUniqueAddresses(addresses []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, entry := range addresses {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func fundSweepFee(floatAccount model.HotWalletAsset, txNetworkAsset model.Network, recipientAddress, network string, cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, serviceErr dto.ServicesRequestErr, assetTransactions []model.Transaction, repository database.BaseRepository) (error, bool) {

	request := dto.OnchainBalanceRequest{
		AssetSymbol: txNetworkAsset.AssetSymbol,
		Network: network,
		Address:     recipientAddress,
	}
	mainCoinOnChainBalanceResponse := dto.OnchainBalanceResponse{}
	if err := services.GetOnchainBalance(cache, logger, config, request, &mainCoinOnChainBalanceResponse, serviceErr); err != nil {
		logger.Error("Error response from Sweep job : %+v while getting on-chain balance for %+v", err, recipientAddress)
		return err, true
	}
	mainCoinOnChainBalance, _ := strconv.ParseUint(mainCoinOnChainBalanceResponse.Balance, 10, 64)
	//check if onchain balance in main coin asset is less than floatAccount.SweepFee
	if int64(mainCoinOnChainBalance) < txNetworkAsset.SweepFee {

		sendSingleTransactionRequest := dto.SendSingleTransactionRequest{
			FromAddress: floatAccount.Address,
			ToAddress:   recipientAddress,
			Amount:      big.NewInt(txNetworkAsset.SweepFee),
			AssetSymbol: txNetworkAsset.AssetSymbol,
			//this currently only supports coins that supports Memo,  other coins will be ignored
			Memo:        utility.SWEEPMEMOBNB,
			ProcessType: utility.FLOATPROCESS,
			Reference:   fmt.Sprintf("%s-%s", txNetworkAsset.AssetSymbol, assetTransactions[0].TransactionReference),
		}
		sendSingleTransactionResponse := dto.SendTransactionResponse{}
		if err := services.SendSingleTransaction(cache, logger, config, sendSingleTransactionRequest, &sendSingleTransactionResponse, &serviceErr); err != nil {
			logger.Error("Error response from Sweep job : %+v while funding sweep fee for  %+v", err, recipientAddress)
			return err, true
		}
		//return immediately after broadcasting sweep fee, this allows for confirmation, next time sweep runs,
		// int64(mainCoinOnChainBalance) will be > floatAccount.SweepFee, and so this if block will be skipped
		//i.e sweep fee will not be resent to user address
		return nil, true
	}
	return nil, false
}

func GetSweepParams(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, floatAccount model.HotWalletAsset, txNetworkAsset model.Network, sweepFund float64) (SweepParam, error) {

	sweepParam := SweepParam{}
	serviceErr := dto.ServicesRequestErr{}

	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	totalUsersBalance, err := GetTotalUserBalance(repository, floatAccount.AssetSymbol, txNetworkAsset.NativeDecimals,  logger, userAssetRepository)
	if err != nil {
		return sweepParam, err
	}
	logger.Info("SWEEP_OPERATION : Total users balance for this hot wallet %+v is %+v and total amount to sweep is %+v", floatAccount.AssetSymbol, totalUsersBalance, sweepFund)

	// Get float chain balance
	prec := uint(64)
	onchainBalanceRequest := dto.OnchainBalanceRequest{
		AssetSymbol: floatAccount.AssetSymbol,
		Network: floatAccount.Network,
		Address:     floatAccount.Address,
	}
	floatOnChainBalanceResponse := dto.OnchainBalanceResponse{}
	if err := services.GetOnchainBalance(cache, logger, config, onchainBalanceRequest, &floatOnChainBalanceResponse, serviceErr); err != nil {
		logger.Error("Error response from Sweep job : %+v while getting float on-chain balance for %+v", err, floatAccount.AssetSymbol)
		return sweepParam, err
	}
	floatOnChainBalance, _ := new(big.Float).SetPrec(prec).SetString(floatOnChainBalanceResponse.Balance)
	logger.Info("SWEEP_OPERATION : Float on-chain balance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, floatOnChainBalance)

	// Get float manager parameters to calculate float range
	floatManagerParams, err := getFloatParamFor(floatAccount.AssetSymbol, floatAccount.Network, repository, logger)
	if err != nil {
		return sweepParam, err
	}
	minimumFloatBalance, maximumFloatBalance := GetFloatBalanceRange(floatManagerParams, totalUsersBalance, logger)

	// Get total deposit sum from the last run of this job
	depositSumFromLastRun, err := getDepositsSumForAssetFromDate(repository, floatAccount.AssetSymbol, floatAccount.Network, logger, floatAccount)
	if err != nil {
		logger.Info("error with float manager process, while trying to get the total deposit sum from last run : %+v", err)
		return sweepParam, err
	}
	logger.Info("depositSumFromLastRun for this hot wallet (%s) is %+v", floatAccount.AssetSymbol, depositSumFromLastRun)

	// Get total withdrawal sum from the last run of this job
	withdrawalSumFromLastRun, err := getWithdrawalsSumForAssetFromDate(repository, floatAccount.AssetSymbol, floatAccount.Network, logger, floatAccount)
	if err != nil {
		logger.Info("error with float manager process, while trying to get the total withdrawal sum from last run : %+v", err)
		return sweepParam, err
	}
	logger.Info("withdrawalSumFromLastRun for this hot wallet %+v is %+v", floatAccount.AssetSymbol, withdrawalSumFromLastRun)

	floatDeficit := GetFloatDeficit(depositSumFromLastRun, withdrawalSumFromLastRun, minimumFloatBalance, maximumFloatBalance, floatOnChainBalance, logger)

	brokerageAccountResponse, err := GetBrokerAccountFor(floatAccount.AssetSymbol, txNetworkAsset, repository, cache, config, logger, serviceErr)
	if err != nil {
		return sweepParam, err
	}

	floatPercent, brokeragePercent := GetSweepPercentages(floatOnChainBalance, minimumFloatBalance, floatDeficit, big.NewFloat(sweepFund), totalUsersBalance, floatManagerParams, logger)

	sweepParam = SweepParam{
		FloatAddress:     floatAccount.Address,
		FloatPercent:     floatPercent,
		BrokerageAddress: brokerageAccountResponse.Address,
		BrokeragePercent: brokeragePercent,
	}

	return sweepParam, nil
}

func GetSweepPercentages(floatOnChainBalance, minimumFloatBalance, floatDeficit, sweepFund, totalUsersBalance *big.Float, floatManagerParams model.FloatManagerParam, logger *utility.Logger) (int64, int64) {

	var floatPercent, brokeragePercent int64

	if floatOnChainBalance.Cmp(minimumFloatBalance) <= 0 {
		if floatDeficit.Cmp(sweepFund) > 0 {
			floatDeficit = sweepFund
		}
		floatPercent = GeTFloatPercent(floatDeficit, sweepFund).Int64()

		logger.Info("SWEEP_OPERATION : FloatOnChainBalance for this hot wallet %+v is %+v, this is below the minimum %+v of total user balance %v which is %+v, moving %v percent of sweep funds %+v to float account ",
			floatManagerParams.AssetSymbol, floatOnChainBalance, floatManagerParams.MinPercentTotalUserBalance, totalUsersBalance, minimumFloatBalance, floatPercent, sweepFund)
	}

	brokeragePercent = int64(100) - floatPercent
	logger.Info("SWEEP_OPERATION : Moving %+v of sweep funds %+v to brokerage account for this hot wallet %+v ", brokeragePercent, sweepFund, floatManagerParams.AssetSymbol)

	return floatPercent, brokeragePercent
}

func GetBrokerAccountFor(assetSymbol string, txNetworkAsset model.Network, repository database.BaseRepository, cache *utility.MemoryCache, config Config.Data, logger *utility.Logger, serviceErr dto.ServicesRequestErr) (dto.DepositAddressResponse, error) {

	brokerageAccountResponse := dto.DepositAddressResponse{}
	denomination := model.Denomination{}
	err := repository.GetByFieldName(&model.Denomination{AssetSymbol: assetSymbol}, &denomination)
	if err != nil {
		return brokerageAccountResponse, err
	}

	if *txNetworkAsset.IsToken {
		err = services.GetDepositAddress(cache, logger, config, assetSymbol, txNetworkAsset.AssetSymbol, &brokerageAccountResponse, serviceErr)
	} else {
		err = services.GetDepositAddress(cache, logger, config, assetSymbol, "", &brokerageAccountResponse, serviceErr)
	}
	if err != nil {
		return brokerageAccountResponse, err
	}

	logger.Info("SWEEP_OPERATION : Brokerage account for this hot wallet %+v is %+v", assetSymbol, brokerageAccountResponse)
	return brokerageAccountResponse, nil
}

func GetFloatDeficit(depositSumFromLastRun, withdrawalSumFromLastRun, minimumBalance, maximumBalance, onChainBalance *big.Float, logger *utility.Logger) *big.Float {

	deficit := new(big.Float)

	if depositSumFromLastRun.Cmp(withdrawalSumFromLastRun) < 0 {
		// if total deposit is less than total withdrawal, use maximum
		deficit.Sub(maximumBalance, onChainBalance)
		logger.Info("SWEEP_OPERATION : depositSumFromLastRun <=  withdrawalSumFromLastRun, using valueOfMaximumFloatPercent to calculate float deficit")
	} else {
		// if total deposit is greater than total withdrawal, use minimum
		deficit.Sub(minimumBalance, onChainBalance)
		logger.Info("SWEEP_OPERATION : depositSumFromLastRun > withdrawalSumFromLastRun, using valueOfMinimumFloatPercent to calculate float deficit")
	}

	return deficit
}

func GetFloatBalanceRange(floatManagerParams model.FloatManagerParam, totalUsersBalance *big.Float, logger *utility.Logger) (*big.Float, *big.Float) {

	valueOfMinimumFloatPercent := new(big.Float)
	valueOfMaximumFloatPercent := new(big.Float)
	valueOfMinimumFloatPercent.Mul(big.NewFloat(floatManagerParams.MinPercentTotalUserBalance), totalUsersBalance)
	valueOfMaximumFloatPercent.Mul(big.NewFloat(floatManagerParams.MaxPercentTotalUserBalance), totalUsersBalance)
	logger.Info("SWEEP_OPERATION : valueOfMinimumFloatPercent and valueOfMaximumFloatPercent for this hot wallet %+v is %+v and %+v", floatManagerParams.AssetSymbol, valueOfMinimumFloatPercent, valueOfMaximumFloatPercent)

	return valueOfMinimumFloatPercent, valueOfMaximumFloatPercent
}

func GeTFloatPercent(accountDeficit, sweepFund *big.Float) *big.Int {

	deficit := new(big.Float)
	floatPercent := new(big.Float)
	floatPercentInInt := new(big.Int)
	deficit.Mul(accountDeficit, big.NewFloat(100))
	floatPercent.Quo(deficit, sweepFund)
	floatPercent.Int(floatPercentInInt)
	return floatPercentInInt
}

func GetSweepAddressAndMemo(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, floatAccount model.HotWalletAsset, txNetworkAsset model.Network) (string, string, error) {

	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	totalUsersBalance, err := GetTotalUserBalance(repository, floatAccount.AssetSymbol, txNetworkAsset.NativeDecimals, logger, userAssetRepository)
	if err != nil {
		return "", "", err
	}
	logger.Info("SWEEP_OPERATION : Total users balance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, totalUsersBalance)

	// Get float chain balance
	prec := uint(64)
	serviceErr := dto.ServicesRequestErr{}
	onchainBalanceRequest := dto.OnchainBalanceRequest{
		AssetSymbol: floatAccount.AssetSymbol,
		Network: floatAccount.Network,
		Address:     floatAccount.Address,
	}
	floatOnChainBalanceResponse := dto.OnchainBalanceResponse{}
	if err := services.GetOnchainBalance(cache, logger, config, onchainBalanceRequest, &floatOnChainBalanceResponse, serviceErr); err != nil {
		logger.Error("SWEEP_OPERATION, err : %+v while getting float on-chain balance for %+v", err, floatAccount.AssetSymbol)
		return "", "", err
	}
	floatOnChainBalance, _ := new(big.Float).SetPrec(prec).SetString(floatOnChainBalanceResponse.Balance)
	logger.Info("SWEEP_OPERATION : Float on-chain balance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, floatOnChainBalance)

	// Get broker account
	brokerageAccountResponse := dto.DepositAddressResponse{}

	if *txNetworkAsset.IsToken {
		err = services.GetDepositAddress(cache, logger, config, floatAccount.AssetSymbol, txNetworkAsset.AssetSymbol, &brokerageAccountResponse, serviceErr)
	} else {
		err = services.GetDepositAddress(cache, logger, config, floatAccount.AssetSymbol, "", &brokerageAccountResponse, serviceErr)
	}
	if err != nil {
		return "", "", err
	}
	logger.Info("SWEEP_OPERATION : Brokerage account for this hot wallet %+v is %+v", floatAccount.AssetSymbol, brokerageAccountResponse)

	// Get float manager parameters to calculate minimum float
	floatManagerParams, err := getFloatParamFor(floatAccount.AssetSymbol, floatAccount.Network, repository, logger)
	if err != nil {
		return "", "", err
	}
	valueOfMinimumFloatPercent := new(big.Float)
	valueOfMinimumFloatPercent.Mul(big.NewFloat(floatManagerParams.MinPercentTotalUserBalance), totalUsersBalance)

	if floatOnChainBalance.Cmp(valueOfMinimumFloatPercent) <= 0 {
		logger.Info("SWEEP_OPERATION : FloatOnChainBalance for this hot wallet %+v is %+v, this is below %v of total user balance %v, moving sweep funds to float account ",
			floatAccount.AssetSymbol, floatOnChainBalance, floatManagerParams.MinPercentTotalUserBalance, totalUsersBalance)
		return floatAccount.Address, "", err
	}
	logger.Info("SWEEP_OPERATION : FloatOnChainBalance for this hot wallet %+v is %+v, this is above %v of total user balance %v, moving sweep funds to brokerage ",
		floatAccount.AssetSymbol, floatOnChainBalance, floatManagerParams.MinPercentTotalUserBalance, totalUsersBalance)

	return brokerageAccountResponse.Address, brokerageAccountResponse.Tag, nil
}

func getFloatDetails(repository database.BaseRepository, symbol, network string, logger *utility.Logger) (model.HotWalletAsset, error) {
	//Get the float address
	var floatAccount model.HotWalletAsset
	if err := repository.Get(&model.HotWalletAsset{AssetSymbol: symbol, Network: network}, &floatAccount); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id and trying to get float detials", err)
		return model.HotWalletAsset{}, err
	}
	return floatAccount, nil
}

func updateSweptStatus(assetTransactions []model.Transaction, repository database.BaseRepository, logger *utility.Logger) error {
	//update all assetTransactions with new swept status
	var assetIdList []uuid.UUID
	for _, tx := range assetTransactions {
		assetIdList = append(assetIdList, tx.ID)
	}
	if err := repository.BulkUpdateTransactionSweptStatus(assetIdList); err != nil {
		logger.Error("Error response from Sweep job : %+v while updating swept status "+utility.UPDATE_SWEPT_STATUS_FAILURE, err)
		return err
	}
	logger.Info(utility.UPDATE_SWEPT_STATUS_SUCCESS)
	return nil
}

func AcquireLock(identifier string, ttl int64, cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, serviceErr dto.ServicesRequestErr) (string, error) {
	// It calls the lock service to obtain a lock for the transaction
	lockerServiceRequest := dto.LockerServiceRequest{
		Identifier:   fmt.Sprintf("%s%s", config.LockerPrefix, identifier),
		ExpiresAfter: ttl,
	}
	lockerServiceResponse := dto.LockerServiceResponse{}
	if err := services.AcquireLock(cache, logger, config, lockerServiceRequest, &lockerServiceResponse, &serviceErr); err != nil {
		logger.Error("Error occured while obtaining lock : %+v; %s", serviceErr, err)
		if !serviceErr.Success && serviceErr.Message != "" {
			return "", errors.New(serviceErr.Message)
		}
		return "", err
	}
	return lockerServiceResponse.Token, nil
}

func releaseLock(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, lockerServiceToken string, serviceErr dto.ServicesRequestErr) error {
	lockReleaseRequest := dto.LockReleaseRequest{
		Identifier: fmt.Sprintf("%s%s", config.LockerPrefix, "sweep"),
		Token:      lockerServiceToken,
	}
	lockReleaseResponse := dto.ServicesRequestSuccess{}
	if err := services.ReleaseLock(cache, logger, config, lockReleaseRequest, &lockReleaseResponse, &serviceErr); err != nil {
		if serviceErr.Code != "" {
			return errors.New(serviceErr.Message)
		}
		return err
	}
	return nil
}

func ExecuteSweepCronJob(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository) {
	c := cron.New()
	c.AddFunc(config.SweepCronInterval, func() { SweepTransactions(cache, logger, config, repository) })
	c.Start()
}