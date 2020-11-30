package test

import (
	"fmt"
	"math/big"
	"time"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/tasks"
	"wallet-adapter/utility"

	"github.com/magiconair/properties/assert"
	uuid "github.com/satori/go.uuid"
)

func (s *Suite) TestSweep() {
	purgeInterval := s.Config.PurgeCacheInterval * time.Second
	cacheDuration := s.Config.ExpireCacheDuration * time.Second
	authCache := utility.InitializeCache(cacheDuration, purgeInterval)
	baseRepository := database.BaseRepository{Database: s.Database}
	tasks.SweepTransactions(authCache, s.Logger, s.Config, baseRepository)

}

func (s *Suite) TestGetSweepAddressAndMemo() {

	purgeInterval := s.Config.PurgeCacheInterval * time.Second
	cacheDuration := s.Config.ExpireCacheDuration * time.Second
	cache := utility.InitializeCache(cacheDuration, purgeInterval)
	baseRepository := database.BaseRepository{Database: s.Database}
	userAssetRepository := database.UserAssetRepository{BaseRepository: baseRepository}

	floatAccount := model.HotWalletAsset{
		BaseModel: model.BaseModel{
			ID: uuid.FromStringOrNil("1ea282ca-8a08-4343-b1c4-372176809b13"),
		},
		Address:     "bnb1x2kvd50cmggdmuqlqgznksyeskquym2zcmvlhg",
		AssetSymbol: "BNB",
		IsDisabled:  false,
	}

	// Get float chain balance
	prec := uint(64)
	serviceErr := dto.ServicesRequestErr{}
	onchainBalanceRequest := dto.OnchainBalanceRequest{
		AssetSymbol: floatAccount.AssetSymbol,
		Address:     floatAccount.Address,
	}
	floatOnChainBalanceResponse := dto.OnchainBalanceResponse{}
	_ = services.GetOnchainBalance(cache, s.Logger, s.Config, onchainBalanceRequest, &floatOnChainBalanceResponse, serviceErr)
	floatOnChainBalance, _ := new(big.Float).SetPrec(prec).SetString(floatOnChainBalanceResponse.Balance)

	// Get total users balance
	totalUserBalance, err := tasks.GetTotalUserBalance(baseRepository, floatAccount.AssetSymbol, s.Logger, userAssetRepository)
	if err != nil {
		s.T().Errorf("Expected GetTotalUserBalance to not error, got %s\n", err)
	}

	valueOfMinimumFloatPercent := new(big.Float)
	valueOfMinimumFloatPercent.Mul(big.NewFloat(0.01), totalUserBalance)

	toAddress, _, _ := tasks.GetSweepAddressAndMemo(cache, s.Logger, s.Config, baseRepository, floatAccount)

	if floatOnChainBalance.Cmp(valueOfMinimumFloatPercent) > 0 {
		if toAddress == "bnb1x2kvd50cmggdmuqlqgznksyeskquym2zcmvlhg" {
			s.T().Errorf("Expected toAddress returned to not be empty and to not equal %s, got %s\n", "bnb1x2kvd50cmggdmuqlqgznksyeskquym2zcmvlhg", toAddress)
		}
	}

}

func (s *Suite) TestCalculateSumOfBtcBatch() {
	addressTransactions := []model.Transaction{}
	transation1 := model.Transaction{
		Value: "0.12390554019510966",
	}
	transation2 := model.Transaction{
		Value: "0.112390554019510966",
	}

	addressTransactions = append(addressTransactions, transation1)
	addressTransactions = append(addressTransactions, transation2)

	sum := tasks.CalculateSumOfBatch(addressTransactions)

	if sum < 0.2 {
		s.T().Errorf("Expected sum returned to be greater than  %s, got %f\n", "0.2", sum)
	}

}

