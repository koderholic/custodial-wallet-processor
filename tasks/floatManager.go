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

	"wallet-adapter/errorcode"

	"github.com/robfig/cron/v3"
	uuid "github.com/satori/go.uuid"
)

func ManageFloat(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, userAssetRepository database.UserAssetRepository) {
	logger.Info("Float manager process begins")

	serviceErr := dto.ServicesRequestErr{}
	token, err := AcquireLock("float", utility.SIX_HUNDRED_MILLISECONDS, cache, logger, config, serviceErr)
	if err != nil {
		logger.Error("Could not acquire lock", err)
		return
	}

	floatAccounts, err := GetFloatAccounts(repository, logger)
	if err != nil {
		return
	}

	for _, floatAccount := range floatAccounts {

		prec := uint(64)
		floatDeficit := new(big.Float)
		floatSurplus := new(big.Float)
		floatAction := ""

		userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
		floatNetworkAsset, err := services.GetNetworkByAssetAndNetwork(&userAssetRepository, floatAccount.Network, floatAccount.AssetSymbol)
		if err != nil {
			logger.Error(fmt.Sprintf("error with getting network asset for assetSymbol : %s and network : %s : %s", floatAccount.AssetSymbol, floatAccount.Network, err))
			continue
		}

		// Get float chain balance
		onchainBalanceRequest := dto.OnchainBalanceRequest{
			AssetSymbol: floatAccount.AssetSymbol,
			Network: floatAccount.Network,
			Address:     floatAccount.Address,
		}
		floatOnChainBalanceResponse := dto.OnchainBalanceResponse{}
		if err := services.GetOnchainBalance(cache, logger, config, onchainBalanceRequest, &floatOnChainBalanceResponse, serviceErr); err != nil {
			logger.Error(fmt.Sprintf("error with getting float on-chain balance for %+v is %+v", floatAccount.AssetSymbol, err))
			continue
		}
		floatOnChainBalance, _ := new(big.Float).SetPrec(prec).SetString(floatOnChainBalanceResponse.Balance)
		logger.Info("floatOnChainBalance for this hot wallet %+v is %+v", floatAccount.AssetSymbol, floatOnChainBalance)

		// Get total users balance
		totalUserBalance, err := GetTotalUserBalance(repository, floatAccount.AssetSymbol, floatNetworkAsset.NativeDecimals, logger, userAssetRepository)
		if err != nil {
			logger.Info("error with float : %+v", err)
			continue
		}
		logger.Info("totalUserBalance for this hot wallet %s is %+v", floatAccount.AssetSymbol, totalUserBalance)

		// Get total deposit sum from the last run of this job
		depositSumFromLastRun, err := getDepositsSumForAssetFromDate(repository, floatAccount.AssetSymbol, floatAccount.Network, logger, floatAccount)
		if err != nil {
			logger.Info("error with float manager process, while trying to get the total deposit sum from last run : %+v", err)
		}
		logger.Info("depositSumFromLastRun for this hot wallet %s (%s) is %+v", floatAccount.AssetSymbol, floatAccount.Network, depositSumFromLastRun)

		// Get total withdrawal sum from the last run of this job
		withdrawalSumFromLastRun, err := getWithdrawalsSumForAssetFromDate(repository, floatAccount.AssetSymbol, floatAccount.Network, logger, floatAccount)
		if err != nil {
			logger.Info("error with float manager process, while trying to get the total withdrawal sum from last run : %+v", err)
		}
		logger.Info("withdrawalSumFromLastRun for this hot wallet %s (%+v') is %+v", floatAccount.AssetSymbol,floatAccount.Network, withdrawalSumFromLastRun)

		// Get the maximum user balance of this float asset type
		maxUserBalance, err := GetMaxUserBalanceFor(userAssetRepository, floatAccount.AssetSymbol)
		if err != nil {
			logger.Info("Error with float manager process, while getting maximum user balance for %s : %+v", floatAccount.AssetSymbol, err)
			continue
		}
		logger.Info("maximum user balanace for asset %s is %+v", floatAccount.AssetSymbol, maxUserBalance)

		// Get float manager parameters to calculate minimum and maximum float range
		floatManagerParams, err := getFloatParamFor(floatAccount.AssetSymbol, floatAccount.Network, repository, logger)
		if err != nil {
			logger.Info("Error getting float manager params : %s", err)
			continue
		}

		// GetMinimum
		minimumFloatBalance := GetMinFloatBalance(floatManagerParams, logger, totalUserBalance, maxUserBalance)
		minimumTriggerLevel := new(big.Float)
		minimumTriggerLevel.Mul(big.NewFloat(floatManagerParams.PercentMinimumTriggerLevel), minimumFloatBalance)
		logger.Info("minimum balance for this hot wallet %+v is %+v and minimum trigger amount is %v", floatAccount.AssetSymbol, minimumFloatBalance, minimumTriggerLevel)

		// GetMaximum
		maximumFloatBalance := GetMaxFloatBalance(floatManagerParams, logger, totalUserBalance, maxUserBalance)
		maximumTriggerLevel := new(big.Float)
		maximumTriggerLevel.Mul(big.NewFloat(floatManagerParams.PercentMaximumTriggerLevel), maximumFloatBalance)
		logger.Info("maximum balance for this hot wallet %+v is %+v and maximum trigger amount is %v", floatAccount.AssetSymbol, maximumFloatBalance, maximumTriggerLevel)

		differenceOfDepositAndWithdrawals := new(big.Float)
		differenceOfDepositAndWithdrawals.Sub(depositSumFromLastRun, withdrawalSumFromLastRun)
		differenceOfDepositAndWithdrawals.Abs(differenceOfDepositAndWithdrawals)

		//it checks if the float balance is below or equal to the minimum trigger level
		if floatOnChainBalance.Cmp(minimumTriggerLevel) <= 0 {

			logger.Info("floatOnChainBalance < minimumFloatBalance")

			floatDeficitInDecimalUnits := new(big.Float)
			denominationDecimal := float64(floatNetworkAsset.NativeDecimals)

			if depositSumFromLastRun.Cmp(withdrawalSumFromLastRun) < 0 {
				//if below the minimum trigger level, it then checks if total deposit is less than total withdrawal, for this it raises float back to the maximum value, since there is a pattern of high withdrawal than deposit, float will need maximum funds
				floatDeficit.Sub(maximumFloatBalance, floatOnChainBalance)
			} else {
				// Total deposit is greater than total withdrawal, for this it raises float back to the minimum value plus a certain percentage, sinces there is a higher deposit rate, having a little above the minimum float balance in float would be sufficient.
				floatDeficit.Sub(minimumFloatBalance, floatOnChainBalance)
			}

			floatDeficitInDecimalUnits.Quo(floatDeficit, big.NewFloat(math.Pow(10, denominationDecimal)))
			logger.Info("deficitInDecimalUnits for this hot wallet %s is %+v", floatAccount.AssetSymbol, floatDeficitInDecimalUnits)

			// Ensure email has not been sent already
			emailSent, _ := IsSentColdWalletMail(repository, floatDeficit, floatAccount.AssetSymbol, floatAccount.Network)
			if !emailSent {

				floatAction = fmt.Sprintf("floatOnChainBalance <= minimumFloatBalance %+v - %s, so sending an email to fund hot wallet for amount %+v in decimal units", floatAccount.AssetSymbol, floatAccount.Network, floatDeficitInDecimalUnits)

				params := map[string]string{
					"amount":      floatDeficitInDecimalUnits.String(),
					"assetSymbol": floatAccount.AssetSymbol,
					"network": floatAccount.Network,
				}
				err = notifyColdWalletUsers("Fund", params, config, err, cache, logger, serviceErr)
			}
			floatAction = fmt.Sprintf("Email to fund hot wallet (%s) of %+v has already been sent", floatAccount.AssetSymbol, floatDeficitInDecimalUnits)
			logger.Info(floatAction)
		}

		// If float balance is greater than the maximum float balance, it moves excess funds to the binance broker account
		if floatOnChainBalance.Cmp(maximumFloatBalance) > 0 {

			floatSurplus := new(big.Float)
			floatSurplusInBigInt := new(big.Int)
			floatSurplusInDecimal := new(big.Float)
			floatSurplus.Sub(floatOnChainBalance, maximumFloatBalance)
			floatSurplusInBigInt, _ = floatSurplus.Int(nil)
			if floatSurplus.Cmp(maximumTriggerLevel) < 0 {
				continue
			}
			logger.Info("floatOnChainBalance > maximum, so withdrawing excess %+v %+v to binance brokage", floatSurplus, floatAccount.AssetSymbol)

			// Get binance broker deposit address, pass network as maincoin in the case of tokens
			depositAddressResponse := dto.DepositAddressResponse{}

			if *floatNetworkAsset.IsToken {
				if err := services.GetDepositAddress(cache, logger, config, floatAccount.AssetSymbol, floatNetworkAsset.NativeAsset, &depositAddressResponse, serviceErr); err != nil {
					logger.Error("Error response from Float manager : %+v while trying to get brokerage deposit ", err)
					continue
				}
			} else {
				if err := services.GetDepositAddress(cache, logger, config, floatAccount.AssetSymbol, "", &depositAddressResponse, serviceErr); err != nil {
					logger.Error("Error response from Float manager : %+v while trying to get brokerage deposit ", err)
					continue
				}
			}

			// Sign and send transaction to chain
			if err := sendSingleTransactionToChain(cache, repository, floatSurplusInBigInt, depositAddressResponse, logger, config, floatAccount, serviceErr); err != nil {
				continue
			}

			// Send email to cold wallet recipients
			floatSurplusInDecimal.Quo(floatSurplus, big.NewFloat(math.Pow(10, float64(floatNetworkAsset.NativeDecimals))))
			params := map[string]string{
				"amount":             floatSurplusInDecimal.String(),
				"assetSymbol":        floatAccount.AssetSymbol,
				"depositAddress":     depositAddressResponse.Address,
				"depositAddressMemo": depositAddressResponse.Tag,
				"network": floatAccount.Network,
			}
			err = notifyColdWalletUsers("Withdraw", params, config, err, cache, logger, serviceErr)

		}

		if err := saveFloatVariables(repository, logger, depositSumFromLastRun, totalUserBalance, withdrawalSumFromLastRun, floatOnChainBalance, maximumFloatBalance, minimumFloatBalance, floatDeficit, floatSurplus, float64(floatAccount.ReservedBalance), floatAction, floatAccount.AssetSymbol, floatAccount.Network); err != nil {
			logger.Error("Error with saving float manager run variables for %s : %s", floatAccount.AssetSymbol, err)
		}

	}

	if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
		logger.Error("Could not release lock", err)
		return
	}
	logger.Info("Float manager process ends successfully, lock released")

}

