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
	//group transactions by recipientId
	transactionsPerAssetId := make(map[uuid.UUID][]model.Transaction)
	for _, tx := range transactions {
		transactionsPerAssetId[tx.RecipientID] = append(transactionsPerAssetId[tx.RecipientID], tx)
	}

	var btcAssets []string
	var btcAssetTransactionsToSweep []model.Transaction
	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	for assetId, assetTransactions := range transactionsPerAssetId {
		//Filter BTC assets, save in a seperate list for batch processing and skip individual processing
		//need recipient Asset to check assetSymbol
		recipientAsset := model.UserAsset{}
		//all the tx in assetTransactions have the same recipientId so just pass the 0th position
		if err := userAssetRepository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetId}}, &recipientAsset); err != nil {
			logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
			if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
				logger.Error("Could not release lock", err)
				return
			}
			return
		}
		if recipientAsset.AssetSymbol == utility.COIN_BTC {
			//get recipient address for each transaction
			transactionsPerRecipientAddress, err := groupTxByAddress(assetTransactions, repository, logger, recipientAsset)
			if err != nil {
				return
			}
			for address, _ := range transactionsPerRecipientAddress {
				btcAssets = append(btcAssets, address)
			}
			btcAssetTransactionsToSweep = append(btcAssetTransactionsToSweep, assetTransactions...)
			//skip futher processing for this asset, will be included a part of batch btc processing
			continue
		}
		transactionsPerRecipientAddress, err := groupTxByAddress(assetTransactions, repository, logger, recipientAsset)
		if err != nil {
			return
		}
		for address, addressTransactions := range transactionsPerRecipientAddress {
			sum := calculateSum(addressTransactions, recipientAsset)
			logger.Info("Sweeping %s with total of %d", address, sum)
			if err := sweepPerAssetIdPerAddress(cache, logger, config, repository, serviceErr, assetTransactions, sum, address); err != nil {
				continue
			}
		}
	}
	//batch process btc
	if len(btcAssets) > 0 {
		if err := sweepBatchTx(cache, logger, config, repository, serviceErr, btcAssets, btcAssetTransactionsToSweep); err != nil {
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

func calculateSum(addressTransactions []model.Transaction, recipientAsset model.UserAsset) int64 {
	//Get total sum to be swept for this assetId address
	var sum = int64(0)
	for _, tx := range addressTransactions {
		//convert to native units
		balance, _ := strconv.ParseFloat(tx.Value, 64)
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

func sweepBatchTx(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, serviceErr dto.ServicesRequestErr, btcAssets []string, btcAssetTransactionsToSweep []model.Transaction) error {
	// Calls key-management to batch sign transaction
	recipientData := []dto.BatchRecipients{}
	//get float
	floatAccount, err := getFloatDetails(repository, "BTC", logger)
	if err != nil {
		return err
	}

	//check total sum threshold for this batch
	totalSweepSum := CalculateSumOfBtcBatch(btcAssetTransactionsToSweep)
	if totalSweepSum < config.SweepBtcBatchMinimum {
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
	signTransactionRequest := dto.BatchBTCRequest{
		AssetSymbol:   "BTC",
		ChangeAddress: sweepParam.BrokerageAddress,
		IsSweep:       true,
		Origins:       btcAssets,
		Recipients:    recipientData,
	}
	signTransactionResponse := dto.SignTransactionResponse{}
	if err := services.SignBatchBTCTransaction(nil, cache, logger, config, signTransactionRequest, &signTransactionResponse, serviceErr); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping batch transactions for BTC", err)
		return err
	}
	if err := broadcastSweepTx(signTransactionResponse, config, "BTC", cache, logger, serviceErr, btcAssetTransactionsToSweep, repository); err != nil {
		return err
	}
	if err := updateSweptStatus(btcAssetTransactionsToSweep, repository, logger); err != nil {
		return err
	}
	return nil

}

func sweepPerAssetIdPerAddress(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, serviceErr dto.ServicesRequestErr, assetTransactions []model.Transaction, sum int64, recipientAddress string) error {
	//need recipient Asset to get recipient address
	recipientAsset := model.UserAsset{}
	//all the tx in assetTransactions have the same recipientId so just pass the 0th position
	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	if err := userAssetRepository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetTransactions[0].RecipientID}}, &recipientAsset); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
		return err
	}
	floatAccount, err := getFloatDetails(repository, recipientAsset.AssetSymbol, logger)
	if err != nil {
		return err
	}
	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: floatAccount.AssetSymbol, IsEnabled: true}, &denomination); err != nil {
		logger.Error("Error response from sweep process : %+v while trying to denomination of float asset", err)
		return err
	}

	//Do this only for BEp-2 tokens and not for BNB itself
	if denomination.CoinType == utility.BNBTOKENSLIP && denomination.AssetSymbol != "BNB" {
		//Check that fee is below X% of the total value.
		err = feeThresholdCheck(denomination.SweepFee, sum, config, logger, recipientAsset)
		if err != nil {
			return err
		}
		//send sweep fee to main address
		err, done := fundSweepFee(floatAccount, denomination, recipientAddress, cache, logger, config, serviceErr, recipientAsset, assetTransactions, repository)
		if done {
			return err
		}
	}

	toAddress, addressMemo, err := GetSweepAddressAndMemo(cache, logger, config, repository, floatAccount)
	if err != nil {
		logger.Error("Error response from Sweep job : %+v while getting sweep toAddress and memo for %s", err, floatAccount.AssetSymbol)
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
	}
	signTransactionResponse := dto.SignTransactionResponse{}
	if err := services.SignTransaction(cache, logger, config, signTransactionRequest, &signTransactionResponse, serviceErr); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
		return err
	}
	//Check that fee is below X% of the total value.
	if err := feeThresholdCheck(signTransactionResponse.Fee, sum, config, logger, recipientAsset); err != nil {
		return err
	}
	if err := broadcastSweepTx(signTransactionResponse, config, recipientAsset.AssetSymbol, cache, logger, serviceErr, assetTransactions, repository); err != nil {
		return err
	}
	if err := updateSweptStatus(assetTransactions, repository, logger); err != nil {
		return err
	}
	return nil
}

