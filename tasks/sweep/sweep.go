package sweep

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
	"wallet-adapter/tasks"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/constants"
	"wallet-adapter/utility/errorcode"
	"wallet-adapter/utility/logger"

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

func SweepTransactions(cache *cache.Memory, config Config.Data, repository database.IRepository) {
	logger.Info("Sweep operation begins")
	userAssetRepository := database.UserAssetRepository{BaseRepository: database.BaseRepository{Database: database.Database{config, repository.Db()}}}

	serviceErr := dto.ExternalServicesRequestErr{}
	token, err := tasks.AcquireLock(&userAssetRepository, "sweep", constants.SIX_HUNDRED_MILLISECONDS, cache, config, serviceErr)
	if err != nil {
		logger.Error("Could not acquire lock", err)
		return
	}

	var transactions []model.Transaction
	if err := repository.FetchSweepCandidates(&transactions); err != nil {
		logger.Error("Error response from Sweep job : could not fetch sweep candidates %+v", err)
		if err := tasks.ReleaseLock(&userAssetRepository, cache, config, token, serviceErr); err != nil {
			logger.Error("Could not release lock", err)
			return
		}
		return
	}

	logger.Info("Fetched %d sweep candidates", len(transactions))

	var btcAddresses []string
	var btcAssetTransactionsToSweep []model.Transaction
	for _, tx := range transactions {
		//Filter BTC assets, save in a seperate list for batch pro cessing and skip individual processing
		//need recipient Asset to check assetSymbol
		recipientAsset := model.UserAsset{}
		//all the tx in assetTransactions have the same recipientId so just pass the 0th position
		if err := userAssetRepository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: tx.RecipientID}}, &recipientAsset); err != nil {
			logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
			if err := tasks.ReleaseLock(&userAssetRepository, cache, config, token, serviceErr); err != nil {
				logger.Error("Could not release lock", err)
				return
			}
			return
		}
		if recipientAsset.AssetSymbol == constants.COIN_BTC {
			//get recipient address for transaction
			chainTransaction := model.ChainTransaction{}
			err = getChainTransaction(repository, tx, &chainTransaction)
			if err != nil {
				logger.Error("Error response from Sweep job, could not get chain transaction :"+
					" %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
				continue

			}
			btcAddresses = append(btcAddresses, chainTransaction.RecipientAddress)
			btcAssetTransactionsToSweep = append(btcAssetTransactionsToSweep, tx)
			//skip futher processing for this asset, will be included a part of batch btc processing
			continue
		}
	}
	btcAddresses = ToUniqueAddresses(btcAddresses)
	//remove btc transactions from list of remaining transactions
	transactions = RemoveBTCTransactions(transactions, btcAssetTransactionsToSweep)
	//Do other Coins apart from BTC
	transactionsPerAddressPerAssetSymbol, err := GroupTxByAddressByAssetSymbol(transactions, repository)
	if err != nil {
		logger.Error("Error grouping By Address", err)
		return
	}
	for addressAndAssetSymbol, addressTransactions := range transactionsPerAddressPerAssetSymbol {
		stringSlice := strings.Split(addressAndAssetSymbol, constants.SWEEP_GROUPING_SEPERATOR)
		var address = stringSlice[0]
		sum := CalculateSum(addressTransactions)
		logger.Info("Sweeping %s with total of %d", address, sum)
		if err := sweepPerAddress(cache, config, repository, serviceErr, addressTransactions, sum, address); err != nil {
			logger.Error("Error response from Sweep job : %+v while sweepPerAddress for address %s", err, address)
			continue
		}
	}
	//batch process btc
	if len(btcAddresses) > 0 {
		if err := sweepBatchTx(cache, config, repository, serviceErr, btcAddresses, btcAssetTransactionsToSweep); err != nil {
			logger.Error("Error response from Sweep job : %+v while sweeping batch transactions for BTC", err)
			if err := tasks.ReleaseLock(&userAssetRepository, cache, config, token, serviceErr); err != nil {
				logger.Error("Could not release lock", err)
				return
			}
			return
		}
	}
	if err := tasks.ReleaseLock(&userAssetRepository, cache, config, token, serviceErr); err != nil {
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

func CalculateSumOfBtcBatch(addressTransactions []model.Transaction) float64 {
	//Get total sum to be swept for this batch
	var sum = float64(0)
	for _, tx := range addressTransactions {
		value, _ := strconv.ParseFloat(tx.Value, 64)
		sum += value
	}
	return sum
}

func sweepBatchTx(cache *cache.Memory, config Config.Data, repository database.IRepository, serviceErr dto.ExternalServicesRequestErr, btcAddresses []string, btcAssetTransactionsToSweep []model.Transaction) error {
	// Calls key-management to batch sign transaction
	recipientData := []dto.BatchRecipients{}
	//get float
	floatAccount, err := getFloatDetails(repository, "BTC")
	if err != nil {
		return err
	}

	//check total sum threshold for this batch
	totalSweepSum := CalculateSumOfBtcBatch(btcAssetTransactionsToSweep)
	if totalSweepSum < config.BTC_minimumSweep {
		logger.Error("Error response from sweep job : Total sweep sum %v for asset (%s) is below the minimum sweep %v, so terminating sweep process", totalSweepSum, constants.COIN_BTC, config.BTC_minimumSweep, err)
		return err
	}

	sweepParam, err := GetSweepParams(cache, config, repository, floatAccount, totalSweepSum)
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
		ProcessType:   constants.SWEEPPROCESS,
	}
	signBatchTransactionAndBroadcastResponse := dto.SignAndBroadcastResponse{}
	KeyManagementService := services.NewKeyManagementService(cache, config, repository, &serviceErr)
	if err := KeyManagementService.SignBatchTransactionAndBroadcast(nil, cache, config, signBatchTransactionAndBroadcastRequest, &signBatchTransactionAndBroadcastResponse, serviceErr); err != nil {
		logger.Error("Error response from SignBatchTransactionAndBroadcast : %+v while sweeping batch transactions for BTC", err)
		return err
	}
	if err := updateSweptStatus(btcAssetTransactionsToSweep, repository, config); err != nil {
		return err
	}
	return nil

}