//save float variables to db
func saveFloatVariables(repository database.BaseRepository, logger *utility.Logger, depositSumFromLastRun, totalUserBalance, withdrawalSumFromLastRun, floatOnChainBalance, maximum, minimum, deficit *big.Float, surplus *big.Float, reservedBalance float64, floatAction, assetSymbol, network string) error {
	DepositSum, _ := depositSumFromLastRun.Float64()
	ResidualAmount := reservedBalance
	TotalUserBalance, _ := totalUserBalance.Float64()
	WithdrawalSum, _ := withdrawalSumFromLastRun.Float64()
	FloatOnChainBalance, _ := floatOnChainBalance.Float64()
	MaximumFloatRange, _ := maximum.Float64()
	MinimumFloatRange, _ := minimum.Float64()
	Deficit, _ := deficit.Float64()
	Surplus, _ := surplus.Float64()

	if err := repository.Create(&model.FloatManager{ResidualAmount: ResidualAmount, AssetSymbol: assetSymbol, Network: network, TotalUserBalance: TotalUserBalance, DepositSum: DepositSum, WithdrawalSum: WithdrawalSum, FloatOnChainBalance: FloatOnChainBalance, MaximumFloatRange: MaximumFloatRange, MinimumFloatRange: MinimumFloatRange, Deficit: Deficit, Surplus: Surplus, Action: floatAction, LastRunTime: time.Now()}); err != nil {
		return err
	}
	return nil
}

