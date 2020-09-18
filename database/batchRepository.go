package database

import (
	"wallet-adapter/utility/logger"

	"github.com/jinzhu/gorm"
)

// IBatchRepository ...
type IBatchRepository interface {
	IRepository
	BulkUpdate(ids interface{}, model interface{}, uodate interface{}) error
	Db() *gorm.DB
}

// BatchRepository ...
type BatchRepository struct {
	BaseRepository
}

func (repo *BatchRepository) BulkUpdate(ids interface{}, model interface{}, uodate interface{}) error {
	if err := repo.DB.Model(model).Where(ids).Updates(uodate).Error; err != nil {
		logger.Error("Error with repository BulkUpdate %s", err)
		return repoError(err)
	}

	return nil
}

func (repo *BatchRepository) Db() *gorm.DB {
	return repo.DB
}
