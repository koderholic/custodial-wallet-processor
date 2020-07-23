package test

import (
	"math/big"
	"time"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/tasks"
	"wallet-adapter/utility"

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
	services.GetOnchainBalance(cache, s.Logger, s.Config, onchainBalanceRequest, &floatOnChainBalanceResponse, serviceErr)
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

	sum := tasks.CalculateSumOfBtcBatch(addressTransactions)

	if sum < 0.2 {
		s.T().Errorf("Expected sum returned to be greater than  %s, got %f\n", "0.2", sum)
	}

}
