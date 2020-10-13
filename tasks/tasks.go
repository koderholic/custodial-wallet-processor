package tasks

import (
	"math"
	"math/big"
	"sort"
	"strconv"
	"time"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/constants"
	"wallet-adapter/utility/errorcode"
	"wallet-adapter/utility/logger"

	uuid "github.com/satori/go.uuid"
)

func ReleaseLock(repository database.IUserAssetRepository, cache *cache.Memory, config Config.Data, lockerServiceToken string) error {
	LockerService := services.NewLockerService(cache, config, repository)
	if err := LockerService.ReleaseLock("sweep", lockerServiceToken); err != nil {
		return err
	}
	return nil
}

func NotifyColdWalletUsersViaSMS(amount big.Int, assetSymbol string, config Config.Data, cache *cache.Memory, repository database.IUserAddressRepository) {
	denomination := model.Denomination{}
	if err := repository.GetByFieldName(&model.Denomination{AssetSymbol: assetSymbol, IsEnabled: true}, &denomination); err != nil {
		logger.Error("Error response from NotifyColdWalletUsersViaSMS : %+v while trying to denomination of float asset", err)
	}
	decimalBalance := ConvertBigIntToDecimalUnit(amount, denomination)
	//send sms
	LockerService := services.NewLockerService(cache, config, repository)
	_, err := LockerService.AcquireLock(errorcode.INSUFFICIENT_BALANCE_FLOAT_SEND_SMS+constants.SEPERATOR+assetSymbol, constants.ONE_HOUR_MILLISECONDS)
	if err != nil {
		//lock was successfully acquired
		NotificationService := services.NewNotificationService(cache, config, repository)
		NotificationService.BuildAndSendSms(assetSymbol, decimalBalance)
	}
}

func ConvertBigIntToDecimalUnit(amount big.Int, denomination model.Denomination) *big.Float {
	amountInFloat, _ := strconv.ParseFloat(amount.String(), 64)
	amountInBigFloat := big.NewFloat(amountInFloat)
	decimalBalance := amountInBigFloat.Quo(amountInBigFloat, big.NewFloat(math.Pow(10, float64(denomination.Decimal))))
	return decimalBalance
}

func GetFloatParamFor(assetSymbol string, repository database.IRepository) (model.FloatManagerParam, error) {
	//Get float manager params
	floatManagerParam := model.FloatManagerParam{AssetSymbol: assetSymbol}
	if err := repository.GetByFieldName(floatManagerParam, &floatManagerParam); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get float manager params", err)
		return model.FloatManagerParam{}, err
	}
	return floatManagerParam, nil
}

func GetDepositsSumForAssetFromDate(repository database.IRepository, config Config.Data, assetSymbol string, hotWallet model.HotWalletAsset) (*big.Float, error) {
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
		GetRecipientAsset(repository, config, deposit.RecipientID, &recipientAsset)
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

func GetRecipientAsset(repository database.IRepository, config Config.Data, assetId uuid.UUID, recipientAsset *model.UserAsset) {
	userAssetRepository := database.UserAssetRepository{BaseRepository: database.BaseRepository{Database: database.Database{Config: config, DB: repository.Db()}}}
	if err := userAssetRepository.GetAssetsByID(&model.UserAsset{BaseModel: model.BaseModel{ID: assetId}}, &recipientAsset); err != nil {
		logger.Error("Error response from Float Manager job : %+v while checking for asset with id %+v", err, recipientAsset.ID)
		return
	}
}

//total liability at any given time
func GetTotalUserBalance(repository database.IRepository, assetSymbol string, userAssetRepository database.IUserAssetRepository) (*big.Float, error) {
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

func GetWithdrawalsSumForAssetFromDate(repository database.IRepository, config Config.Data, assetSymbol string, hotWallet model.HotWalletAsset) (*big.Float, error) {
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
		GetRecipientAsset(repository, config, withdrawal.InitiatorID, &recipientAsset)
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
