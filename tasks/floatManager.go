package tasks

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"time"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"

	"github.com/robfig/cron/v3"
	uuid "github.com/satori/go.uuid"
)

func ManageFloat(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, userAssetRepository database.UserAssetRepository) {
	logger.Info("Float manager process begins")
	serviceErr := dto.ServicesRequestErr{}
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
		request := dto.OnchainBalanceRequest{
			AssetSymbol: floatAccount.AssetSymbol,
			Address:     floatAccount.Address,
		}
		floatOnChainBalanceResponse := dto.OnchainBalanceResponse{}
		services.GetOnchainBalance(cache, logger, config, request, &floatOnChainBalanceResponse, serviceErr)

		//get minimum amount
		totalUserBalance, err := getTotalUserBalance(repository, floatAccount.AssetSymbol, logger, userAssetRepository)
		if err != nil {
			logger.Info("error with float : %+v", err)
			continue
		}
		logger.Info("totalUserBalance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, totalUserBalance)
		depositSumFromLastRun, err := getDepositsSumForAssetFromDate(repository, floatAccount.AssetSymbol, logger, floatAccount)
		if err != nil {
			logger.Info("error with float : %+v", err)
			continue
		}
		logger.Info("depositSumFromLastRun for this hot wallet %+v is %+v", floatAccount.AssetSymbol, depositSumFromLastRun)
		withdrawalSumFromLastRun, err := getWithdrawalsSumForAssetFromDate(repository, floatAccount.AssetSymbol, logger, floatAccount)
		if err != nil {
			logger.Info("error with float : %+v", err)
			continue
		}
		logger.Info("withdrawalSumFromLastRun for this hot wallet %+v is %+v", floatAccount.AssetSymbol, withdrawalSumFromLastRun)

		maxUserBalance, err := GetMaxUserBalanceFor(userAssetRepository, floatAccount.AssetSymbol)
		if err != nil {
			logger.Info("Error getting maximum user balance for %s : %+v", floatAccount.AssetSymbol, err)
			continue
		}
		logger.Info("maximum user balanace is %+v", maxUserBalance)

		minPercentageOfMaxUserBalance := new(big.Float)
		minPercentageOfMaxUserBalance.Quo(big.NewFloat(utility.FLOAT_MINPercentageOfMaxUserBalance), big.NewFloat(float64(100)))
		logger.Info("minimum percentage of maximum user balance used is %+v", minPercentageOfMaxUserBalance)
		maxPercentageOfMaxUserBalance := new(big.Float)
		maxPercentageOfMaxUserBalance.Quo(big.NewFloat(utility.FLOAT_MAXPercentageOfMaxUserBalance), big.NewFloat(float64(100)))
		logger.Info("maximum percentage of maximum users balance used is %+v", maxPercentageOfMaxUserBalance)
		minPercentageOfTotalUserBalance := new(big.Float)
		minPercentageOfTotalUserBalance.Quo(big.NewFloat(utility.FLOAT_MINPercentageValueOfTotalUserBalance), big.NewFloat(float64(100)))
		logger.Info("minimum percentage value of total user balance used is %+v", minPercentageOfTotalUserBalance)
		averagePercentageOfTotalUserBalance := new(big.Float)
		averagePercentageOfTotalUserBalance.Quo(big.NewFloat(utility.FLOAT_AVERAGEPercentageOfTotalUserBalance), big.NewFloat(float64(100)))
		logger.Info("average percentage of total users balance used is %+v", averagePercentageOfTotalUserBalance)
		maxPercentageOfTotalUserBalance := new(big.Float)
		maxPercentageOfTotalUserBalance.Quo(big.NewFloat(utility.FLOAT_MAXPercentageOfTotalUserBalance), big.NewFloat(float64(100)))
		logger.Info("maximum percentage of total users balance used is %+v", maxPercentageOfTotalUserBalance)

		minPercentageValueOfMaxUserBalance := new(big.Float)
		maxPercentageValueOfMaxUserBalance := new(big.Float)
		minPercentageValueOfTotalUserBalance := new(big.Float)
		averagePercentageValueOfTotalUserBalance := new(big.Float)
		maxPercentageValueOfTotalUserBalance := new(big.Float)

		minPercentageValueOfMaxUserBalance.Mul(minPercentageOfMaxUserBalance, maxUserBalance)
		logger.Info("minimum percentage value of maximum users balance is %+v", minPercentageValueOfMaxUserBalance)
		maxPercentageValueOfMaxUserBalance.Mul(maxPercentageOfMaxUserBalance, maxUserBalance)
		logger.Info("maximum percentage value of maximum users balance is %+v", maxPercentageValueOfMaxUserBalance)
		minPercentageValueOfTotalUserBalance.Mul(minPercentageOfTotalUserBalance, totalUserBalance)
		logger.Info("minimum percentage value of total users balance is %+v", minPercentageValueOfTotalUserBalance)
		averagePercentageValueOfTotalUserBalance.Mul(averagePercentageOfTotalUserBalance, totalUserBalance)
		logger.Info("avearage percentage value of total users balance is %+v", averagePercentageValueOfTotalUserBalance)
		maxPercentageValueOfTotalUserBalance.Mul(maxPercentageOfTotalUserBalance, totalUserBalance)
		logger.Info("maximum percentage value of total users balance is %+v", maxPercentageValueOfTotalUserBalance)
		A := minPercentageValueOfTotalUserBalance.Add(minPercentageValueOfTotalUserBalance, maxPercentageValueOfMaxUserBalance)
		C := utility.MinFloat(A, totalUserBalance)

		minimumFloatBalance := utility.MaxFloat(averagePercentageValueOfTotalUserBalance, minPercentageValueOfMaxUserBalance)
		logger.Info("minimum balance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, minimumFloatBalance)
		maximumFloatBalance := utility.MaxFloat(maxPercentageValueOfTotalUserBalance, C)
		logger.Info("maximum balance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, maximumFloatBalance)

		differenceOfDepositAndWithdrawals := new(big.Float)
		differenceOfDepositAndWithdrawals.Sub(depositSumFromLastRun, withdrawalSumFromLastRun)
		differenceOfDepositAndWithdrawals.Abs(differenceOfDepositAndWithdrawals)
		floatOnChainBalance, _ := new(big.Float).SetPrec(prec).SetString(floatOnChainBalanceResponse.Balance)
		logger.Info("floatOnChainBalance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, floatOnChainBalance)
		deficit := new(big.Float)
		floatAction := ""
		//it checks if the float balance is below the minimum balance or above the maximum balance
		if floatOnChainBalance.Cmp(minimumFloatBalance) < 0 {
			//if below the minimum balance, it then checks if deposit - withdrawal < 0,
			// then we call binance broker api to fund hot wallet and raise the float balance from
			// it's deficit amount to the maximum amount (residual + % of total user
			// balance + delta(total_deposit - total_withdrawal) since its last run).
			if depositSumFromLastRun.Cmp(withdrawalSumFromLastRun) < 0 {
				denomination := model.Denomination{}
				if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: floatAccount.AssetSymbol, IsEnabled: true}, &denomination); err != nil {
					logger.Error("Error response from Float manager : %+v while trying to denomination of float asset", err)
					break
				}
				denominationDecimal := float64(denomination.Decimal)
				deficit.Sub(maximumFloatBalance, floatOnChainBalance)
				//decimal units
				deficitInDecimalUnits := new(big.Float)
				deficitInDecimalUnits.Quo(deficit, big.NewFloat(math.Pow(10, denominationDecimal)))
				logger.Info("deficitInDecimalUnits for this hot wallet %+v is %+v", floatAccount.AssetSymbol, deficitInDecimalUnits)
				var bigIntDeficit *big.Int
				deficit.Int(bigIntDeficit)
				//trigger alert to cold wallet user
				floatAction = fmt.Sprintf("sending an email to fund hot wallet for amount %+v in decimal units", floatAccount.AssetSymbol, deficitInDecimalUnits)
				logger.Info(floatAction)

				params := map[string]string{
					"amount":      deficitInDecimalUnits.String(),
					"assetSymbol": floatAccount.AssetSymbol,
				}
				err = notifyColdWalletUsers("Fund", params, config, err, cache, logger, serviceErr)
			} else {
				//But if it then checks if deposit - withdrawal >= 0, then we trigger call to cold wallet
				// using notification service to raise the float balance from it's deficit amount to
				// or above the minimum amount (residual amount)
				deficit.Sub(minimumFloatBalance, floatOnChainBalance)
				denomination := model.Denomination{}
				if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: floatAccount.AssetSymbol, IsEnabled: true}, &denomination); err != nil {
					logger.Error("Error response from Float manager : %+v while trying to denomination of float asset", err)
					continue
				}
				denominationDecimal := float64(denomination.Decimal)
				//decimal units
				deficitInDecimalUnits := new(big.Float)
				deficitInDecimalUnits.Quo(deficit, big.NewFloat(math.Pow(10, denominationDecimal)))
				floatAction = fmt.Sprintf("deposit - withdrawal >= 0 %+v, so sending an email to fund hot wallet for amount %+v in decimal units", floatAccount.AssetSymbol, deficitInDecimalUnits)
				logger.Info(floatAction)
				params := map[string]string{
					"amount":      deficitInDecimalUnits.String(),
					"assetSymbol": floatAccount.AssetSymbol,
				}
				err = notifyColdWalletUsers("Fund", params, config, err, cache, logger, serviceErr)
			}
		}
		if floatOnChainBalance.Cmp(maximumFloatBalance) > 0 {
			//debit float address
			logger.Info("floatOnChainBalance > maximum, so withdrawing excess %+v %+v to binance brokage", floatOnChainBalance.Sub(floatOnChainBalance, maximumFloatBalance), floatAccount.AssetSymbol)
			depositAddressResponse := dto.DepositAddressResponse{}
			var bigIntDeficit *big.Int
			excessDeficit := new(big.Float)
			excessDeficit.Sub(floatOnChainBalance, maximumFloatBalance)
			bigIntDeficit, _ = excessDeficit.Int(nil)
			denomination := model.Denomination{}
			if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: floatAccount.AssetSymbol, IsEnabled: true}, &denomination); err != nil {
				logger.Error("Error response from Float manager : %+v while trying to denomination of float asset", err)
				continue
			}
			//Pass network as maincoin in the case of tokens
			if denomination.IsToken {
				services.GetDepositAddress(cache, logger, config, floatAccount.AssetSymbol, denomination.MainCoinAssetSymbol, &depositAddressResponse, serviceErr)
			} else {
				services.GetDepositAddress(cache, logger, config, floatAccount.AssetSymbol, "", &depositAddressResponse, serviceErr)
			}
			signTxAndBroadcastToChain(cache, repository, bigIntDeficit, depositAddressResponse, logger, config, floatAccount, serviceErr)
			surplusInDecimalUnits := new(big.Float)
			denominationDecimal := float64(denomination.Decimal)
			surplusInDecimalUnits.Quo(excessDeficit, big.NewFloat(math.Pow(10, denominationDecimal)))
			params := map[string]string{
				"amount":             surplusInDecimalUnits.String(),
				"assetSymbol":        floatAccount.AssetSymbol,
				"depositAddress":     depositAddressResponse.Address,
				"depositAddressMemo": depositAddressResponse.Tag,
			}
			err = notifyColdWalletUsers("Withdraw", params, config, err, cache, logger, serviceErr)
		}

		if err := saveFloatVariables(repository, logger, depositSumFromLastRun, totalUserBalance, withdrawalSumFromLastRun, floatOnChainBalance, maximumFloatBalance, minimumFloatBalance, deficit, float64(floatAccount.ReservedBalance), floatAction, floatAccount.AssetSymbol); err != nil {
			logger.Error("Error with creating saving float manager run variables for %s : %s", floatAccount.AssetSymbol, err)
		}

	}
	if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
		logger.Error("Could not release lock", err)
		return
	}
	logger.Info("Float manager process ends successfully, lock released")
}