func sweepPerAddress(cache *cache.Memory, config Config.Data, repository database.IRepository, serviceErr dto.ExternalServicesRequestErr, addressTransactions []model.Transaction, sum float64, recipientAddress string) error {
	transactionListInfo, e := getTransactionListInfo(repository, config, addressTransactions)
	if e != nil {
		return e
	}
	floatAccount, err := getFloatDetails(repository, transactionListInfo.AssetSymbol)
	if err != nil {
		return err
	}
	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: floatAccount.AssetSymbol, IsEnabled: true}, &denomination); err != nil {
		logger.Error("Error response from sweep job : %+v while trying to denomination of float asset", err)
		return err
	}

	userAssetRepository := database.UserAssetRepository{BaseRepository: database.BaseRepository{Database: database.Database{config, repository.Db()}}}
	toAddress, addressMemo, err := GetSweepAddressAndMemo(cache, config, &userAssetRepository, floatAccount)
	if err != nil {
		logger.Error("Error response from Sweep job : %+v while getting sweep toAddress and memo for %s", err, floatAccount.AssetSymbol)
		return err
	}

	//Check that sweep amount is not below the minimum sweep amount
	isAmountSufficient, err := CheckSweepMinimum(denomination, config, sum)
	if !isAmountSufficient {
		return err
	}
	//Do this only for BEp-2 tokens and not for BNB itself
	if denomination.CoinType == constants.BNBTOKENSLIP && denomination.AssetSymbol != constants.COIN_BNB {
		//send sweep fee to main address
		err, _ := fundSweepFee(floatAccount, denomination, recipientAddress, cache, config, serviceErr, addressTransactions, repository)
		if err != nil {
			return err
		}
		time.Sleep(time.Second * constants.FUND_SWEEP_FEE_WAIT_TIME)
	}

	// Calls key-management to sign transaction
	signTransactionRequest := dto.SignTransactionRequest{
		FromAddress: recipientAddress,
		ToAddress:   toAddress,
		Memo:        addressMemo,
		Amount:      big.NewInt(0),
		AssetSymbol: transactionListInfo.AssetSymbol,
		IsSweep:     true,
		ProcessType: constants.SWEEPPROCESS,
		Reference:   addressTransactions[0].TransactionReference,
	}
	SignTransactionAndBroadcastResponse := dto.SignAndBroadcastResponse{}
	KeyManagementService := services.NewKeyManagementService(cache, config, repository, &serviceErr)
	if err := KeyManagementService.SignTransactionAndBroadcast(signTransactionRequest, &SignTransactionAndBroadcastResponse); err != nil {
		logger.Error("Error response from SignTransactionAndBroadcast : %+v while sweeping for address with id %+v", err, recipientAddress)
		switch serviceErr.Code {
		case errorcode.INSUFFICIENT_FUNDS:
			if err := updateSweptStatus(addressTransactions, repository, logger); err != nil {
				return err
			}
		default:
			return err
		}
	}
	if err := updateSweptStatus(addressTransactions, repository, config); err != nil {
		return err
	}
	return nil
}

