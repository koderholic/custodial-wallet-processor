package tasks

import (
	"fmt"
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

func ManageFloat(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, userAssetRepository database.UserAssetRepository) {
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
		totalUserBalance, err := getTotalUserBalance(repository, floatAccount.AssetSymbol, logger, userAssetRepository)
		if err != nil {
			continue
		}
		depositSumFromLastRun, err := getDepositsSumForAssetFromDate(repository, floatAccount.AssetSymbol, logger, floatAccount)
		if err != nil {
			continue
		}
		withdrawalSumFromLastRun, err := getWithdrawalsSumForAssetFromDate(repository, floatAccount.AssetSymbol, logger, floatAccount)
		if err != nil {
			continue
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
						binanceBalance, _ := strconv.ParseFloat(coin.Balance, 64)
						denomination := dto.Denomination{}
						if err := repository.GetByFieldName(&dto.Denomination{AssetSymbol: floatAccount.AssetSymbol, IsEnabled: true}, &denomination); err != nil {
							logger.Error("Error response from Float manager : %+v while trying to denomination of float asset", err)
							break
						}
						denominationDecimal := float64(denomination.Decimal)
						scaledBinanceBalance := int64(binanceBalance * math.Pow(10, denominationDecimal))
						deficit := maximum - floatOnChainBalance
						//decimal units
						deficitInDecimalUnits := float64(deficit) / math.Pow(10, denominationDecimal)
						deficitInDecimalUnits = math.Round(deficitInDecimalUnits*1000) / 1000

						if scaledBinanceBalance > (maximum - floatOnChainBalance) {
							//Go ahead and withdraw to hotwallet
							money := model.Money{
								Value:        strconv.FormatInt(scaledBinanceBalance-(maximum-floatOnChainBalance), 10),
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
							//not enough in binance balance so trigger alert to cold wallet user
							params := make(map[string]string)
							params["amount"] = fmt.Sprintf("%f", deficitInDecimalUnits)
							coldWalletEmails := []model.EmailUser{}
							coldWalletEmails[0] = model.EmailUser{
								Name:  "Yele",
								Email: config.ColdWalletEmail,
							}
							sendEmailRequest := model.SendEmailRequest{
								Subject: "Please fund Bundle hot wallet address for " + floatAccount.AssetSymbol,
								Content: "",
								Template: model.EmailTemplate{
									ID:     "",
									Params: params,
								},
								Sender: model.EmailUser{
									Name:  "Bundle",
									Email: "info@bundle.africa",
								},
								Receivers: coldWalletEmails,
							}
							sendEmailResponse := model.SendEmailResponse{}
							services.SendEmailNotification(cache, logger, config, sendEmailRequest, &sendEmailResponse, serviceErr)
						}
					}

				}
			} else {
				//But if it then checks if deposit - withdrawal >= 0, then we trigger call to cold wallet
				// using notification service to raise the float balance from it's deficit amount to
				// or above the minimum amount (residual amount)
				deficit := minimum - floatOnChainBalance
				denomination := dto.Denomination{}
				if err := repository.GetByFieldName(&dto.Denomination{AssetSymbol: floatAccount.AssetSymbol, IsEnabled: true}, &denomination); err != nil {
					logger.Error("Error response from Float manager : %+v while trying to denomination of float asset", err)
					continue
				}
				denominationDecimal := float64(denomination.Decimal)
				//decimal units
				deficitInDecimalUnits := float64(deficit) / math.Pow(10, denominationDecimal)
				deficitInDecimalUnits = math.Round(deficitInDecimalUnits*1000) / 1000
				params := make(map[string]string)
				params["amount"] = fmt.Sprintf("%f", deficitInDecimalUnits)
				coldWalletEmails := []model.EmailUser{}
				coldWalletEmails[0] = model.EmailUser{
					Name:  "Yele",
					Email: config.ColdWalletEmail,
				}
				sendEmailRequest := model.SendEmailRequest{
					Subject: "Please fund Bundle hot wallet address for " + floatAccount.AssetSymbol,
					Content: "",
					Template: model.EmailTemplate{
						ID:     "",
						Params: params,
					},
					Sender: model.EmailUser{
						Name:  "Bundle",
						Email: "info@bundle.africa",
					},
					Receivers: coldWalletEmails,
				}
				sendEmailResponse := model.SendEmailResponse{}
				services.SendEmailNotification(cache, logger, config, sendEmailRequest, &sendEmailResponse, serviceErr)

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
func getTotalUserBalance(repository database.BaseRepository, assetSymbol string, logger *utility.Logger, userAssetRepository database.UserAssetRepository) (int64, error) {
	denomination := dto.Denomination{}
	if err := repository.GetByFieldName(&dto.Denomination{AssetSymbol: assetSymbol, IsEnabled: true}, &denomination); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to denomination of float asset", err)
	}
	sum, err := userAssetRepository.SumAmountField(dto.UserAsset{AssetSymbol: assetSymbol})
	if err != nil {
		return 0, err
	}
	denominationDecimal := float64(denomination.Decimal)
	scaledTotalSum := int64(float64(sum) * math.Pow(10, denominationDecimal))
	return scaledTotalSum, nil
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

func getDepositsSumForAssetFromDate(repository database.BaseRepository, assetSymbol string, logger *utility.Logger, hotWallet dto.HotWalletAsset) (int64, error) {
	deposits := []dto.Transaction{}
	if err := repository.FetchByFieldNameFromDate(dto.Transaction{
		TransactionTag: "DEPOSIT",
		AssetSymbol:    assetSymbol,
	}, &deposits, hotWallet.LastDepositCreatedAt); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get deposits", err)
		return 0, err
	}
	sum := int64(0)
	var lastCreatedAt *time.Time
	for _, deposit := range deposits {
		recipientAsset := dto.UserAsset{}
		getRecipientAsset(repository, deposit.RecipientID, &recipientAsset, logger)
		//convert to native units
		balance, _ := strconv.ParseFloat(deposit.Value, 64)
		denominationDecimal := float64(recipientAsset.Decimal)
		scaledBalance := int64(balance * math.Pow(10, denominationDecimal))
		sum = sum + scaledBalance
		lastCreatedAt = &deposit.CreatedAt
	}
	if lastCreatedAt != nil {
		if err := repository.Update(&hotWallet, &dto.HotWalletAsset{LastDepositCreatedAt: lastCreatedAt}); err != nil {
			logger.Error("Error occured while updating hot wallet LastDepositCreatedAt to On-going : %s", err)
		}
	}
	return sum, nil
}

func getWithdrawalsSumForAssetFromDate(repository database.BaseRepository, assetSymbol string, logger *utility.Logger, hotWallet dto.HotWalletAsset) (int64, error) {
	withdrawals := []dto.Transaction{}
	if err := repository.FetchByFieldNameFromDate(dto.Transaction{
		TransactionTag: "WITHDRAW",
		AssetSymbol:    assetSymbol,
	}, &withdrawals, hotWallet.LastWithdrawalCreatedAt); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get withdrawals", err)
		return 0, err
	}
	var lastCreatedAt *time.Time
	sum := int64(0)
	for _, withdrawal := range withdrawals {
		recipientAsset := dto.UserAsset{}
		getRecipientAsset(repository, withdrawal.InitiatorID, &recipientAsset, logger)
		//convert to native units
		balance, _ := strconv.ParseFloat(withdrawal.Value, 64)
		denominationDecimal := float64(recipientAsset.Decimal)
		scaledBalance := int64(balance * math.Pow(10, denominationDecimal))
		sum = sum + scaledBalance
		lastCreatedAt = &withdrawal.CreatedAt
	}
	if lastCreatedAt != nil {
		if err := repository.Update(&hotWallet, &dto.HotWalletAsset{LastWithdrawalCreatedAt: lastCreatedAt}); err != nil {
			logger.Error("Error occured while updating hot wallet LastWithdrawalCreatedAt to On-going : %s", err)
		}
	}

	return sum, nil
}

func signTxAndBroadcastToChain(cache *utility.MemoryCache, repository database.BaseRepository, amount int64, destinationAddress string, logger *utility.Logger, config Config.Data, floatAccount dto.HotWalletAsset, serviceErr model.ServicesRequestErr) {
	// Calls key-management to sign transaction
	signTransactionRequest := model.SignTransactionRequest{
		FromAddress: floatAccount.Address,
		ToAddress:   destinationAddress,
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

func ExecuteFloatManagerCronJob(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, userAssetRepository database.UserAssetRepository) {
	c := cron.New()
	c.AddFunc(config.FloatCronInterval, func() { ManageFloat(cache, logger, config, repository, userAssetRepository) })
	c.Start()
}
