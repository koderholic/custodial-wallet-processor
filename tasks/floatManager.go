package tasks

import (
	"github.com/robfig/cron/v3"
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
		response := model.OnchainBalanceResponse{}
		services.GetOnchainBalance(cache, logger, config, request, &response, serviceErr)

		//TODO Orchestrate binance broker and cold wallet

	}

	if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
		logger.Error("Could not release lock", err)
		return
	}
	logger.Info("Float manager process ends successfully, lock released")
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

func getDepositsForAsset(repository database.BaseRepository, assetSymbol string, logger *utility.Logger) ([]dto.Transaction, error) {
	deposits := []dto.Transaction{}
	if err := repository.FetchByFieldName(dto.Transaction{
		TransactionTag: "DEPOSIT",
		AssetSymbol:    assetSymbol,
	}, deposits); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get deposits", err)
		return nil, err
	}
	return deposits, nil
}

func getWithdrawalsForAsset(repository database.BaseRepository, assetSymbol string, logger *utility.Logger) ([]dto.Transaction, error) {
	deposits := []dto.Transaction{}
	if err := repository.FetchByFieldName(dto.Transaction{
		TransactionTag: "WITHDRAW",
		AssetSymbol:    assetSymbol,
	}, deposits); err != nil {
		logger.Error("Error response from Float manager : %+v while trying to get withdrawals", err)
		return nil, err
	}
	return deposits, nil
}

func ExecuteFloatManagerCronJob(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository) {
	c := cron.New()
	c.AddFunc(config.SweepCronInterval, func() { manageFloat(cache, logger, config, repository) })
	c.Start()
}