func NotifyColdWalletUsersViaSMS(amount big.Int, assetSymbol, network string, config Config.Data, cache *utility.MemoryCache, logger *utility.Logger, serviceErr dto.ServicesRequestErr, repository database.BaseRepository) {

	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	floatNetworkAsset, err := services.GetNetworkByAssetAndNetwork(&userAssetRepository, network, assetSymbol)
	if  err != nil {
		logger.Error("Error response from NotifyColdWalletUsersViaSMS : %+v while trying to denomination of float asset", err)
	}
	decimalBalance := ConvertBigIntToDecimalUnit(amount, floatNetworkAsset)
	//send sms
	if config.EnableFloatManager {
		if _, err := AcquireLock(errorcode.INSUFFICIENT_BALANCE_FLOAT_SEND_SMS+utility.SEPERATOR+assetSymbol+utility.SEPERATOR+network, utility.ONE_HOUR_MILLISECONDS, cache, logger, config, serviceErr); err == nil {
			//lock was successfully acquired
			services.BuildAndSendSms(assetSymbol, network, decimalBalance, cache, logger, config, serviceErr)
		}
	}
}

func ConvertBigIntToDecimalUnit(amount big.Int, floatNetworkAsset model.Network) *big.Float {
	amountInFloat, _ := strconv.ParseFloat(amount.String(), 64)
	amountInBigFloat := big.NewFloat(amountInFloat)
	decimalBalance := amountInBigFloat.Quo(amountInBigFloat, big.NewFloat(math.Pow(10, float64(floatNetworkAsset.NativeDecimals))))
	return decimalBalance
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
			sendEmailRequest.Subject = "Live: Please fund Bundle hot wallet address for " + params["assetSymbol"] + " - "+params["network"]
			params["subject"] = sendEmailRequest.Subject
		} else {
			sendEmailRequest.Subject = "Test: Please fund Bundle hot wallet address for " + params["assetSymbol"] + " - "+params["network"]
			params["subject"] = sendEmailRequest.Subject
		}
		sendEmailRequest.Template = dto.EmailTemplate{
			ID:     config.ColdWalletEmailTemplateId,
			Params: params,
		}
	case "Withdraw":
		if config.SENTRY_ENVIRONMENT == utility.ENV_PRODUCTION {
			sendEmailRequest.Subject = "Live: Withdrawing excess funds to brokerage for " + params["assetSymbol"] + " - "+params["network"]
		} else {
			sendEmailRequest.Subject = "Test: Withdrawing excess funds to brokerage for " + params["assetSymbol"] + " - "+params["network"]
		}
		sendEmailRequest.Content = fmt.Sprintf(`
		Attention:
		To regulate float account, %+v %s - %s has been moved from the HotWallet Address to the Brokerage Account Address %s with Memo (%s).
		Please check to verify that movement was successful.
		`, params["amount"], params["assetSymbol"], params["network"], params["depositAddress"], params["depositAddressMemo"])
	}

	sendEmailResponse := dto.SendEmailResponse{}
	err = services.SendEmailNotification(cache, logger, config, sendEmailRequest, &sendEmailResponse, serviceErr)
	if err != nil {
		logger.Info("An error occurred while sending email notification to cold wallet user %+v", err.Error())
	}
	return err
}

