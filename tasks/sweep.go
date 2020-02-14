package tasks

import (
	"errors"
	"fmt"
	"github.com/robfig/cron/v3"
	uuid "github.com/satori/go.uuid"
	"strconv"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"
)

func SweepTransactions(logger *utility.Logger, config Config.Data, repository database.BaseRepository) {
	logger.Info("Sweep operation begins")
	serviceErr := model.ServicesRequestErr{}
	token, err := acquireLock(logger, config, serviceErr)
	if err != nil {
		logger.Error("Could not acquire lock", err)
		return
	}

	var transactions []dto.Transaction
	if err := repository.FetchByFieldName(&dto.Transaction{TransactionTag: dto.TransactionTag.DEPOSIT,
		SweptStatus: false, TransactionStatus: dto.TransactionStatus.COMPLETED}, &transactions); err != nil {
		logger.Error("Error response from Sweep job : could not fetch sweep candidates %+v", err)
		return
	}
	//group transactions by recipientId
	transactionsPerUserId := make(map[uuid.UUID][]dto.Transaction)
	for _, tx := range transactions {
		transactionsPerUserId[tx.RecipientID] = append(transactionsPerUserId[tx.RecipientID], tx)
	}
	//For each recipient id, group transactions by symbol
	transactionsPerUserIdPerSymbol := make(map[string][]dto.Transaction)
	for userId, userTransactions := range transactionsPerUserId {
		//For each recipient id, group transactions by symbol
		for _, tx := range userTransactions {
			//need recipient Asset to know which crypto symbol this Tx relates to
			recipientAsset := dto.UserAssetBalance{}
			if err := repository.Get(tx.RecipientID, &recipientAsset); err != nil {
				logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
				continue
			}
			//the key has to be unique for each user and each coin symbol
			transactionsPerUserIdPerSymbol[userId.String()+"_"+recipientAsset.Symbol] = append(transactionsPerUserIdPerSymbol[recipientAsset.Symbol], tx)
		}
	}
	for _, userTransactions := range transactionsPerUserIdPerSymbol {
		//Get total sum to be swept for this symbol
		var sum = int64(0)
		var count = 0
		for _, tx := range userTransactions {
			floatValue, err := strconv.ParseFloat(tx.Value, 64)
			if err == nil {
				//Value has no values after dp, as we expect the all crypto values are in their smallest unit
				sum = sum + int64(floatValue)
				count++
			}
		}
		//passing the first tx(just needed for data) here and the sum
		if err := sweepPerUserPerSymbol(logger, config, repository, serviceErr, userTransactions, sum); err != nil {
			continue
		}
	}

	if err := releaseLock(logger, config, token, serviceErr); err != nil {
		logger.Error("Could not release lock", err)
		return
	}
	logger.Info("Sweep operation ends successfully, lock released")
}

func sweepPerUserPerSymbol(logger *utility.Logger, config Config.Data, repository database.BaseRepository, serviceErr model.ServicesRequestErr, userTransactions []dto.Transaction, sum int64) error {
	//need recipient Asset to get recipient address
	recipientAsset := dto.UserAssetBalance{}
	if err := repository.Get(userTransactions[0].RecipientID, &recipientAsset); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
		return err
	}
	//get recipient address
	recipientAddress := dto.UserAddress{}
	if err := repository.Get(userTransactions[0].RecipientID, &recipientAddress); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
		return err
	}
	// Get address onchain balance
	onchainBalanceRequest := model.OnchainBalanceRequest{
		Address:     recipientAddress.Address,
		AssetSymbol: recipientAsset.Symbol,
	}
	onchainBalanceResponse := model.OnchainBalanceResponse{}
	if err := services.GetOnchainBalance(logger, config, onchainBalanceRequest, &onchainBalanceResponse, &serviceErr); err != nil {
		if serviceErr.Code != "" {
			logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
			return err
		}
		return err
	}

	//Get the float address
	var floatAccount dto.HotWalletAsset
	if err := repository.GetByFieldName(&dto.HotWalletAsset{AssetSymbol: recipientAsset.Symbol}, &floatAccount); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
		return err
	}

	switch recipientAsset.Symbol {
	case "ETH":
		if sum < config.EthTreshholdValue {
			return errors.New(fmt.Sprintf("Skipping asset, %s total sum for this asset, for this coin %s is not up to treshhold value %s", recipientAsset.ID, recipientAsset.Symbol, config.EthTreshholdValue))
		}
	default:
		logger.Error("Could not sweep for asset with id %+v : %+v with unknown assetSymbol of type %+v", recipientAsset.ID, recipientAsset.Symbol)
	}
	// Calls key-management to sign transaction
	signTransactionRequest := model.SignTransactionRequest{
		FromAddress: recipientAddress.Address,
		ToAddress:   floatAccount.Address,
		Amount:      sum,
		CoinType:    recipientAsset.Symbol,
	}
	signTransactionResponse := model.SignTransactionResponse{}
	if err := services.SignTransaction(logger, config, signTransactionRequest, &signTransactionResponse, serviceErr); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
		return err
	}

	// Send the signed data to crypto adapter to send to chain
	broadcastToChainRequest := model.BroadcastToChainRequest{
		SignedData:  signTransactionResponse.SignedData,
		AssetSymbol: recipientAsset.Symbol,
	}
	broadcastToChainResponse := model.BroadcastToChainResponse{}

	if err := services.BroadcastToChain(logger, config, broadcastToChainRequest, &broadcastToChainResponse, serviceErr); err == nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
		return err
	}
	//update all tx with new swept status
	for _, tx := range userTransactions {
		tx.SweptStatus = true
		if err := repository.Update(tx.ID, &tx); err != nil {
			logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
			return err
		}
	}
	return nil
}

func acquireLock(logger *utility.Logger, config Config.Data, serviceErr model.ServicesRequestErr) (string, error) {
	// It calls the lock service to obtain a lock for the transaction
	lockerServiceRequest := model.LockerServiceRequest{
		Identifier:   fmt.Sprintf("%s%s", config.LockerPrefix, "sweep"),
		ExpiresAfter: 600000,
	}
	lockerServiceResponse := model.LockerServiceResponse{}
	if err := services.AcquireLock(logger, config, lockerServiceRequest, &lockerServiceResponse, &serviceErr); err != nil {
		if !serviceErr.Success && serviceErr.Message != "" {
			return "", errors.New(serviceErr.Message)
		}
		return "", err
	}
	return lockerServiceResponse.Token, nil
}

func releaseLock(logger *utility.Logger, config Config.Data, lockerServiceToken string, serviceErr model.ServicesRequestErr) error {
	lockReleaseRequest := model.LockReleaseRequest{
		Identifier: fmt.Sprintf("%s%s", config.LockerPrefix, "sweep"),
		Token:      lockerServiceToken,
	}
	lockReleaseResponse := model.ServicesRequestSuccess{}
	if err := services.ReleaseLock(logger, config, lockReleaseRequest, &lockReleaseResponse, &serviceErr); err != nil {
		if serviceErr.Code != "" {
			return errors.New(serviceErr.Message)
		}
		return err
	}
	return nil
}

func ExecuteCronJob(logger *utility.Logger, config Config.Data, repository database.BaseRepository) {
	c := cron.New()
	c.AddFunc(config.SweepCronInterval, func() { SweepTransactions(logger, config, repository) })
	c.Start()
}