//save float variables to db
func saveFloatVariables(repository database.BaseRepository, logger *utility.Logger, depositSumFromLastRun, totalUserBalance, withdrawalSumFromLastRun, floatOnChainBalance, maximum, minimum, deficit *big.Float, reservedBalance float64, floatAction, assetSymbol string) error {
	DepositSum, _ := depositSumFromLastRun.Float64()
	ResidualAmount := reservedBalance
	TotalUserBalance, _ := totalUserBalance.Float64()
	WithdrawalSum, _ := withdrawalSumFromLastRun.Float64()
	FloatOnChainBalance, _ := floatOnChainBalance.Float64()
	MaximumFloatRange, _ := maximum.Float64()
	MinimumFloatRange, _ := minimum.Float64()
	Deficit, _ := deficit.Float64()

	if err := repository.Create(&model.FloatManager{ResidualAmount: ResidualAmount, AssetSymbol: assetSymbol, TotalUserBalance: TotalUserBalance, DepositSum: DepositSum, WithdrawalSum: WithdrawalSum, FloatOnChainBalance: FloatOnChainBalance, MaximumFloatRange: MaximumFloatRange, MinimumFloatRange: MinimumFloatRange, Deficit: Deficit, Action: floatAction, LastRunTime: time.Now()}); err != nil {
		return err
	}
	return nil
}

