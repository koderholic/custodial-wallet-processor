package tasks

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"

	"github.com/robfig/cron/v3"
	uuid "github.com/satori/go.uuid"
)

// BTCSweepParam ... Model definition for BTC sweep
type (
	BTCSweepParam struct {
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

	logger.Info("Fetched %d sweep candidates", len(transactions))

	var btcAddresses []string
	var btcAssetTransactionsToSweep []model.Transaction
	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	for _, tx := range transactions {
		//Filter BTC assets, save in a seperate list for batch processing and skip individual processing
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
		if recipientAsset.AssetSymbol == utility.COIN_BTC {
			//get recipient address for transaction
			chainTransaction := model.ChainTransaction{}
			_ = getChainTransaction(repository, tx, chainTransaction, logger)
			btcAddresses = append(btcAddresses, chainTransaction.RecipientAddress)
			btcAssetTransactionsToSweep = append(btcAssetTransactionsToSweep, tx)
			//skip futher processing for this asset, will be included a part of batch btc processing
			continue
		}
	}
	btcAddresses = toUniqueAddresses(btcAddresses)
	//remove btc transactions from list of remaining transactions
	transactions = RemoveBTCTransactions(transactions, btcAssetTransactionsToSweep)

	transactionsPerAddress, err := GroupTxByAddress(transactions, repository, logger)
	if err != nil {
		return
	}
	for address, addressTransactions := range transactionsPerAddress {
		sum := calculateSum(repository, addressTransactions, logger)
		logger.Info("Sweeping %s with total of %d", address, sum)
		if err := sweepPerAddress(cache, logger, config, repository, serviceErr, addressTransactions, sum, address); err != nil {
			continue
		}
	}
	//batch process btc
	if len(btcAddresses) > 0 {
		if err := sweepBatchTx(cache, logger, config, repository, serviceErr, btcAddresses, btcAssetTransactionsToSweep); err != nil {
			logger.Error("Error response from Sweep job : %+v while sweeping batch transactions for BTC", err)
			if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
				logger.Error("Could not release lock", err)
				return
			}
			return
		}
	}
	if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
		logger.Error("Could not release lock", err)
		return
	}
	logger.Info("Sweep operation ends successfully, lock released")
}

func calculateSum(repository database.BaseRepository, addressTransactions []model.Transaction, logger *utility.Logger) int64 {
	recipientAsset, _ := getRepresentativeAssetForOthers(repository, addressTransactions, logger)
	//Get total sum to be swept for this assetId address
	var sum = int64(0)
	for _, tx := range addressTransactions {
		//convert to native units
		balance, _ := strconv.ParseFloat(tx.Value, 64)
		//choose 1st of the address transaction, would have
		// the same denominationDecimal as the rest
		denominationDecimal := float64(recipientAsset.Decimal)
		scaledBalance := int64(balance * math.Pow(10, denominationDecimal))
		sum = sum + scaledBalance
	}
	return sum
}

func CalculateSumOfBtcBatch(addressTransactions []model.Transaction) float64 {
	//Get total sum to be swept for this batch
	var sum = float64(0)
	for _, tx := range addressTransactions {
		value, _ := strconv.ParseFloat(tx.Value, 64)
		sum += value
	}
	return sum
}

func sweepBatchTx(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, serviceErr dto.ServicesRequestErr, btcAddresses []string, btcAssetTransactionsToSweep []model.Transaction) error {
	// Calls key-management to batch sign transaction
	recipientData := []dto.BatchRecipients{}
	//get float
	floatAccount, err := getFloatDetails(repository, "BTC", logger)
	if err != nil {
		return err
	}

	//check total sum threshold for this batch
	totalSweepSum := CalculateSumOfBtcBatch(btcAssetTransactionsToSweep)
	if totalSweepSum < config.BTC_minimumSweep {
		logger.Error("Error response from sweep job : Total sweep sum %v for asset (%s) is below the minimum sweep %v, so terminating sweep process", totalSweepSum, utility.COIN_BTC, config.BTC_minimumSweep, err)
		return err
	}

	sweepParam, err := GetSweepParams(cache, logger, config, repository, floatAccount, totalSweepSum)
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
	signBatchTransactionAndBroadcastRequest := dto.BatchBTCRequest{
		AssetSymbol:   "BTC",
		ChangeAddress: sweepParam.BrokerageAddress,
		IsSweep:       true,
		Origins:       btcAddresses,
		Recipients:    recipientData,
		ProcessType:   utility.SWEEPPROCESS,
	}
	signBatchTransactionAndBroadcastResponse := dto.SignAndBroadcastResponse{}
	if err := services.SignBatchTransactionAndBroadcast(nil, cache, logger, config, signBatchTransactionAndBroadcastRequest, &signBatchTransactionAndBroadcastResponse, serviceErr); err != nil {
		logger.Error("Error response from SignBatchTransactionAndBroadcast : %+v while sweeping batch transactions for BTC", err)
		return err
	}
	if err := updateSweptStatus(btcAssetTransactionsToSweep, repository, logger); err != nil {
		return err
	}
	return nil

}

