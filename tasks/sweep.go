package tasks

import (
	"errors"
	"fmt"
	"github.com/robfig/cron/v3"
	uuid "github.com/satori/go.uuid"
	"math"
	"strconv"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"
)

func SweepTransactions(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository) {
	logger.Info("Sweep operation begins")
	serviceErr := model.ServicesRequestErr{}
	token, err := acquireLock("sweep", cache, logger, config, serviceErr)
	if err != nil {
		logger.Error("Could not acquire lock", err)
		return
	}

	var transactions []dto.Transaction
	if err := repository.FetchSweepCandidates(&transactions); err != nil {
		logger.Error("Error response from Sweep job : could not fetch sweep candidates %+v", err)
		if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
			logger.Error("Could not release lock", err)
			return
		}
		return
	}
	//group transactions by recipientId
	transactionsPerAssetId := make(map[uuid.UUID][]dto.Transaction)
	for _, tx := range transactions {
		transactionsPerAssetId[tx.RecipientID] = append(transactionsPerAssetId[tx.RecipientID], tx)
	}

	var btcAssets []string
	var btcAssetTransactionsToSweep []dto.Transaction
	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	for assetId, assetTransactions := range transactionsPerAssetId {
		//Filter BTC assets, save in a seperate list for batch processing and skip individual processing
		//need recipient Asset to check assetSymbol
		recipientAsset := dto.UserAsset{}
		//all the tx in assetTransactions have the same recipientId so just pass the 0th position
		if err := userAssetRepository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: assetId}}, &recipientAsset); err != nil {
			logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
			if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
				logger.Error("Could not release lock", err)
				return
			}
			return
		}
		if recipientAsset.AssetSymbol == "BTC" {
			//get recipient address
			recipientAddress := dto.UserAddress{}
			if err := repository.Get(dto.UserAddress{AssetID: assetId}, &recipientAddress); err != nil {
				logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
				if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
					logger.Error("Could not release lock", err)
					return
				}
				return
			}
			btcAssets = append(btcAssets, recipientAddress.Address)
			btcAssetTransactionsToSweep = append(btcAssetTransactionsToSweep, assetTransactions...)
			//skip futher processing for this asset, will be included a part of batch btc processing
			continue
		}
		//Get total sum to be swept for this assetId
		var sum = int64(0)
		var count = 0
		for _, tx := range assetTransactions {
			//convert to native units
			balance, _ := strconv.ParseFloat(tx.Value, 64)
			denominationDecimal := float64(recipientAsset.Decimal)
			scaledBalance := int64(balance * math.Pow(10, denominationDecimal))
			sum = sum + scaledBalance
			count++
		}
		if err := sweepPerAssetId(cache, logger, config, repository, serviceErr, assetTransactions, sum); err != nil {
			continue
		}
	}
	//batch process btc
	if len(btcAssets) > 0 {
		if err := sweepBatchTx(cache, logger, config, repository, serviceErr, btcAssets, btcAssetTransactionsToSweep); err != nil {
			logger.Error("Error response from Sweep job : %+v while sweeping batch transactions for BTC", err)
			if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
				logger.Error("Could not release lock", err)
				return
			}
			return
		}
	}
	if err := releaseLock(cache, logger, config, token, serviceErr); err != nil {
		logger.Error("Could not release lock", err)
		return
	}
	logger.Info("Sweep operation ends successfully, lock released")
}

func sweepBatchTx(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, serviceErr model.ServicesRequestErr, btcAssets []string, btcAssetTransactionsToSweep []dto.Transaction) error {
	// Calls key-management to batch sign transaction
	recipientData := []model.BatchRecipients{}
	//get float
	floatAccount, err := getFloatDetails(repository, "BTC", logger)
	if err != nil {
		return err
	}
	floatRecipient := model.BatchRecipients{
		Address: floatAccount.Address,
		Value:   0,
	}
	recipientData = append(recipientData, floatRecipient)
	signTransactionRequest := model.BatchBTCRequest{
		AssetSymbol:   "BTC",
		ChangeAddress: floatAccount.Address,
		IsSweep:       true,
		Origins:       btcAssets,
		Recipients:    recipientData,
	}
	signTransactionResponse := model.SignTransactionResponse{}
	if err := services.SignBatchBTCTransaction(nil, cache, logger, config, signTransactionRequest, &signTransactionResponse, serviceErr); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping batch transactions for BTC", err)
		return err
	}
	if err := broadcastSweepTx(signTransactionResponse, config, "BTC", cache, logger, serviceErr, btcAssetTransactionsToSweep, repository); err != nil {
		return err
	}
	if err := updateSweptStatus(btcAssetTransactionsToSweep, repository, logger); err != nil {
		return err
	}
	return nil

}