func notifyColdWalletUsers(emailType string, params map[string]string, config Config.Data, err error, cache *utility.MemoryCache, logger *utility.Logger, serviceErr dto.ServicesRequestErr) error {
	coldWalletEmails := []dto.EmailUser{
		dto.EmailUser{
			Name:  "Binance Cold wallet user",
			Email: config.ColdWalletEmail,
		},
	}
	sendEmailRequest := dto.SendEmailRequest{
		Sender: dto.EmailUser{
			Name:  "Bundle",
			Email: "info@bundle.africa",
		},
		Receivers: coldWalletEmails,
	}

	switch emailType {
	case "Fund":
		if config.SENTRY_ENVIRONMENT == utility.ENV_PRODUCTION {
			sendEmailRequest.Subject = "Live: Please fund Bundle hot wallet address for " + params["assetSymbol"]
		} else {
			sendEmailRequest.Subject = "Test: Please fund Bundle hot wallet address for " + params["assetSymbol"]
		}
		sendEmailRequest.Template = dto.EmailTemplate{
			ID:     config.ColdWalletEmailTemplateId,
			Params: params,
		}
	case "Withdraw":
		if config.SENTRY_ENVIRONMENT == utility.ENV_PRODUCTION {
			sendEmailRequest.Subject = "Live: Withdrawing excess funds to brokerage for " + params["assetSymbol"]
		} else {
			sendEmailRequest.Subject = "Test: Withdrawing excess funds to brokerage for " + params["assetSymbol"]
		}
		sendEmailRequest.Content = fmt.Sprintf(`
		Attention:
		To regulate float account, %+v %s has been moved from the HotWallet Address to the Brokerage Account Address %s with Memo (%s).
		Please check to verify that movement was successful.
		`, params["amount"], params["assetSymbol"], params["depositAddress"], params["depositAddressMemo"])
	}

	sendEmailResponse := dto.SendEmailResponse{}
	err = services.SendEmailNotification(cache, logger, config, sendEmailRequest, &sendEmailResponse, serviceErr)
	if err != nil {
		logger.Info("An error occurred while sending email notification to cold wallet user %+v", err.Error())
	}
	return err
}