func sweepPerAddress(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, serviceErr dto.ServicesRequestErr, addressTransactions []model.Transaction, sum int64, recipientAddress string) error {
	recipientAsset, e := getRepresentativeAssetForOthers(repository, addressTransactions, logger)
	if e != nil {
		return e
	}
	floatAccount, err := getFloatDetails(repository, recipientAsset.AssetSymbol, logger)
	if err != nil {
		return err
	}
	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: floatAccount.AssetSymbol, IsEnabled: true}, &denomination); err != nil {
		logger.Error("Error response from sweep job : %+v while trying to denomination of float asset", err)
		return err
	}

	toAddress, addressMemo, err := GetSweepAddressAndMemo(cache, logger, config, repository, floatAccount)
	if err != nil {
		logger.Error("Error response from Sweep job : %+v while getting sweep toAddress and memo for %s", err, floatAccount.AssetSymbol)
		return err
	}

	//Check that sweep amount is not below the minimum sweep amount
	var minimumSweep float64
	switch denomination.AssetSymbol {
	case utility.COIN_ETH:
		minimumSweep = config.ETH_minimumSweep
	case utility.COIN_BNB:
		minimumSweep = config.BNB_minimumSweep
	case utility.COIN_BUSD:
		minimumSweep = config.BUSD_minimumSweep
	}

	if float64(sum) < minimumSweep {
		logger.Error("Error response from sweep job : Total sweep sum %v for asset (%s) is below the minimum sweep %v, so terminating sweep process", sum, denomination.AssetSymbol, config.BTC_minimumSweep, err)
		return err
	}

	// Calls key-management to sign transaction
	signTransactionRequest := dto.SignTransactionRequest{
		FromAddress: recipientAddress,
		ToAddress:   toAddress,
		Memo:        addressMemo,
		Amount:      big.NewInt(0),
		AssetSymbol: recipientAsset.AssetSymbol,
		IsSweep:     true,
		ProcessType: utility.SWEEPPROCESS,
	}
	SignTransactionAndBroadcastResponse := dto.SignAndBroadcastResponse{}
	if err := services.SignTransactionAndBroadcast(cache, logger, config, signTransactionRequest, &SignTransactionAndBroadcastResponse, serviceErr); err != nil {
		logger.Error("Error response from SignTransactionAndBroadcast : %+v while sweeping for address with id %+v", err, recipientAddress)
		return err
	}
	if err := updateSweptStatus(addressTransactions, repository, logger); err != nil {
		return err
	}
	return nil
}

func getRepresentativeAssetForOthers(repository database.BaseRepository, assetTransactions []model.Transaction, logger *utility.Logger) (model.UserAsset, error) {
	//need representative Asset to get common things about this list like symbol, Decimals etc
	recipientAsset := model.UserAsset{}
	//all the tx in assetTransactions have the same recipientId so just pass the 0th position
	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	if err := userAssetRepository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetTransactions[0].RecipientID}}, &recipientAsset); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
		return model.UserAsset{}, err
	}
	return recipientAsset, nil
}

func GroupTxByAddress(transactions []model.Transaction, repository database.BaseRepository, logger *utility.Logger) (map[string][]model.Transaction, error) {
	//loop over assetTransactions, get the chainTx and group by address
	//group transactions by addresses
	transactionsPerRecipientAddress := make(map[string][]model.Transaction)
	for _, tx := range transactions {
		chainTransaction := model.ChainTransaction{}
		e := getChainTransaction(repository, tx, chainTransaction, logger)
		if e != nil {
			return nil, e
		}
		if chainTransaction.RecipientAddress != "" {
			transactionsPerRecipientAddress[chainTransaction.RecipientAddress] = append(transactionsPerRecipientAddress[chainTransaction.RecipientAddress], tx)
		}

	}
	return transactionsPerRecipientAddress, nil
}

func getChainTransaction(repository database.BaseRepository, tx model.Transaction, chainTransaction model.ChainTransaction, logger *utility.Logger) error {
	err := repository.Get(&model.ChainTransaction{BaseModel: model.BaseModel{ID: tx.OnChainTxId}}, &chainTransaction)
	if err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v cant fetch chainTransaction for depsoit tx",
			err)
		return err
	}
	return nil
}