func sweepPerAssetId(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository, serviceErr model.ServicesRequestErr, assetTransactions []dto.Transaction, sum int64) error {
	//need recipient Asset to get recipient address
	recipientAsset := dto.UserAsset{}
	//all the tx in assetTransactions have the same recipientId so just pass the 0th position
	userAssetRepository := database.UserAssetRepository{BaseRepository: repository}
	if err := userAssetRepository.GetAssetsByID(&dto.UserAsset{BaseDTO: dto.BaseDTO{ID: assetTransactions[0].RecipientID}}, &recipientAsset); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
		return err
	}
	//get recipient address
	recipientAddress := dto.UserAddress{}
	if err := repository.Get(dto.UserAddress{AssetID: assetTransactions[0].RecipientID}, &recipientAddress); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
		return err
	}
	floatAccount, err := getFloatDetails(repository, recipientAsset.AssetSymbol, logger)
	if err != nil {
		return err
	}
	denomination := dto.Denomination{}
	if err := repository.GetByFieldName(&dto.Denomination{AssetSymbol: floatAccount.AssetSymbol, IsEnabled: true}, &denomination); err != nil {
		logger.Error("Error response from sweep process : %+v while trying to denomination of float asset", err)
		return err
	}
	//Do this only for BEp-2 tokens and not for BNB itself
	if denomination.CoinType == utility.BNBTOKENSLIP && denomination.AssetSymbol != "BNB" {
		//Check that fee is below X% of the total value.
		err = feeThresholdCheck(denomination.SweepFee, sum, config, logger, recipientAsset)
		if err != nil {
			return err
		}
		//send sweep fee to main address
		err, done := fundSweepFee(floatAccount, denomination, recipientAddress, cache, logger, config, serviceErr, recipientAsset, assetTransactions, repository)
		if done {
			return err
		}
	}

	// Calls key-management to sign transaction
	signTransactionRequest := model.SignTransactionRequest{
		FromAddress: recipientAddress.Address,
		ToAddress:   floatAccount.Address,
		Amount:      0,
		AssetSymbol: recipientAsset.AssetSymbol,
		IsSweep:     true,
	}
	signTransactionResponse := model.SignTransactionResponse{}
	if err := services.SignTransaction(cache, logger, config, signTransactionRequest, &signTransactionResponse, serviceErr); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
		return err
	}
	//Check that fee is below X% of the total value.
	if err := feeThresholdCheck(signTransactionResponse.Fee, sum, config, logger, recipientAsset); err != nil {
		return err
	}
	if err := broadcastSweepTx(signTransactionResponse, config, recipientAsset.AssetSymbol, cache, logger, serviceErr, assetTransactions, repository); err != nil {
		return err
	}
	if err := updateSweptStatus(assetTransactions, repository, logger); err != nil {
		return err
	}
	return nil
}

func feeThresholdCheck(fee int64, sum int64, config Config.Data, logger *utility.Logger, recipientAsset dto.UserAsset) error {
	if (((fee) / sum) * 100) > config.SweepFeePercentageThreshold {
		logger.Error("Skipping asset, %+v ratio of fee to sum for this asset with asset symbol %+v is greater than the sweepFeePercentageThreshold, would be too expensive to sweep %+v", recipientAsset.ID, recipientAsset.AssetSymbol, config.SweepFeePercentageThreshold)
		return errors.New(fmt.Sprintf("Skipping asset, %s ratio of fee to sum for this asset with asset symbol %s is greater than the sweepFeePercentageThreshold, would be too expensive to sweep %s", recipientAsset.ID, recipientAsset.AssetSymbol, config.SweepFeePercentageThreshold))
	}
	return nil
}

