package services

import (
	Config "wallet-adapter/config"
	"wallet-adapter/utility"
	"wallet-adapter/database"
	"wallet-adapter/model"
	uuid "github.com/satori/go.uuid"
)

func GetWaitingBTCBatchId(repository database.IUserAssetRepository, logger *utility.Logger) (uuid.UUID, error)  {
	
	var currentBatch model.BatchRequest
	if err := repository.GetByFieldName(&model.BatchRequest{Status: model.BatchStatus.WAIT_MODE}, &currentBatch); err != nil {
		if err.Error() != utility.SQL_404 {
			logger.Error("Error response from batch service : ", err)
			return uuid.UUID{}, err
		}
		// Create new batch entry
		currentBatch.AssetSymbol = utility.BTC
		if err := repository.Create(&currentBatch); err != nil {
			logger.Error("Error response from batch service : ", err)
			return uuid.UUID{}, err
		}
	}
	return currentBatch.ID, nil
}

func GetAllActiveBatches(repository database.IUserAssetRepository, logger *utility.Logger, config Config.Data) ([]model.BatchRequest, error)  {
	
	var activeBatches []model.BatchRequest
	if err := repository.FetchActiveBatches([]string{model.BatchStatus.WAIT_MODE, model.BatchStatus.RETRY_MODE}, &activeBatches); err != nil {
		return []model.BatchRequest{}, err
	}

	return activeBatches, nil
}

func CheckBatchExistAndReturn(repository database.IUserAssetRepository, logger *utility.Logger, batchId uuid.UUID ) (bool, model.BatchRequest, error)  {
	batchDetails := model.BatchRequest{}
	if err := repository.GetByFieldName(&model.BatchRequest{BaseModel :model.BaseModel{ID : batchId}}, &batchDetails); err != nil {
		logger.Error("Error getting batch details : %+v for batch with id %+v", err)
		if err.Error() != utility.SQL_404 {
			return false, model.BatchRequest{}, nil
		}
		return false, model.BatchRequest{}, err
	}
	return true, batchDetails, nil
}