func groupTxByAddress(assetTransactions []model.Transaction, repository database.BaseRepository, logger *utility.Logger, recipientAsset model.UserAsset) (map[string][]model.Transaction, error) {
	//loop over assetTransactions, get the chainTx and group by address
	//group transactions by addresses
	transactionsPerRecipientAddress := make(map[string][]model.Transaction)
	for _, tx := range assetTransactions {
		chainTransaction := model.ChainTransaction{}
		err := repository.Get(&model.ChainTransaction{BaseModel: model.BaseModel{ID: tx.OnChainTxId}}, &chainTransaction)
		if err != nil {
			logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v cant fetch chainTransaction for depsoit tx",
				err, recipientAsset.ID)
			return nil, err
		}
		if chainTransaction.RecipientAddress != "" {
			transactionsPerRecipientAddress[chainTransaction.RecipientAddress] = append(transactionsPerRecipientAddress[chainTransaction.RecipientAddress], tx)
		} else {
			//Case when the chainTransaction.RecipientAddress is not set and we need to try to sweep all available address types
			//get allrecipient address
			recipientAddresses := []model.UserAddress{}
			if err := repository.Get(model.UserAddress{AssetID: recipientAsset.ID}, &recipientAddresses); err != nil {
				logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
				return nil, err
			}
			for _, address := range recipientAddresses {
				transactionsPerRecipientAddress[address.Address] = append(transactionsPerRecipientAddress[address.Address], tx)
			}
		}

	}
	return transactionsPerRecipientAddress, nil
}

func feeThresholdCheck(fee int64, sum int64, config Config.Data, logger *utility.Logger, recipientAsset model.UserAsset) error {
	if (((fee) / sum) * 100) > config.SweepFeePercentageThreshold {
		logger.Error("Skipping asset, %+v ratio of fee to sum for this asset with asset symbol %+v is greater than the sweepFeePercentageThreshold, would be too expensive to sweep %+v", recipientAsset.ID, recipientAsset.AssetSymbol, config.SweepFeePercentageThreshold)
		return errors.New(fmt.Sprintf("Skipping asset, %s ratio of fee to sum for this asset with asset symbol %s is greater than the sweepFeePercentageThreshold, would be too expensive to sweep %s", recipientAsset.ID, recipientAsset.AssetSymbol, config.SweepFeePercentageThreshold))
	}
	return nil
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
		signTransactionRequest := dto.SignTransactionRequest{
			FromAddress: floatAccount.Address,
			ToAddress:   recipientAddress,
			Amount:      big.NewInt(denomination.SweepFee),
			AssetSymbol: denomination.MainCoinAssetSymbol,
			//this currently only supports coins that supports Memo, ETH will not be ignored
			Memo: utility.SWEEPMEMOBNB,
		}
		signTransactionResponse := dto.SignTransactionResponse{}
		if err := services.SignTransaction(cache, logger, config, signTransactionRequest, &signTransactionResponse, serviceErr); err != nil {
			logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
			return err, true
		}
		if err := broadcastSweepTx(signTransactionResponse, config, recipientAsset.AssetSymbol, cache, logger, serviceErr, assetTransactions, repository); err != nil {
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

func broadcastSweepTx(signTransactionResponse dto.SignTransactionResponse, config Config.Data, symbol string, cache *utility.MemoryCache, logger *utility.Logger, serviceErr dto.ServicesRequestErr, assetTransactions []model.Transaction, repository database.BaseRepository) error {
	// Send the signed data to crypto adapter to send to chain
	broadcastToChainRequest := dto.BroadcastToChainRequest{
		SignedData:  signTransactionResponse.SignedData,
		AssetSymbol: symbol,
		ProcessType: utility.SWEEPPROCESS,
	}
	broadcastToChainResponse := dto.BroadcastToChainResponse{}
	if err := services.BroadcastToChain(cache, logger, config, broadcastToChainRequest, &broadcastToChainResponse, serviceErr); err != nil {
		logger.Error("Error response from Sweep job : %+v while broadcasting to chain", err)
		return err
	}
	return nil
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
