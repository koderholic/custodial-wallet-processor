package services

import (
	Config "wallet-adapter/config"
	"wallet-adapter/utility"
	"wallet-adapter/database"
	"wallet-adapter/model"
	uuid "github.com/satori/go.uuid"
	"time"
)

func GetActiveBTCBatchId(repository database.IUserAssetRepository, logger *utility.Logger) (uuid.UUID, error)  {
	
	var activeBatch model.BatchRequest
	if err := repository.GetByFieldName(&model.BatchRequest{Status: model.BatchStatus.WAIT_MODE}, &activeBatch); err != nil {
		if err.Error() != utility.SQL_404 {
			logger.Error("Error response from batch service : ", err)
			return uuid.UUID{}, err
		}
		// Create new batch entry
		activeBatch.AssetSymbol = utility.BTC
		if err := repository.Create(&activeBatch); err != nil {
			logger.Error("Error response from batch service : ", err)
			return uuid.UUID{}, err
		}
	}
	return activeBatch.ID, nil
}

func CheckActiveBatchExistAndReturn(repository database.IUserAssetRepository, logger *utility.Logger, config Config.Data) (bool, uuid.UUID, error)  {
	
	var activeBatch model.BatchRequest
	if err := repository.GetByFieldName(&model.BatchRequest{Status: model.BatchStatus.WAIT_MODE}, &activeBatch); err != nil {
		if err.Error() == utility.SQL_404 {
			logger.Error("Error response from batch service : ", err)
			return false, uuid.UUID{}, nil
		}
		return false, uuid.UUID{}, err
	}

	// Check batch duration
	timeElapsed := time.Since(activeBatch.CreatedAt) 
	timeElapsedMinutes := timeElapsed.Minutes()

	if timeElapsedMinutes < float64(config.BTCBatchInterval) {
		return false, uuid.UUID{}, nil 
	}

	return true, activeBatch.ID, nil
}