// RemoveBTCTransactions returns the elements in `a` that aren't in `b`.
func RemoveBTCTransactions(a []model.Transaction, b []model.Transaction) []model.Transaction {

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

func toUniqueAddresses(addresses []string) []string {
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

func fundSweepFee(floatAccount model.HotWalletAsset, denomination model.Denomination, recipientAddress string, cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, serviceErr dto.ServicesRequestErr, recipientAsset model.UserAsset, assetTransactions []model.Transaction, repository database.BaseRepository) (error, bool) {

	request := dto.OnchainBalanceRequest{
		AssetSymbol: denomination.MainCoinAssetSymbol,
		Address:     recipientAddress,
	}
	mainCoinOnChainBalanceResponse := dto.OnchainBalanceResponse{}
	services.GetOnchainBalance(cache, logger, config, request, &mainCoinOnChainBalanceResponse, serviceErr)
	mainCoinOnChainBalance, _ := strconv.ParseUint(mainCoinOnChainBalanceResponse.Balance, 10, 64)
	//check if onchain balance in main coin asset is less than floatAccount.SweepFee
	if int64(mainCoinOnChainBalance) < denomination.SweepFee {
		// Calls key-management to sign sweep fee transaction
		signTransactionAndBroadcastRequest := dto.SignTransactionRequest{
			FromAddress: floatAccount.Address,
			ToAddress:   recipientAddress,
			Amount:      big.NewInt(denomination.SweepFee),
			AssetSymbol: denomination.MainCoinAssetSymbol,
			//this currently only supports coins that supports Memo, ETH will not be ignored
			Memo:        utility.SWEEPMEMOBNB,
			ProcessType: utility.SWEEPPROCESS,
		}
		signTransactionAndBroadcastResponse := dto.SignAndBroadcastResponse{}
		if err := services.SignTransactionAndBroadcast(cache, logger, config, signTransactionAndBroadcastRequest, &signTransactionAndBroadcastResponse, serviceErr); err != nil {
			logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
			return err, true
		}
		//return immediately after broadcasting sweep fee, this allows for confirmation, next time sweep runs,
		// int64(mainCoinOnChainBalance) will be > floatAccount.SweepFee, and so this if block will be skipped
		//i.e sweep fee will not be resent to user address
		return nil, true
	}
	// else? i.e. if mainCoinOnChainBalance > floatAccount.SweepFee, then we want to proceed like the general case for non token sweep
	return nil, false
}

func GetSweepParams(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, floatAccount model.HotWalletAsset, sweepFund float64) (BTCSweepParam, error) {

	sweepParam := BTCSweepParam{}
	serviceErr := dto.ServicesRequestErr{}

	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	totalUsersBalance, err := GetTotalUserBalance(repository, floatAccount.AssetSymbol, logger, userAssetRepository)
	if err != nil {
		return sweepParam, err
	}
	logger.Info("SWEEP_OPERATION : Total users balance for this hot wallet %+v is %+v and total amount to sweep is %+v", floatAccount.AssetSymbol, totalUsersBalance, sweepFund)

	// Get float chain balance
	prec := uint(64)
	onchainBalanceRequest := dto.OnchainBalanceRequest{
		AssetSymbol: floatAccount.AssetSymbol,
		Address:     floatAccount.Address,
	}
	floatOnChainBalanceResponse := dto.OnchainBalanceResponse{}
	services.GetOnchainBalance(cache, logger, config, onchainBalanceRequest, &floatOnChainBalanceResponse, serviceErr)
	floatOnChainBalance, _ := new(big.Float).SetPrec(prec).SetString(floatOnChainBalanceResponse.Balance)
	logger.Info("SWEEP_OPERATION : Float on-chain balance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, floatOnChainBalance)

	// Get float manager parameters to calculate float range
	floatManagerParams, err := getFloatParamFor(floatAccount.AssetSymbol, repository, logger)
	if err != nil {
		return sweepParam, err
	}
	minimumFloatBalance, maximumFloatBalance := GetFloatBalanceRange(floatManagerParams, totalUsersBalance, logger)

	// Get total deposit sum from the last run of this job
	depositSumFromLastRun, err := getDepositsSumForAssetFromDate(repository, floatAccount.AssetSymbol, logger, floatAccount)
	if err != nil {
		logger.Info("error with float manager process, while trying to get the total deposit sum from last run : %+v", err)
		return sweepParam, err
	}
	logger.Info("depositSumFromLastRun for this hot wallet (%s) is %+v", floatAccount.AssetSymbol, depositSumFromLastRun)

	// Get total withdrawal sum from the last run of this job
	withdrawalSumFromLastRun, err := getWithdrawalsSumForAssetFromDate(repository, floatAccount.AssetSymbol, logger, floatAccount)
	if err != nil {
		logger.Info("error with float manager process, while trying to get the total withdrawal sum from last run : %+v", err)
		return sweepParam, err
	}
	logger.Info("withdrawalSumFromLastRun for this hot wallet %+v is %+v", floatAccount.AssetSymbol, withdrawalSumFromLastRun)

	floatDeficit := GetFloatDeficit(depositSumFromLastRun, withdrawalSumFromLastRun, minimumFloatBalance, maximumFloatBalance, floatOnChainBalance, logger)

	brokerageAccountResponse, err := GetBrokerAccountFor(floatAccount.AssetSymbol, repository, cache, config, logger, serviceErr)
	if err != nil {
		return sweepParam, err
	}

	floatPercent, brokeragePercent := GetSweepPercentages(floatOnChainBalance, minimumFloatBalance, floatDeficit, big.NewFloat(sweepFund), totalUsersBalance, floatManagerParams, logger)

	sweepParam = BTCSweepParam{
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

func GetBrokerAccountFor(assetSymbol string, repository database.BaseRepository, cache *utility.MemoryCache, config Config.Data, logger *utility.Logger, serviceErr dto.ServicesRequestErr) (dto.DepositAddressResponse, error) {

	brokerageAccountResponse := dto.DepositAddressResponse{}
	denomination := model.Denomination{}
	err := repository.GetByFieldName(&model.Denomination{AssetSymbol: assetSymbol, IsEnabled: true}, &denomination)
	if err != nil {
		return brokerageAccountResponse, err
	}

	if denomination.IsToken {
		err = services.GetDepositAddress(cache, logger, config, assetSymbol, denomination.MainCoinAssetSymbol, &brokerageAccountResponse, serviceErr)
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

func GetSweepAddressAndMemo(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, floatAccount model.HotWalletAsset) (string, string, error) {

	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	totalUsersBalance, err := GetTotalUserBalance(repository, floatAccount.AssetSymbol, logger, userAssetRepository)
	if err != nil {
		return "", "", err
	}
	logger.Info("SWEEP_OPERATION : Total users balance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, totalUsersBalance)

	// Get float chain balance
	prec := uint(64)
	serviceErr := dto.ServicesRequestErr{}
	onchainBalanceRequest := dto.OnchainBalanceRequest{
		AssetSymbol: floatAccount.AssetSymbol,
		Address:     floatAccount.Address,
	}
	floatOnChainBalanceResponse := dto.OnchainBalanceResponse{}
	services.GetOnchainBalance(cache, logger, config, onchainBalanceRequest, &floatOnChainBalanceResponse, serviceErr)
	floatOnChainBalance, _ := new(big.Float).SetPrec(prec).SetString(floatOnChainBalanceResponse.Balance)
	logger.Info("SWEEP_OPERATION : Float on-chain balance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, floatOnChainBalance)

	// Get broker account
	brokerageAccountResponse := dto.DepositAddressResponse{}
	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: floatAccount.AssetSymbol, IsEnabled: true}, &denomination); err != nil {
		return "", "", err
	}

	if denomination.IsToken {
		err = services.GetDepositAddress(cache, logger, config, floatAccount.AssetSymbol, denomination.MainCoinAssetSymbol, &brokerageAccountResponse, serviceErr)
	} else {
		err = services.GetDepositAddress(cache, logger, config, floatAccount.AssetSymbol, "", &brokerageAccountResponse, serviceErr)
	}
	if err != nil {
		return "", "", err
	}
	logger.Info("SWEEP_OPERATION : Brokerage account for this hot wallet %+v is %+v", floatAccount.AssetSymbol, brokerageAccountResponse)

	// Get float manager parameters to calculate minimum float
	floatManagerParams, err := getFloatParamFor(floatAccount.AssetSymbol, repository, logger)
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

func getFloatDetails(repository database.BaseRepository, symbol string, logger *utility.Logger) (model.HotWalletAsset, error) {
	//Get the float address
	var floatAccount model.HotWalletAsset
	if err := repository.Get(&model.HotWalletAsset{AssetSymbol: symbol}, &floatAccount); err != nil {
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
		logger.Error("Error response from Sweep job : %+v while broadcasting to chain", err)
		return err
	}
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
