package services

import (
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/utility/logger"

	"wallet-adapter/dto"
	"wallet-adapter/utility"

	uuid "github.com/satori/go.uuid"
)

//BatchService object
type BatchService struct {
	Cache  *utility.MemoryCache
	Config Config.Data
	Error  *dto.ExternalServicesRequestErr
}

func NewBatchService(cache *utility.MemoryCache, config Config.Data) *BatchService {
	baseService := BatchService{
		Cache:  cache,
		Config: config,
	}
	return &baseService
}

func (service BatchService) GetWaitingBTCBatchId(repository database.IBatchRepository, assetSymbol string) (uuid.UUID, error) {

	var currentBatch model.BatchRequest
	if err := repository.GetByFieldName(&model.BatchRequest{Status: model.BatchStatus.WAIT_MODE, AssetSymbol: assetSymbol}, &currentBatch); err != nil {
		if err.Error() != errorcode.SQL_404 {
			logger.Error("Error response from batch service : ", err)
			return uuid.UUID{}, err
		}
		// Create new batch entry
		currentBatch.AssetSymbol = assetSymbol
		if err := repository.Create(&currentBatch); err != nil {
			logger.Error("Error response from batch service : ", err)
			return uuid.UUID{}, err
		}
	}

	return currentBatch.ID, nil
}

func (service BatchService) GetAllActiveBatches(repository database.IBatchRepository) ([]model.BatchRequest, error) {

	var activeBatches []model.BatchRequest
	if err := repository.FetchBatchesWithStatus([]string{model.BatchStatus.WAIT_MODE, model.BatchStatus.RETRY_MODE, model.BatchStatus.START_MODE}, &activeBatches); err != nil {
		return []model.BatchRequest{}, err
	}
	return activeBatches, nil
}

func (service BatchService) CheckBatchExistAndReturn(repository database.IBatchRepository, batchId uuid.UUID) (bool, model.BatchRequest, error) {
	batchDetails := model.BatchRequest{}
	if batchId == uuid.Nil {
		return false, batchDetails, nil
	}
	if err := repository.GetByFieldName(&model.BatchRequest{BaseModel: model.BaseModel{ID: batchId}}, &batchDetails); err != nil {
		logger.Error("Error getting batch details : %+v for batch with id %+v", err)
		if err.Error() != errorcode.SQL_404 {
			return false, model.BatchRequest{}, err
		}
		return false, model.BatchRequest{}, nil
	}
	return true, batchDetails, nil
}
