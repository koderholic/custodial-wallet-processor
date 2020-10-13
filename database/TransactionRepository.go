package database

import (
	"github.com/jinzhu/gorm"
)

// ITransactionRepository ...
type ITransactionRepository interface {
	IUserAssetRepository
}

// TransactionRepository ...
type TransactionRepository struct {
	UserAssetRepository
}

func (repo *TransactionRepository) Db() *gorm.DB {
	return repo.DB
}