func CheckSweepMinimum(denomination model.Denomination, config Config.Data, sum float64) (bool, error) {
	var minimumSweep float64
	switch denomination.AssetSymbol {
	case constants.COIN_ETH:
		minimumSweep = config.ETH_minimumSweep

	case constants.COIN_BNB:
		minimumSweep = config.BNB_minimumSweep
	case constants.COIN_BUSD:
		minimumSweep = config.BUSD_minimumSweep
	default:
		return false, appError.Err{
			ErrType: errorcode.SWEEP_ERROR_ASSET_NOT_SUPPORTED,
			Err:     errors.New(errorcode.SWEEP_ERROR_ASSET_NOT_SUPPORTED),
		}
	}
	if sum < minimumSweep {
		logger.Error("Error response from sweep job : Total sweep sum %v for asset (%s) is below the minimum sweep %v, so terminating sweep process", sum, denomination.AssetSymbol, minimumSweep)
		return false, appError.Err{
			ErrType: errorcode.SWEEP_ERROR_INSUFFICIENT,
			Err:     errors.New(errorcode.SWEEP_ERROR_INSUFFICIENT),
		}
	}
	return true, nil
}

func getTransactionListInfo(repository database.IRepository, config Config.Data, assetTransactions []model.Transaction) (dto.TransactionListInfo, error) {
	//need representative Asset to get common things about this list like symbol, Decimals etc
	var transactionListInfo = dto.TransactionListInfo{}

	recipientAsset := model.UserAsset{}
	//all the tx in assetTransactions have the same recipientId so get info from the 0th position
	userAssetRepository := database.UserAssetRepository{BaseRepository: database.BaseRepository{Database: database.Database{config, repository.Db()}}}
	if err := userAssetRepository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetTransactions[0].RecipientID}}, &recipientAsset); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
		return dto.TransactionListInfo{}, err
	}
	transactionListInfo.AssetSymbol = recipientAsset.AssetSymbol
	transactionListInfo.Decimal = recipientAsset.Decimal
	return transactionListInfo, nil
}

func GroupTxByAddressByAssetSymbol(transactions []model.Transaction, repository database.IRepository) (map[string][]model.Transaction, error) {
	//loop over assetTransactions, get the chainTx and group by address
	//group transactions by addresses
	transactionsPerRecipientAddress := make(map[string][]model.Transaction)
	for _, tx := range transactions {
		logger.Info("GroupByTx - getting chain transaction for  %+v", tx.ID)
		chainTransaction := model.ChainTransaction{}
		e := getChainTransaction(repository, tx, &chainTransaction)
		logger.Info("GroupByTx - chaintx is  %+v", chainTransaction)
		if e != nil {
			logger.Info("GroupByTx - getting chain transaction FAILED for  %+v", tx.ID)
			return nil, e
		}
		if chainTransaction.RecipientAddress != "" {
			key := chainTransaction.RecipientAddress + constants.SWEEP_GROUPING_SEPERATOR + tx.AssetSymbol
			transactionsPerRecipientAddress[key] = append(transactionsPerRecipientAddress[key], tx)
		}

	}
	logger.Info("GroupByTx -  transactionsPerRecipientAddress is ", transactionsPerRecipientAddress)

	return transactionsPerRecipientAddress, nil
}