//total liability at any given time
func getTotalUserBalance(repository database.BaseRepository, assetSymbol string, logger *utility.Logger, userAssetRepository database.UserAssetRepository) (*big.Float, error) {
	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: assetSymbol, IsEnabled: true}, &denomination); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to denomination of float asset", err)
	}
	sum, err := userAssetRepository.SumAmountField(&model.UserAsset{DenominationID: denomination.ID})
	if err != nil {
		return nil, err
	}
	denominationDecimal := float64(denomination.Decimal)
	scaledTotalSum := big.NewFloat(float64(sum) * math.Pow(10, denominationDecimal))
	return scaledTotalSum, nil
}

func getFloatAccounts(repository database.BaseRepository, logger *utility.Logger) ([]model.HotWalletAsset, error) {
	//Get the float address
	floatAccounts := []model.HotWalletAsset{}
	if err := repository.Fetch(&floatAccounts); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get float balances", err)
		return nil, err
	}
	return floatAccounts, nil
}
func getRecipientAsset(repository database.BaseRepository, assetId uuid.UUID, recipientAsset *model.UserAsset, logger *utility.Logger) {
	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	if err := userAssetRepository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetId}}, &recipientAsset); err != nil {
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

func getDepositsSumForAssetFromDate(repository database.BaseRepository, assetSymbol string, logger *utility.Logger, hotWallet model.HotWalletAsset) (*big.Float, error) {
	deposits := []model.Transaction{}
	if err := repository.FetchByFieldNameFromDate(model.Transaction{
		TransactionTag: "DEPOSIT",
		AssetSymbol:    assetSymbol,
	}, &deposits, hotWallet.LastDepositCreatedAt); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get deposits", err)
		return nil, err
	}

	sum := new(big.Float)
	sum.SetFloat64(0)
	var lastCreatedAt *time.Time
	//sort deposits by creation date asc
	sort.Slice(deposits, func(i, j int) bool {
		return deposits[i].BaseModel.CreatedAt.Before(deposits[j].BaseModel.CreatedAt)
	})
	for _, deposit := range deposits {
		recipientAsset := model.UserAsset{}
		getRecipientAsset(repository, deposit.RecipientID, &recipientAsset, logger)
		//convert to native units
		balance, _ := strconv.ParseFloat(deposit.Value, 64)
		denominationDecimal := float64(recipientAsset.Decimal)
		scaledBalance := big.NewFloat(balance * math.Pow(10, denominationDecimal))
		sum = sum.Add(sum, scaledBalance)
		lastCreatedAt = &deposit.CreatedAt
	}
	if lastCreatedAt != nil {
		if err := repository.Update(&hotWallet, &model.HotWalletAsset{LastDepositCreatedAt: lastCreatedAt}); err != nil {
			logger.Error("Error occured while updating hot wallet LastDepositCreatedAt to On-going : %s", err)
		}
	}
	return sum, nil
}

