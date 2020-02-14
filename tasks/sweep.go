package tasks

import (
	"errors"
	"fmt"
	"github.com/jasonlvhit/gocron"
	"strconv"
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/dto"
	"wallet-adapter/model"
	"wallet-adapter/services"
	"wallet-adapter/utility"
)

func SweepTransactions(logger *utility.Logger, config Config.Data, repository database.BaseRepository) {
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

	for i, tx := range transactions {
		fmt.Println(i, tx.ID)

		chainTransaction := dto.ChainTransaction{}
		if err := repository.Get(tx.OnChainTxId, &chainTransaction); err != nil {
			logger.Error("Error response from Sweep job : %+v at tx id %+v", err, tx.ID)
			continue
		}

		recipientAsset := dto.UserAssetBalance{}
		if err := repository.Get(tx.RecipientID, &recipientAsset); err != nil {
			logger.Error("Error response from Sweep job : %+v at tx id %+v", err, tx.ID)
			continue
		}
		recipientAddress := dto.UserAddress{}
		if err := repository.Get(tx.RecipientID, &recipientAddress); err != nil {
			logger.Error("Error response from Sweep job : %+v at tx id %+v", err, tx.ID)
			continue
		}
		// Get address onchain balance
		onchainBalanceRequest := model.OnchainBalanceRequest{
			Address:     recipientAddress.Address,
			AssetSymbol: recipientAsset.Symbol,
		}
		onchainBalanceResponse := model.OnchainBalanceResponse{}
		if err := services.GetOnchainBalance(logger, config, onchainBalanceRequest, &onchainBalanceResponse, &serviceErr); err != nil {
			if serviceErr.Code != "" {
				logger.Error("Error response from Sweep job : %+v at tx id %+v", err, tx.ID)
				continue
			}
			continue
		}

		recipientBalance, err := strconv.ParseInt(onchainBalanceResponse.Balance, 10, 64)
		if err != nil {
			logger.Error("Error response from Sweep job : %+v at tx id %+v", err, tx.ID)
			continue
		}
		var floatAccount dto.HotWalletAsset
		// The routine fetches the float account info from the db
		if err := repository.GetByFieldName(&dto.HotWalletAsset{AssetSymbol: recipientAsset.Symbol}, &floatAccount); err != nil {
			logger.Error("Error response from Sweep job : %+v at tx id %+v", err, tx.ID)
			continue
		}
		//signTx
		switch recipientAsset.Symbol {
		case "ETH":
			if recipientBalance > config.EthTreshholdValue {
				// Calls key-management to sign transaction
				signTransactionRequest := model.SignTransactionRequest{
					FromAddress: recipientAddress.Address,
					ToAddress:   floatAccount.Address,
					Amount:      recipientBalance,
					CoinType:    recipientAsset.Symbol,
				}
				signTransactionResponse := model.SignTransactionResponse{}
				if err := services.SignTransaction(logger, config, signTransactionRequest, &signTransactionResponse, serviceErr); err != nil {
					logger.Error("Error response from Sweep job : %+v at tx id %+v", err, tx.ID)
					continue
				}

				// Send the signed data to crypto adapter to send to chain
				broadcastToChainRequest := model.BroadcastToChainRequest{
					SignedData:  signTransactionResponse.SignedData,
					AssetSymbol: recipientAsset.Symbol,
				}
				broadcastToChainResponse := model.BroadcastToChainResponse{}

				if err := services.BroadcastToChain(logger, config, broadcastToChainRequest, &broadcastToChainResponse, serviceErr); err == nil {
					logger.Error("Error response from Sweep job : %+v at tx id %+v", err, tx.ID)
					continue
				}
				tx.SweptStatus = true
				if err := repository.Update(tx.ID, &tx); err != nil {
					logger.Error("Error response from Sweep job : %+v at tx id %+v", err, tx.ID)
					continue
				}
			}

		default:
			logger.Error("Could not sweep tx id : %+v with unknown assetSymbol of type %+v", tx.ID, recipientAsset.Symbol)
		}

	}
	if err := releaseLock(logger, config, token, serviceErr); err != nil {
		logger.Error("Could not release lock", err)
		return
	}
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

func ExecuteCronJob(logger *utility.Logger, config Config.Data, userAssetRepository database.BaseRepository) {
	s := gocron.NewScheduler()
	s.Every(10).Minutes().From(gocron.NextTick()).DoSafely(SweepTransactions, logger, config, userAssetRepository)
	<-s.Start()
}
