package services

import (
	Config "wallet-adapter/config"
	"wallet-adapter/database"
	"wallet-adapter/model"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/cache"
	"wallet-adapter/utility/errorcode"
	"wallet-adapter/utility/logger"

	"wallet-adapter/dto"

	uuid "github.com/satori/go.uuid"
)

//BatchService object
type BatchService struct {
	Cache      *cache.Memory
	Config     Config.Data
	Error      *dto.ExternalServicesRequestErr
	Repository database.IBatchRepository
}

func NewBatchService(cache *cache.Memory, config Config.Data, repository database.IBatchRepository) *BatchService {
	baseService := BatchService{
		Cache:      cache,
		Config:     config,
		Repository: repository,
	}
	return &baseService
}

func (service BatchService) GetWaitingBatchId(assetSymbol string) (uuid.UUID, error) {

func (service BatchService) GetWaitingBatchId(assetSymbol string) (uuid.UUID, error) {
	repository := service.Repository.(database.IBatchRepository)
	var currentBatch model.BatchRequest
	if err := service.Repository.GetByFieldName(&model.BatchRequest{Status: model.BatchStatus.WAIT_MODE, AssetSymbol: assetSymbol}, &currentBatch); err != nil {

		appErr := err.(appError.Err)
		if appErr.ErrType != errorcode.RECORD_NOT_FOUND {
			logger.Error("GetWaitingBTCBatchId Logs : Error fetching batch in WAIT_MODE for %s  > %s ", assetSymbol, err)
			return uuid.UUID{}, err
		}
		// Create new batch entry
		currentBatch.AssetSymbol = assetSymbol
		if err := service.Repository.Create(&currentBatch); err != nil {
			logger.Error("GetWaitingBTCBatchId Logs : Error creating new batch in WAIT_MODE for %s > %s", assetSymbol, err)
			return uuid.UUID{}, err
		}
	}

	return currentBatch.ID, nil
}

func (service BatchService) GetAllActiveBatches() ([]model.BatchRequest, error) {

	var activeBatches []model.BatchRequest
	if err := service.Repository.FetchBatchesWithStatus([]string{model.BatchStatus.WAIT_MODE, model.BatchStatus.RETRY_MODE, model.BatchStatus.START_MODE}, &activeBatches); err != nil {

		logger.Error("GetAllActiveBatches Logs : Error fetching all active batches > %s", err)
		return []model.BatchRequest{}, err
	}
	return activeBatches, nil
}

func (service BatchService) CheckBatchExistAndReturn(batchId uuid.UUID) (bool, model.BatchRequest, error) {

	batchDetails := model.BatchRequest{}
	if batchId == uuid.Nil {
		return false, batchDetails, nil
	}

	if err := service.Repository.GetByFieldName(&model.BatchRequest{BaseModel: model.BaseModel{ID: batchId}}, &batchDetails); err != nil {
		logger.Error("GetByFieldName Logs : Error fetching batch details for batchId %v > %s", batchId, err)
		if err.Error() != errorcode.SQL_404 {
			return false, model.BatchRequest{}, err
		}
		return false, model.BatchRequest{}, nil
	}
	return true, batchDetails, nil
}