func getWithdrawalsSumForAssetFromDate(repository database.BaseRepository, assetSymbol string, logger *utility.Logger, hotWallet model.HotWalletAsset) (*big.Float, error) {
	withdrawals := []model.Transaction{}
	if err := repository.FetchByFieldNameFromDate(model.Transaction{
		TransactionTag: "WITHDRAW",
		AssetSymbol:    assetSymbol,
	}, &withdrawals, hotWallet.LastWithdrawalCreatedAt); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get withdrawals", err)
		return nil, err
	}
	var lastCreatedAt *time.Time
	sum := new(big.Float)
	sum.SetFloat64(0)
	//sort withdrawals by creation date asc
	sort.Slice(withdrawals, func(i, j int) bool {
		return withdrawals[i].BaseModel.CreatedAt.Before(withdrawals[j].BaseModel.CreatedAt)
	})
	for _, withdrawal := range withdrawals {
		recipientAsset := model.UserAsset{}
		getRecipientAsset(repository, withdrawal.InitiatorID, &recipientAsset, logger)
		//convert to native units
		balance, _ := strconv.ParseFloat(withdrawal.Value, 64)
		denominationDecimal := float64(recipientAsset.Decimal)
		scaledBalance := big.NewFloat(balance * math.Pow(10, denominationDecimal))
		sum = sum.Add(sum, scaledBalance)
		lastCreatedAt = &withdrawal.CreatedAt
	}
	if lastCreatedAt != nil {
		if err := repository.Update(&hotWallet, &model.HotWalletAsset{LastWithdrawalCreatedAt: lastCreatedAt}); err != nil {
			logger.Error("Error occured while updating hot wallet LastWithdrawalCreatedAt to On-going : %s", err)
		}
	}

	return sum, nil
}

