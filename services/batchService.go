package services

import (
	"wallet-adapter/database"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"

	uuid "github.com/satori/go.uuid"
)

type BatchService struct {
	BaseService
}

func (service BatchService) GetWaitingBatchId(repository database.IBatchRepository, assetSymbol, network string) (uuid.UUID, error) {

	var currentBatch model.BatchRequest
	if err := repository.GetByFieldName(&model.BatchRequest{Status: model.BatchStatus.WAIT_MODE, AssetSymbol: assetSymbol, Network: network}, &currentBatch); err != nil {
		if err.Error() != errorcode.SQL_404 {
			service.Logger.Error("Error response from batch service : ", err)
			return uuid.UUID{}, err
		}
		// Create new batch entry
		currentBatch.AssetSymbol = assetSymbol
		currentBatch.Network = network
		if err := repository.Create(&currentBatch); err != nil {
			service.Logger.Error("Error response from batch service : ", err)
			return uuid.UUID{}, err
		}
	}

	return currentBatch.ID, nil
}

func (service BatchService) GetAllActiveBatches(repository database.IBatchRepository, assetSymbol string) ([]model.BatchRequest, error) {

	var activeBatches []model.BatchRequest
	if err := repository.FetchBatchesByStatusAndSymbol([]string{model.BatchStatus.WAIT_MODE, model.BatchStatus.RETRY_MODE, model.BatchStatus.START_MODE}, assetSymbol, &activeBatches); err != nil {
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
		service.Logger.Error("Error getting batch details : %+v for batch with id %+v", err)
		if err.Error() != errorcode.SQL_404 {
			return false, model.BatchRequest{}, err
		}
		return false, model.BatchRequest{}, nil
	}
	return true, batchDetails, nil
}
