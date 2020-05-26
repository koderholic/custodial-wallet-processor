package database

import (
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"
)

// IBatchRepository ...
type IBatchRepository interface {
	IRepository
	BulkUpdate(ids interface{}, model interface{}, uodate interface{}) error
	FetchActiveBatches(statuses []string, batches interface{}) error
	Db() *gorm.DB
}

// BatchRepository ...
type BatchRepository struct {
	BaseRepository
}

func (repo *BatchRepository) FetchActiveBatches(statuses []string, batches interface{}) error {
	if err := repo.DB.Where("status IN (?)", statuses).Find(batches).Error; err != nil {
		repo.Logger.Error("Error with repository FetchActiveBatches %s", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}

	return nil
}

func (repo *BatchRepository) BulkUpdate(ids interface{}, model interface{}, uodate interface{}) error {
	if err := repo.DB.Model(model).Where(ids).Updates(uodate).Error; err != nil {
		repo.Logger.Error("Error with repository BulkUpdate %s", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}

	return nil
}

func (repo *BatchRepository) Db() *gorm.DB {
	return repo.DB
}
