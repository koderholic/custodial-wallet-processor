package tasks

import (
	"github.com/robfig/cron/v3"
	uuid "github.com/satori/go.uuid"
	"math"
	"strconv"
	"time"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"
)

func manageFloat(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository) {
	logger.Info("Float manager process begins")
	serviceErr := model.ServicesRequestErr{}
	token, err := acquireLock(cache, logger, config, serviceErr)
	if err != nil {
		logger.Error("Could not acquire lock", err)
		return
	}
	//get float balance
	floatAccounts, err := getFloatAccounts(repository, logger)
	if err != nil {
		return
	}
	for _, floatAccount := range floatAccounts {
		request := model.OnchainBalanceRequest{
			AssetSymbol: floatAccount.AssetSymbol,
			Address:     floatAccount.Address,
		}
		floatOnChainBalanceResponse := model.OnchainBalanceResponse{}
		services.GetOnchainBalance(cache, logger, config, request, &floatOnChainBalanceResponse, serviceErr)

		//get minimum amount
		totalUserBalance, err := getTotalUserBalance(repository, floatAccount.AssetSymbol, logger)
		if err != nil {
			break
		}
		depositSumFromLastRun, err := getDepositsSumForAssetFromDate(repository, floatAccount.AssetSymbol, logger, floatAccount)
		if err != nil {
			break
		}
		withdrawalSumFromLastRun, err := getWithdrawalsSumForAssetFromDate(repository, floatAccount.AssetSymbol, logger, floatAccount)
		if err != nil {
			break
		}
		minimum := floatAccount.ReservedBalance + int64((float64(config.FloatPercentage)/100)*float64(totalUserBalance))
		maximum := minimum + Abs(depositSumFromLastRun-withdrawalSumFromLastRun)
		floatOnChainBalance, _ := strconv.ParseInt(floatOnChainBalanceResponse.Balance, 10, 64)
		//it checks if the float balance is below the minimum balance or above the maximum balance
		if floatOnChainBalance < minimum {
			//if below the minimum balance, it then checks if deposit - withdrawal < 0,
			// then we call binance broker api to fund hot wallet and raise the float balance from
			// it's deficit amount to the maximum amount (residual + % of total user
			// balance + delta(total_deposit - total_withdrawal) since its last run).
			if depositSumFromLastRun-withdrawalSumFromLastRun < 0 {
				binanceAssetBalances := model.BinanceAssetBalances{}
				services.GetOnChainBinanceAssetBalances(cache, logger, config, &binanceAssetBalances, serviceErr)
				for _, coin := range binanceAssetBalances.CoinList {
					if coin.Coin == floatAccount.AssetSymbol {
						//check if balance is enough to fill deficit
						balance, _ := strconv.ParseFloat(coin.Balance, 64)
						denomination := dto.Denomination{}
						if err := repository.GetByFieldName(&dto.Denomination{AssetSymbol: floatAccount.AssetSymbol, IsEnabled: true}, &denomination); err != nil {
							logger.Error("Error response from Float manager : %+v while trying to denomination of float asset", err)
							break
						}
						denominationDecimal := float64(denomination.Decimal)
						scaledBalance := int64(balance * math.Pow(10, denominationDecimal))

						if scaledBalance > (maximum - floatOnChainBalance) {
							//Go ahead and withdraw to hotwallet
							money := model.Money{
								Value:        strconv.FormatInt(scaledBalance-(maximum-floatOnChainBalance), 10),
								Denomination: floatAccount.AssetSymbol,
							}
							requestData := model.WitdrawToHotWalletRequest{
								Address:            floatAccount.Address,
								Name:               floatAccount.AssetSymbol + " Bundle Hot wallet",
								Amount:             money,
								TransactionFeeFlag: false,
							}
							responseData := model.WitdrawToHotWalletResponse{}
							services.WithdrawToHotWallet(cache, logger, config, requestData, &responseData, serviceErr)
						} else {
							//todo trigger cold wallet notifications in next PR
						}
					}

				}
			} else {
				//But if it then checks if deposit - withdrawal >= 0, then we trigger call to cold wallet
				// using notification service to raise the float balance from it's deficit amount to
				// or above the minimum amount (residual amount)
				//todo trigger cold wallet notification in next PR
			}
		}
		if floatOnChainBalance > maximum {
			//debit float address
			depositAddressResponse := model.DepositAddressResponse{}
			services.GetDepositAddress(cache, logger, config, floatAccount.AssetSymbol, "", &depositAddressResponse, serviceErr)
			signTxAndBroadcastToChain(cache, repository, (floatOnChainBalance - maximum), depositAddressResponse.Address, logger, config, floatAccount, serviceErr)
		}
	}
	if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
		logger.Error("Could not release lock", err)
		return
	}
	logger.Info("Float manager process ends successfully, lock released")
}

