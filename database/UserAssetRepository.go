package database

import (
	"errors"
	"strings"
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"
)

// IUserAssetRepository ...
type IUserAssetRepository interface {
	IRepository
	GetAssetsByID(id, model interface{}) error
	UpdateAssetBalByID(amount, model interface{}) error
	FindOrCreateAssets(checkExistOrUpdate, model interface{}) error
	BulkUpdate(ids interface{}, model interface{}, uodate interface{}) error
	FetchActiveBatches(statuses interface{}, batches interface{}) error
	Db() *gorm.DB
}

// UserAssetRepository ...
type UserAssetRepository struct {
	BaseRepository
}

// GetAssetsByID ...
func (repo *UserAssetRepository) GetAssetsByID(id, model interface{}) error {
	if err := repo.DB.Select("denominations.asset_symbol, denominations.decimal,denominations.coin_type, denominations.requires_memo, user_assets.*").Joins("inner join denominations ON denominations.id = user_assets.denomination_id").Where(id).Find(model).Error; err != nil {
		repo.Logger.Error("Error with repository GetAssetsByID %s", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}
	return nil
}

func (repo *UserAssetRepository) BulkUpdate(ids interface{}, model interface{}, uodate interface{}) error {
	if err := repo.DB.Model(model).Where(ids).Updates(uodate).Error; err != nil {
		repo.Logger.Error("Error with repository BulkUpdate %s", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}

	return nil
}


func (repo *UserAssetRepository) FetchActiveBatches(statuses interface{}, batches interface{}) error {
	if err :=  repo.DB.Where("status IN (?)", statuses).Find(&batches).Error; err != nil {
		repo.Logger.Error("Error with repository FetchActiveBatches %s", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}

	return nil
}

// GetAssetsByID ...
func (repo *UserAssetRepository) SumAmountField(model interface{}) (float64, error) {
	//var sum float64
	//Note i am summing here using sql here so addition is in crypto decimal units which is what its saved in.
	// This is fine for float management but dont use this method for transactional stuff. Floating point addition
	// is a problem. rater convert to native units and then sum. :)

	type NResult struct {
		N float64 //or int ,or some else
	}

	var n NResult
	repo.DB.Table("user_assets").Select("sum(available_balance) as n").Where(model).Scan(&n)
	return n.N, nil

	/*if err := repo.DB.Table("user_assets").Select("sum(available_balance)").Row().Scan(&sum); err != nil {
		repo.Logger.Error("Error with repository GetAssetsByID %s", err)
		return 0, utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}
	return sum, nil*/
}

// UpdateAssetByID ...
func (repo *UserAssetRepository) UpdateAssetBalByID(amount, model interface{}) error {
	if err := repo.DB.Model(&model).Update("available_balance", gorm.Expr("available_balance - ?", amount)).Error; err != nil {
		repo.Logger.Error("Error with repository GetAssetsByID %s", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}

	return nil
}

// FindOrCreate ...
func (repo *UserAssetRepository) FindOrCreateAssets(checkExistOrUpdate interface{}, model interface{}) error {
	if err := repo.DB.Select("denominations.asset_symbol, denominations.decimal,user_assets.*").Joins("inner join denominations ON denominations.id = user_assets.denomination_id").Where(checkExistOrUpdate).Find(model).Error; err != nil {
		if err.Error() == "record not found" {
			if err := repo.DB.Create(model).Error; err != nil {
				repo.Logger.Error("Error with repository Create : %s", err)
				return utility.AppError{
					ErrType: "INPUT_ERR",
					Err:     errors.New(strings.Join(strings.Split(err.Error(), " ")[2:], " ")),
				}
			}
			if err := repo.DB.Select("denominations.asset_symbol, denominations.decimal,user_assets.*").Joins("inner join denominations ON denominations.id = user_assets.denomination_id").Where(checkExistOrUpdate).Find(model).Error; err != nil {
				repo.Logger.Error("Error with repository GetAssetsByID %s", err)
				return utility.AppError{
					ErrType: "INPUT_ERR",
					Err:     err,
				}
			}
			return nil
		}
		repo.Logger.Error("Error with repository GetAssetsByID %s", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}

	return nil
}

func (repo *UserAssetRepository) Db() *gorm.DB {
	return repo.DB
}
