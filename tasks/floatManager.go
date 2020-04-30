package tasks

import (
	"fmt"
	"github.com/robfig/cron/v3"
	uuid "github.com/satori/go.uuid"
	"math"
	"math/big"
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
	token, err := acquireLock("float", cache, logger, config, serviceErr)
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
		prec := uint(64)
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
		logger.Info("totalUserBalance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, totalUserBalance)
		depositSumFromLastRun, err := getDepositsSumForAssetFromDate(repository, floatAccount.AssetSymbol, logger, floatAccount)
		if err != nil {
			continue
		}
		logger.Info("depositSumFromLastRun for this hot wallet %+v is %+v", floatAccount.AssetSymbol, depositSumFromLastRun)
		withdrawalSumFromLastRun, err := getWithdrawalsSumForAssetFromDate(repository, floatAccount.AssetSymbol, logger, floatAccount)
		if err != nil {
			continue
		}
		logger.Info("withdrawalSumFromLastRun for this hot wallet %+v is %+v", floatAccount.AssetSymbol, withdrawalSumFromLastRun)
		percentageOfUserBalance := big.NewFloat(float64(config.FloatPercentage / 100))
		percentageOfUserBalance.Mul(percentageOfUserBalance, totalUserBalance)
		minimum := new(big.Float)
		minimum.Add(percentageOfUserBalance, big.NewFloat(float64(floatAccount.ReservedBalance)))
		logger.Info("minimum balance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, minimum)
		maximum := new(big.Float)
		differenceOfDepositAndWithdrawals := new(big.Float)
		differenceOfDepositAndWithdrawals.Sub(depositSumFromLastRun, withdrawalSumFromLastRun)
		differenceOfDepositAndWithdrawals.Abs(differenceOfDepositAndWithdrawals)
		maximum.Add(minimum, differenceOfDepositAndWithdrawals)
		logger.Info("maximum balance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, maximum)
		floatOnChainBalance, _ := new(big.Float).SetPrec(prec).SetString(floatOnChainBalanceResponse.Balance)
		logger.Info("floatOnChainBalance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, floatOnChainBalance)
		//it checks if the float balance is below the minimum balance or above the maximum balance
		if floatOnChainBalance.Cmp(minimum) < 0 {
			//if below the minimum balance, it then checks if deposit - withdrawal < 0,
			// then we call binance broker api to fund hot wallet and raise the float balance from
			// it's deficit amount to the maximum amount (residual + % of total user
			// balance + delta(total_deposit - total_withdrawal) since its last run).
			if depositSumFromLastRun.Cmp(withdrawalSumFromLastRun) < 0 {
				binanceAssetBalances := model.BinanceAssetBalances{}
				services.GetOnChainBinanceAssetBalances(cache, logger, config, &binanceAssetBalances, serviceErr)
				for _, coin := range binanceAssetBalances.CoinList {
					if coin.Coin == floatAccount.AssetSymbol {
						//check if balance is enough to fill deficit
						binanceBalance, _ := strconv.ParseFloat(coin.Balance, 64)
						logger.Info("binanceBalance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, binanceBalance)
						denomination := dto.Denomination{}
						if err := repository.GetByFieldName(&dto.Denomination{AssetSymbol: floatAccount.AssetSymbol, IsEnabled: true}, &denomination); err != nil {
							logger.Error("Error response from Float manager : %+v while trying to denomination of float asset", err)
							break
						}
						denominationDecimal := float64(denomination.Decimal)
						scaledBinanceBalance := big.NewFloat(binanceBalance * math.Pow(10, denominationDecimal))
						deficit := new(big.Float)
						deficit.Sub(maximum, floatOnChainBalance)
						//decimal units
						deficitInDecimalUnits := new(big.Float)
						deficitInDecimalUnits.Quo(deficit, big.NewFloat(math.Pow(10, denominationDecimal)))
						logger.Info("deficitInDecimalUnits for this hot wallet %+v is %+v", floatAccount.AssetSymbol, deficitInDecimalUnits)
						var bigIntDeficit *big.Int
						deficit.Int(bigIntDeficit)
						if scaledBinanceBalance.Cmp(deficit) > 0 {
							//Go ahead and withdraw to hotwallet
							logger.Info("Binance balance is higher than deficit for this hot wallet, so withdraawing %+v from binance broker acc %+v ", scaledBinanceBalance, floatAccount.AssetSymbol)
							money := model.Money{
								Value:        bigIntDeficit.String(),
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
							logger.Info("Not enough in this binance wallet %+v, so sending an email to fund hot wallet for amount %+v in decimal units", floatAccount.AssetSymbol, deficitInDecimalUnits)

							params := make(map[string]string)
							params["amount"] = fmt.Sprintf("%f", deficitInDecimalUnits)
							coldWalletEmails := []model.EmailUser{
								model.EmailUser{
									Name:  "Binance Cold wallet user",
									Email: config.ColdWalletEmail,
								},
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
				deficit := new(big.Float)
				deficit.Sub(minimum, floatOnChainBalance)
				denomination := dto.Denomination{}
				if err := repository.GetByFieldName(&dto.Denomination{AssetSymbol: floatAccount.AssetSymbol, IsEnabled: true}, &denomination); err != nil {
					logger.Error("Error response from Float manager : %+v while trying to denomination of float asset", err)
					continue
				}
				denominationDecimal := float64(denomination.Decimal)
				//decimal units
				deficitInDecimalUnits := new(big.Float)
				deficitInDecimalUnits.Quo(deficit, big.NewFloat(math.Pow(10, denominationDecimal)))
				logger.Info("deposit - withdrawal >= 0 %+v, so sending an email to fund hot wallet for amount %+v in decimal units", floatAccount.AssetSymbol, deficitInDecimalUnits)
				var bigIntDeficit *big.Int
				deficit.Int(bigIntDeficit)
				params := make(map[string]string)
				params["amount"] = bigIntDeficit.String()
				coldWalletEmails := []model.EmailUser{
					model.EmailUser{
						Name:  "Binance Cold wallet user",
						Email: config.ColdWalletEmail,
					},
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
		if floatOnChainBalance.Cmp(maximum) > 0 {
			//debit float address
			logger.Info("floatOnChainBalance > maximum, so withdrawing excess %+v %+v to binance brokage", floatOnChainBalance.Sub(floatOnChainBalance, maximum), floatAccount.AssetSymbol)
			depositAddressResponse := model.DepositAddressResponse{}
			var bigIntDeficit *big.Int
			excessDeficit := new(big.Float)
			excessDeficit.Sub(floatOnChainBalance, maximum).Int(bigIntDeficit)
			services.GetDepositAddress(cache, logger, config, floatAccount.AssetSymbol, "", &depositAddressResponse, serviceErr)
			signTxAndBroadcastToChain(cache, repository, bigIntDeficit.Int64(), depositAddressResponse.Address, logger, config, floatAccount, serviceErr)
		}
	}
	if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
		logger.Error("Could not release lock", err)
		return
	}
	logger.Info("Float manager process ends successfully, lock released")
}

//total liability at any given time
func getTotalUserBalance(repository database.BaseRepository, assetSymbol string, logger *utility.Logger, userAssetRepository database.UserAssetRepository) (*big.Float, error) {
	denomination := dto.Denomination{}
	if err := repository.GetByFieldName(&dto.Denomination{AssetSymbol: assetSymbol, IsEnabled: true}, &denomination); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to denomination of float asset", err)
	}
	sum, err := userAssetRepository.SumAmountField(&dto.UserAsset{DenominationID: denomination.ID})
	if err != nil {
		return nil, err
	}
	denominationDecimal := float64(denomination.Decimal)
	scaledTotalSum := big.NewFloat(float64(sum) * math.Pow(10, denominationDecimal))
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
		logger.Error("Error response from Float Manager job : %+v while checking for asset with id %+v", err, recipientAsset.ID)
		return
	}
}

func Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func getDepositsSumForAssetFromDate(repository database.BaseRepository, assetSymbol string, logger *utility.Logger, hotWallet dto.HotWalletAsset) (*big.Float, error) {
	deposits := []dto.Transaction{}
	if err := repository.FetchByFieldNameFromDate(dto.Transaction{
		TransactionTag: "DEPOSIT",
		AssetSymbol:    assetSymbol,
	}, &deposits, hotWallet.LastDepositCreatedAt); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get deposits", err)
		return nil, err
	}

	sum := new(big.Float)
	sum.SetFloat64(0)
	var lastCreatedAt *time.Time
	for _, deposit := range deposits {
		recipientAsset := dto.UserAsset{}
		getRecipientAsset(repository, deposit.RecipientID, &recipientAsset, logger)
		//convert to native units
		balance, _ := strconv.ParseFloat(deposit.Value, 64)
		denominationDecimal := float64(recipientAsset.Decimal)
		scaledBalance := big.NewFloat(balance * math.Pow(10, denominationDecimal))
		sum = sum.Add(sum, scaledBalance)
		lastCreatedAt = &deposit.CreatedAt
	}
	if lastCreatedAt != nil {
		if err := repository.Update(&hotWallet, &dto.HotWalletAsset{LastDepositCreatedAt: lastCreatedAt}); err != nil {
			logger.Error("Error occured while updating hot wallet LastDepositCreatedAt to On-going : %s", err)
		}
	}
	return sum, nil
}

func getWithdrawalsSumForAssetFromDate(repository database.BaseRepository, assetSymbol string, logger *utility.Logger, hotWallet dto.HotWalletAsset) (*big.Float, error) {
	withdrawals := []dto.Transaction{}
	if err := repository.FetchByFieldNameFromDate(dto.Transaction{
		TransactionTag: "WITHDRAW",
		AssetSymbol:    assetSymbol,
	}, &withdrawals, hotWallet.LastWithdrawalCreatedAt); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get withdrawals", err)
		return nil, err
	}
	var lastCreatedAt *time.Time
	sum := new(big.Float)
	sum.SetFloat64(0)
	for _, withdrawal := range withdrawals {
		recipientAsset := dto.UserAsset{}
		getRecipientAsset(repository, withdrawal.InitiatorID, &recipientAsset, logger)
		//convert to native units
		balance, _ := strconv.ParseFloat(withdrawal.Value, 64)
		denominationDecimal := float64(recipientAsset.Decimal)
		scaledBalance := big.NewFloat(balance * math.Pow(10, denominationDecimal))
		sum = sum.Add(sum, scaledBalance)
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
		IsSweep:     false,
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