func getChainTransaction(repository database.IRepository, tx model.Transaction, chainTransaction *model.ChainTransaction) error {
	err := repository.Get(&model.ChainTransaction{BaseModel: model.BaseModel{ID: tx.OnChainTxId}}, chainTransaction)
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

func fundSweepFee(floatAccount model.HotWalletAsset, denomination model.Denomination, recipientAddress string, cache *cache.Memory, config Config.Data, serviceErr dto.ExternalServicesRequestErr, assetTransactions []model.Transaction, repository database.IRepository) (error, bool) {

	request := dto.OnchainBalanceRequest{
		AssetSymbol: denomination.MainCoinAssetSymbol,
		Address:     recipientAddress,
	}
	mainCoinOnChainBalanceResponse := dto.OnchainBalanceResponse{}
	CryptoAdapterService := services.NewCryptoAdapterService(cache, config, repository, &serviceErr)
	if err := CryptoAdapterService.GetOnchainBalance(request, &mainCoinOnChainBalanceResponse); err != nil {
		logger.Error("Error response from Sweep job : %+v while getting on-chain balance for %+v", err, recipientAddress)
		return err, true
	}
	mainCoinOnChainBalance, _ := strconv.ParseUint(mainCoinOnChainBalanceResponse.Balance, 10, 64)
	//check if onchain balance in main coin asset is less than floatAccount.SweepFee
	if int64(mainCoinOnChainBalance) < denomination.SweepFee {
		// Calls key-management to sign sweep fee transaction
		signTransactionAndBroadcastRequest := dto.SignTransactionRequest{
			FromAddress: floatAccount.Address,
			ToAddress:   recipientAddress,
			Amount:      big.NewInt(denomination.SweepFee),
			AssetSymbol: denomination.MainCoinAssetSymbol,
			//this currently only supports coins that supports Memo, ETH will be ignored
			Memo:        constants.SWEEPMEMOBNB,
			ProcessType: constants.FLOATPROCESS,
			Reference:   fmt.Sprintf("%s-%s", denomination.MainCoinAssetSymbol, assetTransactions[0].TransactionReference),
		}
		signTransactionAndBroadcastResponse := dto.SignAndBroadcastResponse{}
		KeyManagementService := services.NewKeyManagementService(cache, config, repository, &serviceErr)
		if err := KeyManagementService.SignTransactionAndBroadcast(signTransactionAndBroadcastRequest, &signTransactionAndBroadcastResponse); err != nil {
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

func GetSweepParams(cache *cache.Memory, config Config.Data, repository database.IRepository, floatAccount model.HotWalletAsset, sweepFund float64) (BTCSweepParam, error) {

	sweepParam := BTCSweepParam{}
	serviceErr := dto.ExternalServicesRequestErr{}

	userAssetRepository := database.UserAssetRepository{BaseRepository: database.BaseRepository{Database: database.Database{config, repository.Db()}}}
	totalUsersBalance, err := tasks.GetTotalUserBalance(repository, floatAccount.AssetSymbol, &userAssetRepository)
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
	CryptoAdapterService := services.NewCryptoAdapterService(cache, config, repository, &serviceErr)
	if err := CryptoAdapterService.GetOnchainBalance(onchainBalanceRequest, &floatOnChainBalanceResponse); err != nil {
		logger.Error("Error response from Sweep job : %+v while getting float on-chain balance for %+v", err, floatAccount.AssetSymbol)
		return sweepParam, err
	}
	floatOnChainBalance, _ := new(big.Float).SetPrec(prec).SetString(floatOnChainBalanceResponse.Balance)
	logger.Info("SWEEP_OPERATION : Float on-chain balance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, floatOnChainBalance)

	// Get float manager parameters to calculate float range
	floatManagerParams, err := tasks.GetFloatParamFor(floatAccount.AssetSymbol, repository)
	if err != nil {
		return sweepParam, err
	}
	minimumFloatBalance, maximumFloatBalance := GetFloatBalanceRange(floatManagerParams, totalUsersBalance)

	// Get total deposit sum from the last run of this job
	depositSumFromLastRun, err := tasks.GetDepositsSumForAssetFromDate(repository, config, floatAccount.AssetSymbol, floatAccount)
	if err != nil {
		logger.Info("error with float manager process, while trying to get the total deposit sum from last run : %+v", err)
		return sweepParam, err
	}
	logger.Info("depositSumFromLastRun for this hot wallet (%s) is %+v", floatAccount.AssetSymbol, depositSumFromLastRun)

	// Get total withdrawal sum from the last run of this job
	withdrawalSumFromLastRun, err := tasks.GetWithdrawalsSumForAssetFromDate(repository, config, floatAccount.AssetSymbol, floatAccount)
	if err != nil {
		logger.Info("error with float manager process, while trying to get the total withdrawal sum from last run : %+v", err)
		return sweepParam, err
	}
	logger.Info("withdrawalSumFromLastRun for this hot wallet %+v is %+v", floatAccount.AssetSymbol, withdrawalSumFromLastRun)

	floatDeficit := GetFloatDeficit(depositSumFromLastRun, withdrawalSumFromLastRun, minimumFloatBalance, maximumFloatBalance, floatOnChainBalance)

	brokerageAccountResponse, err := GetBrokerAccountFor(floatAccount.AssetSymbol, repository, cache, config, serviceErr)
	if err != nil {
		return sweepParam, err
	}

	floatPercent, brokeragePercent := GetSweepPercentages(floatOnChainBalance, minimumFloatBalance, floatDeficit, big.NewFloat(sweepFund), totalUsersBalance, floatManagerParams)

	sweepParam = BTCSweepParam{
		FloatAddress:     floatAccount.Address,
		FloatPercent:     floatPercent,
		BrokerageAddress: brokerageAccountResponse.Address,
		BrokeragePercent: brokeragePercent,
	}

	return sweepParam, nil
}

func GetSweepPercentages(floatOnChainBalance, minimumFloatBalance, floatDeficit, sweepFund, totalUsersBalance *big.Float, floatManagerParams model.FloatManagerParam) (int64, int64) {

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

func GetBrokerAccountFor(assetSymbol string, repository database.IRepository, cache *cache.Memory, config Config.Data, serviceErr dto.ExternalServicesRequestErr) (dto.DepositAddressResponse, error) {

	brokerageAccountResponse := dto.DepositAddressResponse{}
	denomination := model.Denomination{}
	err := repository.GetByFieldName(&model.Denomination{AssetSymbol: assetSymbol, IsEnabled: true}, &denomination)
	if err != nil {
		return brokerageAccountResponse, err
	}

	OrderBookService := services.NewOrderBookService(cache, config, repository, &serviceErr)
	if *denomination.IsToken {
		err = OrderBookService.GetDepositAddress(assetSymbol, denomination.MainCoinAssetSymbol, &brokerageAccountResponse)
	} else {
		err = OrderBookService.GetDepositAddress(assetSymbol, "", &brokerageAccountResponse)
	}
	if err != nil {
		return brokerageAccountResponse, err
	}

	logger.Info("SWEEP_OPERATION : Brokerage account for this hot wallet %+v is %+v", assetSymbol, brokerageAccountResponse)
	return brokerageAccountResponse, nil
}

func GetFloatDeficit(depositSumFromLastRun, withdrawalSumFromLastRun, minimumBalance, maximumBalance, onChainBalance *big.Float) *big.Float {

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

func GetFloatBalanceRange(floatManagerParams model.FloatManagerParam, totalUsersBalance *big.Float) (*big.Float, *big.Float) {

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

func GetSweepAddressAndMemo(cache *cache.Memory, config Config.Data, repository database.IUserAddressRepository, floatAccount model.HotWalletAsset) (string, string, error) {

	userAssetRepository := database.UserAssetRepository{BaseRepository: database.BaseRepository{Database: database.Database{config, repository.Db()}}}
	totalUsersBalance, err := tasks.GetTotalUserBalance(repository, floatAccount.AssetSymbol, &userAssetRepository)
	if err != nil {
		return "", "", err
	}
	logger.Info("SWEEP_OPERATION : Total users balance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, totalUsersBalance)

	// Get float chain balance
	prec := uint(64)
	serviceErr := dto.ExternalServicesRequestErr{}
	onchainBalanceRequest := dto.OnchainBalanceRequest{
		AssetSymbol: floatAccount.AssetSymbol,
		Address:     floatAccount.Address,
	}
	floatOnChainBalanceResponse := dto.OnchainBalanceResponse{}
	CryptoAdapterService := services.NewCryptoAdapterService(cache, config, &userAssetRepository, &serviceErr)
	if err := CryptoAdapterService.GetOnchainBalance(onchainBalanceRequest, &floatOnChainBalanceResponse); err != nil {
		logger.Error("SWEEP_OPERATION, err : %+v while getting float on-chain balance for %+v", err, floatAccount.AssetSymbol)
		return "", "", err
	}
	floatOnChainBalance, _ := new(big.Float).SetPrec(prec).SetString(floatOnChainBalanceResponse.Balance)
	logger.Info("SWEEP_OPERATION : Float on-chain balance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, floatOnChainBalance)

	// Get broker account
	brokerageAccountResponse := dto.DepositAddressResponse{}
	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: floatAccount.AssetSymbol, IsEnabled: true}, &denomination); err != nil {
		return "", "", err
	}

	OrderBookService := services.NewOrderBookService(cache, config, &userAssetRepository, &serviceErr)
	if *denomination.IsToken {
		err = OrderBookService.GetDepositAddress(floatAccount.AssetSymbol, denomination.MainCoinAssetSymbol, &brokerageAccountResponse)
	} else {
		err = OrderBookService.GetDepositAddress(floatAccount.AssetSymbol, "", &brokerageAccountResponse)
	}
	if err != nil {
		return "", "", err
	}
	logger.Info("SWEEP_OPERATION : Brokerage account for this hot wallet %+v is %+v", floatAccount.AssetSymbol, brokerageAccountResponse)

	// Get float manager parameters to calculate minimum float
	floatManagerParams, err := tasks.GetFloatParamFor(floatAccount.AssetSymbol, repository)
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

func getFloatDetails(repository database.IRepository, symbol string) (model.HotWalletAsset, error) {
	//Get the float address
	var floatAccount model.HotWalletAsset
	if err := repository.Get(&model.HotWalletAsset{AssetSymbol: symbol}, &floatAccount); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id and trying to get float detials", err)
		return model.HotWalletAsset{}, err
	}
	return floatAccount, nil
}

func updateSweptStatus(assetTransactions []model.Transaction, repository database.IRepository, config Config.Data) error {
	//update all assetTransactions with new swept status
	var assetIdList []uuid.UUID
	for _, tx := range assetTransactions {
		assetIdList = append(assetIdList, tx.ID)
	}
	userAssetRepository := database.UserAssetRepository{BaseRepository: database.BaseRepository{Database: database.Database{config, repository.Db()}}}
	if err := userAssetRepository.BulkUpdateTransactionSweptStatus(assetIdList); err != nil {
		logger.Error("Error response from Sweep job : %+v while broadcasting to chain", err)
		return err
	}
	return nil
}

func ExecuteSweepCronJob(cache *cache.Memory, config Config.Data, repository database.IRepository) {
	c := cron.New()
	c.AddFunc(config.SweepCronInterval, func() { SweepTransactions(cache, config, repository) })
	c.Start()
}
