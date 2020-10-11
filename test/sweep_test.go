package test

import (
	"fmt"
	"math/big"
	"time"
	"wallet-adapter/database"
	"wallet-adapter/model"
	tasks "wallet-adapter/tasks/sweep"
	"wallet-adapter/utility/cache"

	"github.com/magiconair/properties/assert"
	uuid "github.com/satori/go.uuid"
)

func (s *Suite) TestSweep() {
	purgeInterval := s.Config.PurgeCacheInterval * time.Second
	cacheDuration := s.Config.ExpireCacheDuration * time.Second
	authCache := cache.Initialize(cacheDuration, purgeInterval)
	baseRepository := database.BaseRepository{Database: s.Database}
	tasks.SweepTransactions(authCache, s.Config, &baseRepository)

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

	result := tasks.GetFloatDeficit(depositSum, withdrawalSum, minimumFloat, maximumFloat, onchainBalance)
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

	min, max := tasks.GetFloatBalanceRange(floatParam, totalUserBalance)
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

	floatPercent, brokeragePercent := tasks.GetSweepPercentages(onchainBalance, minimumFloat, floatDeficit, sweepFund, totalUsersBalance, floatParam)
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

	floatPercent, brokeragePercent := tasks.GetSweepPercentages(onchainBalance, minimumFloat, floatDeficit, sweepFund, totalUsersBalance, floatParam)

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
	isAmountSufficient, _ := tasks.CheckSweepMinimum(denomination, s.Config, sum)

	assert.Equal(s.T(), isAmountSufficient, false, "Sum should not be sufficient")
}