//total liability at any given time
func getTotalUserBalance(repository database.BaseRepository, assetSymbol string, logger *utility.Logger) (int64, error) {
	depositSum, err := getDepositsSumForAsset(repository, assetSymbol, logger)
	if err != nil {
		logger.Error("Error response from Float manager : %+v while trying to getTotalUserBalance", err)
		return 0, err
	}
	withdrawalSum, err := getWithdrawalsSumForAsset(repository, assetSymbol, logger)
	if err != nil {
		logger.Error("Error response from Float manager : %+v while trying to getTotalUserBalance", err)
		return 0, err
	}
	creditSum, err := getCreditsForAsset(repository, assetSymbol, logger)
	if err != nil {
		logger.Error("Error response from Float manager : %+v while trying to getTotalUserBalance", err)
		return 0, err
	}
	debitSum, err := getDebitsForAsset(repository, assetSymbol, logger)
	if err != nil {
		logger.Error("Error response from Float manager : %+v while trying to getTotalUserBalance", err)
		return 0, err
	}
	return (depositSum - withdrawalSum + creditSum - debitSum), nil
}

func getFloatAccounts(repository database.BaseRepository, logger *utility.Logger) ([]dto.HotWalletAsset, error) {
	//Get the float address
	floatAccounts := []dto.HotWalletAsset{}
	if err := repository.Fetch(&floatAccounts); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get float balances", err)
		return nil, err
	}
	return floatAccounts, nil
}
func getRecipientAsset(repository database.BaseRepository, assetId uuid.UUID, recipientAsset *dto.UserAsset, logger *utility.Logger) {
	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	if err := userAssetRepository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: assetId}}, &recipientAsset); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
		return
	}
}

func Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func getDepositsSumForAsset(repository database.BaseRepository, assetSymbol string, logger *utility.Logger) (int64, error) {
	deposits := []dto.Transaction{}
	if err := repository.FetchByFieldName(dto.Transaction{
		TransactionTag: "DEPOSIT",
		AssetSymbol:    assetSymbol,
	}, deposits); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get deposits", err)
		return 0, err
	}
	sum := int64(0)
	for _, deposit := range deposits {
		recipientAsset := dto.UserAsset{}
		getRecipientAsset(repository, deposit.RecipientID, &recipientAsset, logger)
		//convert to native units
		balance, _ := strconv.ParseFloat(deposit.Value, 64)
		denominationDecimal := float64(recipientAsset.Decimal)
		scaledBalance := int64(balance * math.Pow(10, denominationDecimal))
		sum = sum + scaledBalance
	}
	return sum, nil
}

func getDepositsSumForAssetFromDate(repository database.BaseRepository, assetSymbol string, logger *utility.Logger, hotWallet dto.HotWalletAsset) (int64, error) {
	deposits := []dto.Transaction{}
	if err := repository.FetchByFieldNameFromDate(dto.Transaction{
		TransactionTag: "DEPOSIT",
		AssetSymbol:    assetSymbol,
	}, deposits, hotWallet.LastDepositCreatedAt); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get deposits", err)
		return 0, err
	}
	sum := int64(0)
	var lastCreatedAt time.Time
	for _, deposit := range deposits {
		recipientAsset := dto.UserAsset{}
		getRecipientAsset(repository, deposit.RecipientID, &recipientAsset, logger)
		//convert to native units
		balance, _ := strconv.ParseFloat(deposit.Value, 64)
		denominationDecimal := float64(recipientAsset.Decimal)
		scaledBalance := int64(balance * math.Pow(10, denominationDecimal))
		sum = sum + scaledBalance
		lastCreatedAt = deposit.CreatedAt
	}
	if err := repository.Update(&hotWallet, &dto.HotWalletAsset{LastDepositCreatedAt: lastCreatedAt}); err != nil {
		logger.Error("Error occured while updating hot wallet lastCreatedAt to On-going : %s", err)
	}
	return sum, nil
}

func getWithdrawalsSumForAsset(repository database.BaseRepository, assetSymbol string, logger *utility.Logger) (int64, error) {
	withdrawals := []dto.Transaction{}
	if err := repository.FetchByFieldName(dto.Transaction{
		TransactionTag: "WITHDRAW",
		AssetSymbol:    assetSymbol,
	}, withdrawals); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get withdrawals", err)
		return 0, err
	}
	sum := int64(0)
	for _, withdrawal := range withdrawals {
		recipientAsset := dto.UserAsset{}
		getRecipientAsset(repository, withdrawal.InitiatorID, &recipientAsset, logger)
		//convert to native units
		balance, _ := strconv.ParseFloat(withdrawal.Value, 64)
		denominationDecimal := float64(recipientAsset.Decimal)
		scaledBalance := int64(balance * math.Pow(10, denominationDecimal))
		sum = sum + scaledBalance
	}
	return sum, nil
}