//total liability at any given time
func GetTotalUserBalance(repository database.BaseRepository, assetSymbol string, floatDecimal int, logger *utility.Logger, userAssetRepository database.UserAssetRepository) (*big.Float, error) {
	floatDenomAsset := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: assetSymbol}, &floatDenomAsset); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to denomination of float asset", err)
	}
	sum, err := userAssetRepository.SumAmountField(&model.UserAsset{DenominationID: floatDenomAsset.ID})
	if err != nil {
		return nil, err
	}
	denominationDecimal := float64(floatDecimal)
	scaledTotalSum := big.NewFloat(float64(sum) * math.Pow(10, denominationDecimal))
	return scaledTotalSum, nil
}

func getFloatParamFor(assetSymbol, network string, repository database.BaseRepository, logger *utility.Logger) (model.FloatManagerParam, error) {
	//Get float manager params
	floatManagerParam := model.FloatManagerParam{AssetSymbol: assetSymbol, Network : network}
	if err := repository.GetByFieldName(floatManagerParam, &floatManagerParam); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get float manager params", err)
		return model.FloatManagerParam{}, err
	}
	return floatManagerParam, nil
}

func GetFloatAccounts(repository database.BaseRepository, logger *utility.Logger) ([]model.HotWalletAsset, error) {
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

func getDepositsSumForAssetFromDate(repository database.BaseRepository, assetSymbol, network string,  logger *utility.Logger, hotWallet model.HotWalletAsset) (*big.Float, error) {
	deposits := []model.Transaction{}
	if err := repository.FetchByFieldNameFromDate(model.Transaction{
		TransactionTag: "DEPOSIT",
		AssetSymbol:    assetSymbol,
		Network: network,
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
	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	recipientNetworkAsset, err := services.GetNetworkByAssetAndNetwork(&userAssetRepository, network, assetSymbol)
	if err != nil {
		logger.Error(fmt.Sprintf("error with getting network asset for deposit txn with assetSymbol : %s and network : %s : %s", assetSymbol, network, err))
		return nil, err
	}
	for _, deposit := range deposits {
		//convert to native units
		balance, _ := strconv.ParseFloat(deposit.Value, 64)
		denominationDecimal := float64(recipientNetworkAsset.NativeDecimals)
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

func getWithdrawalsSumForAssetFromDate(repository database.BaseRepository, assetSymbol, network string, logger *utility.Logger, hotWallet model.HotWalletAsset) (*big.Float, error) {
	withdrawals := []model.Transaction{}
	if err := repository.FetchByFieldNameFromDate(model.Transaction{
		TransactionTag: "WITHDRAW",
		AssetSymbol:    assetSymbol,
		Network: network,
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
	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	withdrawalNetworkAsset, err := services.GetNetworkByAssetAndNetwork(&userAssetRepository, network, assetSymbol)
	if err != nil {
		logger.Error(fmt.Sprintf("error with getting network asset for deposit txn with assetSymbol : %s and network : %s : %s", assetSymbol, network, err))
		return nil, err
	}
	for _, withdrawal := range withdrawals {
		//convert to native units
		balance, _ := strconv.ParseFloat(withdrawal.Value, 64)
		denominationDecimal := float64(withdrawalNetworkAsset.NativeDecimals)
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

func sendSingleTransactionToChain(cache *utility.MemoryCache, repository database.BaseRepository, amount *big.Int, depositAccount dto.DepositAddressResponse, logger *utility.Logger, config Config.Data, floatAccount model.HotWalletAsset, serviceErr dto.ServicesRequestErr) error {

	sendSingleTransactionRequest := dto.SendSingleTransactionRequest{
		FromAddress: floatAccount.Address,
		ToAddress:   depositAccount.Address,
		Memo:        depositAccount.Tag,
		Amount:      amount,
		AssetSymbol: floatAccount.AssetSymbol,
		IsSweep:     false,
		ProcessType: utility.FLOATPROCESS,
		Reference:   uuid.NewV1().String(),
	}
	sendSingleTransactionResponse := dto.SendTransactionResponse{}
	if err := services.SendSingleTransaction(cache, logger, config, sendSingleTransactionRequest, &sendSingleTransactionResponse, &serviceErr); err != nil {
		logger.Error("Error response from float manager : %+v. While signing transaction to debit float for %+v", err, floatAccount.AssetSymbol)
		return err
	}

	return nil
}

func ExecuteFloatManagerCronJob(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, userAssetRepository database.UserAssetRepository) {
	c := cron.New()
	c.AddFunc(config.FloatCronInterval, func() { ManageFloat(cache, logger, config, repository, userAssetRepository) })
	c.Start()
}

func GetMaxUserBalanceFor(repository database.UserAssetRepository, assetType string) (*big.Float, error) {

	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: assetType}, &denomination); err != nil {
		return big.NewFloat(0), err
	}

	maxUserBalance, err := repository.GetMaxUserBalance(denomination.ID)
	if err != nil {
		return big.NewFloat(0), err
	}
	return big.NewFloat(maxUserBalance), nil
}

func GetMinFloatBalance(floatManagerParams model.FloatManagerParam, logger *utility.Logger, totalUserBalance, maxUserBalance *big.Float) *big.Float {

	minPercentageOfMaxUserBalance := big.NewFloat(floatManagerParams.MinPercentMaxUserBalance)
	logger.Info("minimum percentage of maximum user balance used is %+v", minPercentageOfMaxUserBalance)
	minPercentageOfTotalUserBalance := big.NewFloat(floatManagerParams.MinPercentTotalUserBalance)
	logger.Info("average percentage of total users balance used is %+v", minPercentageOfTotalUserBalance)

	minPercentageValueOfMaxUserBalance := new(big.Float)
	minPercentageValueOfTotalUserBalance := new(big.Float)

	minPercentageValueOfMaxUserBalance.Mul(minPercentageOfMaxUserBalance, maxUserBalance)
	logger.Info("minimum percentage value of maximum users balance is %+v", minPercentageValueOfMaxUserBalance)
	minPercentageValueOfTotalUserBalance.Mul(minPercentageOfTotalUserBalance, totalUserBalance)
	logger.Info("avearage percentage value of total users balance is %+v", minPercentageValueOfTotalUserBalance)

	minimumFloatBalance := utility.MaxFloat(minPercentageValueOfTotalUserBalance, minPercentageValueOfMaxUserBalance)
	return minimumFloatBalance
}

func GetMaxFloatBalance(floatManagerParams model.FloatManagerParam, logger *utility.Logger, totalUserBalance, maxUserBalance *big.Float) *big.Float {

	averagePercentageOfTotalUserBalance := big.NewFloat(floatManagerParams.AveragePercentTotalUserBalance)
	logger.Info("minimum percentage value of total user balance used is %+v", averagePercentageOfTotalUserBalance)
	maxPercentageOfTotalUserBalance := big.NewFloat(floatManagerParams.MaxPercentTotalUserBalance)
	logger.Info("maximum percentage of total users balance used is %+v", maxPercentageOfTotalUserBalance)
	maxPercentageOfMaxUserBalance := big.NewFloat(floatManagerParams.MaxPercentMaxUserBalance)
	logger.Info("maximum percentage of maximum users balance used is %+v", maxPercentageOfMaxUserBalance)

	maxPercentageValueOfMaxUserBalance := new(big.Float)
	averagePercentageValueOfTotalUserBalance := new(big.Float)
	maxPercentageValueOfTotalUserBalance := new(big.Float)

	averagePercentageValueOfTotalUserBalance.Mul(averagePercentageOfTotalUserBalance, totalUserBalance)
	logger.Info("minimum percentage value of total users balance is %+v", averagePercentageValueOfTotalUserBalance)
	maxPercentageValueOfMaxUserBalance.Mul(maxPercentageOfMaxUserBalance, maxUserBalance)
	logger.Info("maximum percentage value of maximum users balance is %+v", maxPercentageValueOfMaxUserBalance)

	maxPercentageValueOfTotalUserBalance.Mul(maxPercentageOfTotalUserBalance, totalUserBalance)
	logger.Info("maximum percentage value of total users balance is %+v", maxPercentageValueOfTotalUserBalance)
	A := averagePercentageValueOfTotalUserBalance.Add(averagePercentageValueOfTotalUserBalance, maxPercentageValueOfMaxUserBalance)
	C := utility.MinFloat(A, totalUserBalance)

	maximumFloatBalance := utility.MaxFloat(maxPercentageValueOfTotalUserBalance, C)
	return maximumFloatBalance
}

func IsSentColdWalletMail(repository database.BaseRepository, deficit *big.Float, assetSymbol, network string) (bool, error) {
	floatManager := []model.FloatManager{}
	if err := repository.FetchByLastRunDate(assetSymbol, network, time.Now().Format("2006-01-02"), &floatManager); err != nil {
		if errorcode.SQL_404 == err.Error() {
			return true, nil
		}
		return false, err
	}
	if len(floatManager) == 0 {
		return false, nil
	}

	deficitValue, _ := deficit.Float64()
	if floatManager[0].Deficit == float64(0) {
		return false, nil
	} else if floatManager[0].Deficit == deficitValue {
		return true, nil
	}

	return false, nil
}