func signTxAndBroadcastToChain(cache *utility.MemoryCache, repository database.BaseRepository, amount *big.Int, depositAccount dto.DepositAddressResponse, logger *utility.Logger, config Config.Data, floatAccount model.HotWalletAsset, serviceErr dto.ServicesRequestErr) {
	// Calls key-management to sign transaction
	signTransactionRequest := dto.SignTransactionRequest{
		FromAddress: floatAccount.Address,
		ToAddress:   depositAccount.Address,
		Memo:        depositAccount.Tag,
		Amount:      amount,
		AssetSymbol: floatAccount.AssetSymbol,
		IsSweep:     false,
	}
	signTransactionResponse := dto.SignTransactionResponse{}
	if err := services.SignTransaction(cache, logger, config, signTransactionRequest, &signTransactionResponse, serviceErr); err != nil {
		logger.Error("Error response from float manager : %+v. While signing transaction to debit float for %+v", err, floatAccount.AssetSymbol)
		return
	}
	//need an empty array to be able to reuse the method broadcastAndCompleteFloatTx
	emptyArrayOfTransactions := []model.Transaction{}
	err, _ := broadcastAndCompleteFloatTx(signTransactionResponse, config, floatAccount.AssetSymbol, cache, logger, serviceErr, emptyArrayOfTransactions, repository)
	if err != nil {
		logger.Error("Error response from float manager while broadcast transaction to debit float for %+v : %+v, additional context : %+v", floatAccount.AssetSymbol, err, serviceErr)
		return
	}
}

func ExecuteFloatManagerCronJob(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, userAssetRepository database.UserAssetRepository) {
	c := cron.New()
	c.AddFunc(config.FloatCronInterval, func() { ManageFloat(cache, logger, config, repository, userAssetRepository) })
	c.Start()
}

func broadcastAndCompleteFloatTx(signTransactionResponse dto.SignTransactionResponse, config Config.Data, symbol string, cache *utility.MemoryCache, logger *utility.Logger, serviceErr dto.ServicesRequestErr, assetTransactions []model.Transaction, repository database.BaseRepository) (error, bool) {
	// Send the signed data to crypto adapter to send to chain
	broadcastToChainRequest := dto.BroadcastToChainRequest{
		SignedData:  signTransactionResponse.SignedData,
		AssetSymbol: symbol,
		ProcessType: utility.FLOATPROCESS,
	}
	broadcastToChainResponse := dto.BroadcastToChainResponse{}
	if err := services.BroadcastToChain(cache, logger, config, broadcastToChainRequest, &broadcastToChainResponse, serviceErr); err != nil {
		logger.Error("Error response from Sweep job : %+v while broadcasting to chain", err)
		return err, true
	}
	//update all assetTransactions with new swept status
	var assetIdList []uuid.UUID
	for _, tx := range assetTransactions {
		assetIdList = append(assetIdList, tx.ID)
	}
	if err := repository.BulkUpdateTransactionSweptStatus(assetIdList); err != nil {
		logger.Error("Error response from Sweep job : %+v while broadcasting to chain", err)
		return err, true
	}
	return nil, false
}

func GetMaxUserBalanceFor(repository database.UserAssetRepository, assetType string) (*big.Float, error) {

	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: assetType, IsEnabled: true}, &denomination); err != nil {
		return big.NewFloat(0), err
	}

	maxUserBalance, err := repository.GetMaxUserBalance(denomination.ID)
	if err != nil {
		return big.NewFloat(0), err
	}
	return big.NewFloat(maxUserBalance), nil
}
