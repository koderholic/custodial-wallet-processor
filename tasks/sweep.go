package tasks

import (
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

	var transactions []dto.Transaction
	if err := repository.FetchByFieldName(&dto.Transaction{TransactionTag: dto.TransactionTag.DEPOSIT,
		SweptStatus: false, TransactionStatus: dto.TransactionStatus.COMPLETED}, &transactions); err != nil {
		logger.Error("Error response from Sweep job : %+v", err)
		return
	}

	for i, tx := range transactions {
		fmt.Println(i, tx.ID)

		chainTransaction := dto.ChainTransaction{}
		if err := repository.Get(tx.OnChainTxId, &chainTransaction); err != nil {
			logger.Error("Error response from Sweep job : %+v", err)
			return
		}

		recipientAsset := dto.UserAssetBalance{}
		if err := repository.Get(tx.RecipientID, &recipientAsset); err != nil {
			logger.Error("Error response from Sweep job : %+v", err)
			return
		}
		recipientAddress := dto.UserAddress{}
		if err := repository.Get(tx.RecipientID, &recipientAddress); err != nil {
			logger.Error("Error response from Sweep job : %+v", err)
			return
		}
		// Get address onchain balance
		onchainBalanceRequest := model.OnchainBalanceRequest{
			Address:     recipientAddress.Address,
			AssetSymbol: recipientAsset.Symbol,
		}
		onchainBalanceResponse := model.OnchainBalanceResponse{}
		if err := services.GetOnchainBalance(logger, config, onchainBalanceRequest, &onchainBalanceResponse, &serviceErr); err != nil {
			if serviceErr.Code != "" {
				logger.Error("Error response from Sweep job : %+v", err)
				return
			}
			return
		}

		recipientBalance, err := strconv.ParseInt(onchainBalanceResponse.Balance, 10, 64)
		if err != nil {
			logger.Error("Error response from Sweep job : %+v", err)
			return
		}
		var floatAccount dto.HotWalletAsset
		// The routine fetches the float account info from the db
		if err := repository.GetByFieldName(&dto.HotWalletAsset{AssetSymbol: recipientAsset.Symbol}, &floatAccount); err != nil {
			logger.Error("Error response from Sweep job : %+v", err)
			return
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
					logger.Error("Error response from Sweep job : %+v", err)
					return
				}

				// Send the signed data to crypto adapter to send to chain
				broadcastToChainRequest := model.BroadcastToChainRequest{
					SignedData:  signTransactionResponse.SignedData,
					AssetSymbol: recipientAsset.Symbol,
				}
				broadcastToChainResponse := model.BroadcastToChainResponse{}

				if err := services.BroadcastToChain(logger, config, broadcastToChainRequest, &broadcastToChainResponse, serviceErr); err == nil {
					logger.Error("Error response from Sweep job : %+v", err)
					return
				}
				tx.SweptStatus = true
				if err := repository.Update(tx.ID, &tx); err != nil {
					logger.Error("Error response from Sweep job : %+v", err)
					return
				}
			}

		default:
			logger.Error("Could not sweep tx id : %+v with unknown assetSymbol of type %+v", tx.ID, recipientAsset.Symbol)
		}

	}
}

func ExecuteCronJob(logger *utility.Logger, config Config.Data, userAssetRepository database.BaseRepository) {
	s := gocron.NewScheduler()
	s.Every(10).Minutes().From(gocron.NextTick()).DoSafely(SweepTransactions, logger, config, userAssetRepository)
	<-s.Start()
}
