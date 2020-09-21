package database

import (
	"errors"
	"strconv"
	"strings"
	"wallet-adapter/errorcode"
	"wallet-adapter/model"
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// IUserAssetRepository ...
type IUserAssetRepository interface {
	IRepository
	GetAssetsByID(id, model interface{}) error
	UpdateAssetBalByID(amount, model interface{}) error
	FindOrCreateAssets(checkExistOrUpdate, model interface{}) error
	BulkUpdate(ids interface{}, model interface{}, update interface{}) error
	GetAssetByAddressAndMemo(address, memo, assetSymbol string, model interface{}) error
	GetAssetByAddressAndSymbol(address, assetSymbol string, model interface{}) error
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

func (repo *UserAssetRepository) BulkUpdate(ids interface{}, model interface{}, update interface{}) error {
	if err := repo.DB.Model(model).Where(ids).Updates(update).Error; err != nil {
		repo.Logger.Error("Error with repository BulkUpdate %s", err)
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

// GetMaxUserBalance
func (repo *UserAssetRepository) GetMaxUserBalance(denomination uuid.UUID) (float64, error) {
	maxUserBalance := model.UserAsset{}
	if err := repo.DB.Raw("select available_balance from user_assets where denomination_id=?  order by available_balance desc limit 0,1;", denomination).Scan(&maxUserBalance).Error; err != nil {
		repo.Logger.Error("Error with repository GetMaxUserBalance %s", err)
		return float64(0), utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}
	availableBalance, _ := strconv.ParseFloat(maxUserBalance.AvailableBalance, 64)
	return availableBalance, nil
}

// GetAssetByAddressAndSymbol... Get user asset matching the given condition
func (repo *UserAssetRepository) GetAssetByAddressAndSymbol(address, assetSymbol string, model interface{}) error {
	if err := repo.DB.Raw("SELECT denominations.asset_symbol, denominations.decimal, user_assets.* FROM user_assets inner join denominations ON denominations.id = user_assets.denomination_id WHERE denominations.asset_symbol= ? && user_assets.id in (select id from user_assets where id IN (select asset_id from user_addresses where v2_address = ?))", assetSymbol, address).Scan(model).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return utility.AppError{
				ErrType: errorcode.RECORD_NOT_FOUND,
				Err:     err,
			}
		}
		return utility.AppError{
			ErrType: errorcode.SERVER_ERR,
			Err:     err,
		}
	}
	return nil
}

// GetAssetByAddressAndMemo...  Get user asset matching the given condition
func (repo *UserAssetRepository) GetAssetByAddressAndMemo(address, memo, assetSymbol string, model interface{}) error {
	if err := repo.DB.Raw("SELECT denominations.asset_symbol, denominations.decimal, user_assets.* FROM user_assets inner join denominations ON denominations.id = user_assets.denomination_id WHERE denominations.asset_symbol= ? && user_assets.id in (select id from user_assets where id IN (select asset_id from user_addresses where v2_address = ? && memo = ?))", assetSymbol, address, memo).Scan(model).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return utility.AppError{
				ErrType: errorcode.RECORD_NOT_FOUND,
				Err:     err,
			}
		}
		return utility.AppError{
			ErrType: errorcode.SERVER_ERR,
			Err:     err,
		}
	}
	return nil
}

func (repo *UserAssetRepository) Db() *gorm.DB {
	return repo.DB
}