func (s *Suite) TestRemoveBTCTx() {
	addressTransactions := []model.Transaction{}
	btcTransactions := []model.Transaction{}
	transation1 := model.Transaction{
		BaseModel: model.BaseModel{ID: uuid.NewV1()},
		Value:     "0.12390554019510966",
	}
	transation2 := model.Transaction{
		BaseModel: model.BaseModel{ID: uuid.NewV1()},
		Value:     "0.112390554019510966",
	}

	addressTransactions = append(addressTransactions, transation1)
	addressTransactions = append(addressTransactions, transation2)
	btcTransactions = append(btcTransactions, transation1)

	result := tasks.RemoveBatchTransactions(addressTransactions, btcTransactions)

	if len(result) != 1 {
		fmt.Println(result[0].ID)
		s.T().Errorf("Didnt succesfully get difference in lists ")
	}
}

func (s *Suite) TestUniqueAddress() {
	var addresses []string
	address1 := "bc1q94tsgpe25dtwuu7w0k7de4m62mdzjesle4zjex"
	address2 := "bc1qcg8gqlq84veds0gxe20masexr22f2atjn6g6yj"
	address3 := "bc1qfaawd7h0axqjuhj8e5wta8jgyqxt96rrwfp6qt"

	addresses = append(addresses, address1)
	addresses = append(addresses, address2)
	addresses = append(addresses, address3)
	addresses = append(addresses, address1)
	fmt.Println("length before unique operation is ", len(addresses))

	addresses = tasks.ToUniqueAddresses(addresses)

	if len(addresses) != 3 {
		s.T().Errorf("Didnt succesfully get unique addresses ")
	}
}

func (s *Suite) TestGetFloatDeficit() {
	depositSum := big.NewFloat(5000)
	withdrawalSum := big.NewFloat(3000)
	onchainBalance := big.NewFloat(500)
	minimumFloat := big.NewFloat(1000)
	maximumFloat := big.NewFloat(3000)

	result := tasks.GetFloatDeficit(depositSum, withdrawalSum, minimumFloat, maximumFloat, onchainBalance, s.Logger)
	deficit, _ := result.Float64()

	assert.Equal(s.T(), float64(500), deficit, "Incorrect deficit amount returned")
}

func (s *Suite) TestGeTFloatPercent() {
	floatDeficit := big.NewFloat(500)
	sweepSum := big.NewFloat(5000)

	sweepPercent := tasks.GeTFloatPercent(floatDeficit, sweepSum)

	assert.Equal(s.T(), int64(10), sweepPercent.Int64(), "Incorrect sweep percent for float returned")
}

func (s *Suite) TestGetFloatBalanceRange() {
	floatParam := model.FloatManagerParam{
		MinPercentTotalUserBalance: float64(0.01),
		MaxPercentTotalUserBalance: float64(0.1),
	}
	totalUserBalance := big.NewFloat(5000)

	min, max := tasks.GetFloatBalanceRange(floatParam, totalUserBalance, s.Logger)
	mimBalance, _ := min.Float64()
	maxBalance, _ := max.Float64()

	assert.Equal(s.T(), float64(50), mimBalance, "Incorrect minimum balance returned")
	assert.Equal(s.T(), float64(500), maxBalance, "Incorrect maximum balance returned")
}

func (s *Suite) TestGetSweepPercentages() {
	totalUsersBalance := big.NewFloat(5000)
	onchainBalance := big.NewFloat(500)
	minimumFloat := big.NewFloat(1000)
	floatDeficit := big.NewFloat(500)
	sweepFund := big.NewFloat(500)

	floatParam := model.FloatManagerParam{
		MinPercentTotalUserBalance: float64(0.2),
		MaxPercentTotalUserBalance: float64(0.3),
	}

	floatPercent, brokeragePercent := tasks.GetSweepPercentages(onchainBalance, minimumFloat, floatDeficit, sweepFund, totalUsersBalance, floatParam, s.Logger)
	totalPercent := floatPercent + brokeragePercent

	assert.Equal(s.T(), totalPercent, int64(100), "Sweep percentages do not sum up to 100")
}