func fundSweepFee(floatAccount dto.HotWalletAsset, denomination dto.Denomination, recipientAddress dto.UserAddress, cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, serviceErr model.ServicesRequestErr, recipientAsset dto.UserAsset, assetTransactions []dto.Transaction, repository database.BaseRepository) (error, bool) {

	request := model.OnchainBalanceRequest{
		AssetSymbol: denomination.MainCoinAssetSymbol,
		Address:     recipientAddress.Address,
	}
	mainCoinOnChainBalanceResponse := model.OnchainBalanceResponse{}
	services.GetOnchainBalance(cache, logger, config, request, &mainCoinOnChainBalanceResponse, serviceErr)
	mainCoinOnChainBalance, _ := strconv.ParseUint(mainCoinOnChainBalanceResponse.Balance, 10, 64)
	//check if onchain balance in main coin asset is less than floatAccount.SweepFee
	if int64(mainCoinOnChainBalance) < denomination.SweepFee {
		// Calls key-management to sign sweep fee transaction
		signTransactionRequest := model.SignTransactionRequest{
			FromAddress: floatAccount.Address,
			ToAddress:   recipientAddress.Address,
			Amount:      denomination.SweepFee,
			AssetSymbol: denomination.MainCoinAssetSymbol,
			//this currently only supports coins that supports Memo, ETH will not be ignored
			Memo: utility.SWEEPMEMOBNB,
		}
		signTransactionResponse := model.SignTransactionResponse{}
		if err := services.SignTransaction(cache, logger, config, signTransactionRequest, &signTransactionResponse, serviceErr); err != nil {
			logger.Error("Error response from Sweep job : %+v while sweeping for asset with id %+v", err, recipientAsset.ID)
			return err, true
		}
		if err := broadcastSweepTx(signTransactionResponse, config, recipientAsset.AssetSymbol, cache, logger, serviceErr, assetTransactions, repository); err != nil {
			return err, true
		}
		//return immediately after broadcasting sweep fee, this allows for confirmation, next time sweep runs,
		// int64(mainCoinOnChainBalance) will be > floatAccount.SweepFee, and so this if block will be skipped
		//i.e sweep fee will not be resent to user address
		return nil, true
	}
	// else? i.e. if mainCoinOnChainBalance > floatAccount.SweepFee, then we want to proceed like the general case for non token sweep
	return nil, false
}

func getFloatDetails(repository database.BaseRepository, symbol string, logger *utility.Logger) (dto.HotWalletAsset, error) {
	//Get the float address
	var floatAccount dto.HotWalletAsset
	if err := repository.Get(&dto.HotWalletAsset{AssetSymbol: symbol}, &floatAccount); err != nil {
		logger.Error("Error response from Sweep job : %+v while sweeping for asset with id and trying to get float detials", err)
		return dto.HotWalletAsset{}, err
	}
	return floatAccount, nil
}

func broadcastSweepTx(signTransactionResponse model.SignTransactionResponse, config Config.Data, symbol string, cache *utility.MemoryCache, logger *utility.Logger, serviceErr model.ServicesRequestErr, assetTransactions []dto.Transaction, repository database.BaseRepository) error {
	// Send the signed data to crypto adapter to send to chain
	broadcastToChainRequest := model.BroadcastToChainRequest{
		SignedData:  signTransactionResponse.SignedData,
		AssetSymbol: symbol,
		ProcessType: utility.SWEEPPROCESS,
	}
	broadcastToChainResponse := model.BroadcastToChainResponse{}
	if err := services.BroadcastToChain(cache, logger, config, broadcastToChainRequest, &broadcastToChainResponse, serviceErr); err != nil {
		logger.Error("Error response from Sweep job : %+v while broadcasting to chain", err)
		return err
	}
	return nil
}

func updateSweptStatus(assetTransactions []dto.Transaction, repository database.BaseRepository, logger *utility.Logger) error {
	//update all assetTransactions with new swept status
	var assetIdList []uuid.UUID
	for _, tx := range assetTransactions {
		assetIdList = append(assetIdList, tx.ID)
	}
	if err := repository.BulkUpdateTransactionSweptStatus(assetIdList); err != nil {
		logger.Error("Error response from Sweep job : %+v while broadcasting to chain", err)
		return err
	}
	return nil
}

func acquireLock(identifier string, cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, serviceErr model.ServicesRequestErr) (string, error) {
	// It calls the lock service to obtain a lock for the transaction
	lockerServiceRequest := model.LockerServiceRequest{
		Identifier:   fmt.Sprintf("%s%s", config.LockerPrefix, identifier),
		ExpiresAfter: 600000,
	}
	lockerServiceResponse := model.LockerServiceResponse{}
	if err := services.AcquireLock(cache, logger, config, lockerServiceRequest, &lockerServiceResponse, &serviceErr); err != nil {
		if !serviceErr.Success && serviceErr.Message != "" {
			return "", errors.New(serviceErr.Message)
		}
		return "", err
	}
	return lockerServiceResponse.Token, nil
}

func releaseLock(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, lockerServiceToken string, serviceErr model.ServicesRequestErr) error {
	lockReleaseRequest := model.LockReleaseRequest{
		Identifier: fmt.Sprintf("%s%s", config.LockerPrefix, "sweep"),
		Token:      lockerServiceToken,
	}
	lockReleaseResponse := model.ServicesRequestSuccess{}
	if err := services.ReleaseLock(cache, logger, config, lockReleaseRequest, &lockReleaseResponse, &serviceErr); err != nil {
		if serviceErr.Code != "" {
			return errors.New(serviceErr.Message)
		}
		return err
	}
	return nil
}

func ExecuteSweepCronJob(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data, repository database.BaseRepository) {
	c := cron.New()
	c.AddFunc(config.SweepCronInterval, func() { SweepTransactions(cache, logger, config, repository) })
	c.Start()
}