func getWithdrawalsSumForAssetFromDate(repository database.BaseRepository, assetSymbol string, logger *utility.Logger, hotWallet dto.HotWalletAsset) (int64, error) {
	withdrawals := []dto.Transaction{}
	if err := repository.FetchByFieldNameFromDate(dto.Transaction{
		TransactionTag: "WITHDRAW",
		AssetSymbol:    assetSymbol,
	}, withdrawals, hotWallet.LastWithdrawalCreatedAt); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get withdrawals", err)
		return 0, err
	}
	var lastCreatedAt time.Time
	sum := int64(0)
	for _, withdrawal := range withdrawals {
		recipientAsset := dto.UserAsset{}
		getRecipientAsset(repository, withdrawal.InitiatorID, &recipientAsset, logger)
		//convert to native units
		balance, _ := strconv.ParseFloat(withdrawal.Value, 64)
		denominationDecimal := float64(recipientAsset.Decimal)
		scaledBalance := int64(balance * math.Pow(10, denominationDecimal))
		sum = sum + scaledBalance
		lastCreatedAt = withdrawal.CreatedAt
	}
	if err := repository.Update(&hotWallet, &dto.HotWalletAsset{LastDepositCreatedAt: lastCreatedAt}); err != nil {
		logger.Error("Error occured while updating hot wallet lastCreatedAt to On-going : %s", err)
	}
	return sum, nil
}

func getCreditsForAsset(repository database.BaseRepository, assetSymbol string, logger *utility.Logger) (int64, error) {
	credits := []dto.Transaction{}
	if err := repository.FetchByFieldName(dto.Transaction{
		TransactionTag: "CREDIT",
		AssetSymbol:    assetSymbol,
	}, credits); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get credits", err)
		return 0, err
	}
	sum := int64(0)
	for _, credit := range credits {
		recipientAsset := dto.UserAsset{}
		getRecipientAsset(repository, credit.RecipientID, &recipientAsset, logger)
		//convert to native units
		balance, _ := strconv.ParseFloat(credit.Value, 64)
		denominationDecimal := float64(recipientAsset.Decimal)
		scaledBalance := int64(balance * math.Pow(10, denominationDecimal))
		sum = sum + scaledBalance
	}
	return sum, nil
}

func getDebitsForAsset(repository database.BaseRepository, assetSymbol string, logger *utility.Logger) (int64, error) {
	debits := []dto.Transaction{}
	if err := repository.FetchByFieldName(dto.Transaction{
		TransactionTag: "DEBITS",
		AssetSymbol:    assetSymbol,
	}, debits); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get debits", err)
		return 0, err
	}
	sum := int64(0)
	for _, debit := range debits {
		recipientAsset := dto.UserAsset{}
		getRecipientAsset(repository, debit.InitiatorID, &recipientAsset, logger)
		//convert to native units
		balance, _ := strconv.ParseFloat(debit.Value, 64)
		denominationDecimal := float64(recipientAsset.Decimal)
		scaledBalance := int64(balance * math.Pow(10, denominationDecimal))
		sum = sum + scaledBalance
	}
	return sum, nil
}

func signTxAndBroadcastToChain(cache *utility.MemoryCache, repository database.BaseRepository, amount int64, destinationAddress string, logger *utility.Logger, config Config.Data, floatAccount dto.HotWalletAsset, serviceErr model.ServicesRequestErr) {
	// Calls key-management to sign transaction
	signTransactionRequest := model.SignTransactionRequest{
		FromAddress: floatAccount.Address,
		//todo critical get deposit address instead
		ToAddress:   floatAccount.Address,
		Amount:      amount,
		AssetSymbol: floatAccount.AssetSymbol,
		IsSweep:     true,
	}
	signTransactionResponse := model.SignTransactionResponse{}
	if err := services.SignTransaction(cache, logger, config, signTransactionRequest, &signTransactionResponse, serviceErr); err != nil {
		logger.Error("Error response from float manager : %+v while signing transaction to debit float for %+v", err, floatAccount.AssetSymbol)
		return
	}
	//need an empty array to be able to reuse the method broadcastAndCompleteSweepTx
	emptyArrayOfTransactions := []dto.Transaction{}
	err, _ := broadcastAndCompleteSweepTx(signTransactionResponse, config, floatAccount.AssetSymbol, cache, logger, serviceErr, emptyArrayOfTransactions, repository)
	if err != nil {
		logger.Error("Error response from float manager : %+v while broadcast transaction to debit float for %+v", err, floatAccount.AssetSymbol)
		return
	}
}

func ExecuteFloatManagerCronJob(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository) {
	c := cron.New()
	c.AddFunc(config.FloatCronInterval, func() { manageFloat(cache, logger, config, repository) })
	c.Start()
}