func (s *Suite) TestGetSweepPercentageValues() {
	totalUsersBalance := big.NewFloat(80000)
	onchainBalance := big.NewFloat(10000)
	minimumFloat := big.NewFloat(12000)
	floatDeficit := big.NewFloat(2000)
	sweepFund := big.NewFloat(10000)

	floatParam := model.FloatManagerParam{
		MinPercentTotalUserBalance: float64(1.5),
		MaxPercentTotalUserBalance: float64(0.3),
	}

	floatPercent, brokeragePercent := tasks.GetSweepPercentages(onchainBalance, minimumFloat, floatDeficit, sweepFund, totalUsersBalance, floatParam, s.Logger)

	assert.Equal(s.T(), floatPercent, int64(20), "float percent is invalid")
	assert.Equal(s.T(), brokeragePercent, int64(80), "brokerage percent is invalid")
}

func (s *Suite) TestSumSweepTx() {
	transactions := []model.Transaction{}
	transaction1 := model.Transaction{
		Value: "0.2",
	}
	transaction2 := model.Transaction{
		Value: "0.3",
	}
	transactions = append(transactions, transaction1)
	transactions = append(transactions, transaction2)

	sum := tasks.CalculateSum(transactions)

	assert.Equal(s.T(), sum, float64(0.5), "Sum should be equal to 0.5")
}

func (s *Suite) TestCheckSweepMinimum() {
	denomination := model.Denomination{
		AssetSymbol:      "ETH",
		MinimumSweepable: 0.9,
	}
	sum := float64(0.5)
	isAmountSufficient, _ := tasks.CheckSweepMinimum(denomination, s.Config, sum, s.Logger)

	assert.Equal(s.T(), isAmountSufficient, false, "Sum should not be sufficient")
}

func (s *Suite) TestCheckTrxSweepLimitDoesNotExceed() {
	userAddress := model.UserAddress{
		SweepCount: 10,
		NextSweepTime: time.Now(),
	}
	repository := database.BaseRepository{database.Database{Logger: s.Logger, DB: s.DB, Config: s.Config}}
	isExceededSweepLimit := tasks.CheckTrxSweepLimit(userAddress, s.Logger, "TRX", repository)

	assert.Equal(s.T(), isExceededSweepLimit, false, "Sweep limit should not be exceeded")
}

func (s *Suite) TestCheckTrxSweepLimitExceeds() {
	userAddress := model.UserAddress{
		SweepCount: 25,
		NextSweepTime: time.Now(),
	}
	repository := database.BaseRepository{database.Database{Logger: s.Logger, DB: s.DB, Config: s.Config}}
	isExceededSweepLimit := tasks.CheckTrxSweepLimit(userAddress, s.Logger, "TRX", repository)

	assert.Equal(s.T(), isExceededSweepLimit, true, "Sweep limit should be exceeded")
}

func (s *Suite) TestCheckTrxSweepLimitDoesNotExceedAfterReset() {
	userAddress := model.UserAddress{
		SweepCount: 0,
		NextSweepTime: time.Now(),
	}
	repository := database.BaseRepository{database.Database{Logger: s.Logger, DB: s.DB, Config: s.Config}}
	isExceededSweepLimit := tasks.CheckTrxSweepLimit(userAddress, s.Logger, "TRX", repository)

	assert.Equal(s.T(), isExceededSweepLimit, false, "Sweep limit should not be exceeded")
}

func (s *Suite) TestCheckTrxSweepLimitDoesExceedAfterReset() {
	userAddress := model.UserAddress{
		SweepCount: 0,
		NextSweepTime: time.Now().AddDate(0,0,1),
	}
	repository := database.BaseRepository{database.Database{Logger: s.Logger, DB: s.DB, Config: s.Config}}
	isExceededSweepLimit := tasks.CheckTrxSweepLimit(userAddress, s.Logger, "TRX", repository)

	assert.Equal(s.T(), isExceededSweepLimit, true, "Sweep limit should be exceeded")
}


func (s *Suite) TestResetTRXSweepCount() {
	userAddress := model.UserAddress{
		SweepCount: 20,
		NextSweepTime: time.Now(),
	}
	repository := database.BaseRepository{database.Database{Logger: s.Logger, DB: s.DB, Config: s.Config}}
	_ = tasks.ResetTRXSweepCount(repository, &userAddress)

	assert.Equal(s.T(), userAddress.SweepCount, 0, "Sweep count should reset to zero")
}