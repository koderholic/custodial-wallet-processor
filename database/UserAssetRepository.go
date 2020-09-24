package database

import (
	"strconv"
	"wallet-adapter/model"
	"wallet-adapter/utility/logger"

	uuid "github.com/satori/go.uuid"
)

// IUserAssetRepository ...
type IUserAssetRepository interface {
	IRepository
	GetAssetsByID(id, model interface{}) error
	FindOrCreateAssets(checkExistOrUpdate, model interface{}) error
	BulkUpdate(ids interface{}, model interface{}, update interface{}) error
	GetAssetByAddressAndSymbol(address, assetSymbol string, model interface{}) error
	GetAssetByAddressMemoAndSymbol(address, memo, assetSymbol string, model interface{}) error
	SumAmountField(model interface{}) (float64, error)
	GetMaxUserBalance(denomination uuid.UUID) (float64, error)
}

// UserAssetRepository ...
type UserAssetRepository struct {
	BaseRepository
}

// GetAssetsByID ...
func (repo *UserAssetRepository) GetAssetsByID(id, model interface{}) error {
	if err := repo.DB.Select("denominations.asset_symbol, denominations.decimal,denominations.coin_type, denominations.requires_memo, user_assets.*").Joins("inner join denominations ON denominations.id = user_assets.denomination_id").Where(id).Find(model).Error; err != nil {
		logger.Error("Error with repository GetAssetsByID %s", err)
		return repoError(err)
	}
	return nil
}

func (repo *UserAssetRepository) BulkUpdate(ids interface{}, model interface{}, update interface{}) error {
	if err := repo.DB.Model(model).Where(ids).Updates(update).Error; err != nil {
		logger.Error("Error with repository BulkUpdate %s", err)
		return repoError(err)
	}

	return nil
}

// GetAssetsByID ...
func (repo *UserAssetRepository) SumAmountField(model interface{}) (float64, error) {
	type NResult struct {
		N float64
	}
	var n NResult
	repo.DB.Table("user_assets").Select("sum(available_balance) as n").Where(model).Scan(&n)
	return n.N, nil
}

// FindOrCreate ...
func (repo *UserAssetRepository) FindOrCreateAssets(checkExistOrUpdate interface{}, model interface{}) error {
	if err := repo.DB.Select("denominations.asset_symbol, denominations.decimal,user_assets.*").Joins("inner join denominations ON denominations.id = user_assets.denomination_id").Where(checkExistOrUpdate).Find(model).Error; err != nil {
		if err.Error() == "record not found" {
			if err := repo.DB.Create(model).Error; err != nil {
				logger.Error("Error with repository Create : %s", err)
				return repoError(err)
			}
			if err := repo.DB.Select("denominations.asset_symbol, denominations.decimal,user_assets.*").Joins("inner join denominations ON denominations.id = user_assets.denomination_id").Where(checkExistOrUpdate).Find(model).Error; err != nil {
				logger.Error("Error with repository GetAssetsByID %s", err)
				return repoError(err)
			}
			return nil
		}
		logger.Error("Error with repository GetAssetsByID %s", err)
		return repoError(err)
	}

	return nil
}

// GetMaxUserBalance
func (repo *UserAssetRepository) GetMaxUserBalance(denomination uuid.UUID) (float64, error) {
	maxUserBalance := model.UserAsset{}
	if err := repo.DB.Raw("select available_balance from user_assets where denomination_id=?  order by available_balance desc limit 0,1;", denomination).Scan(&maxUserBalance).Error; err != nil {
		logger.Error("Error with repository GetMaxUserBalance %s", err)
		return float64(0), repoError(err)
	}
	availableBalance, _ := strconv.ParseFloat(maxUserBalance.AvailableBalance, 64)
	return availableBalance, nil
}

// GetAssetByAddressAndSymbol... Get user asset matching the given condition
func (repo *UserAssetRepository) GetAssetByAddressAndSymbol(address, assetSymbol string, model interface{}) error {
	if err := repo.DB.Select("denominations.asset_symbol, denominations.decimal, user_addresses.address, user_assets.*").
		Joins("inner join denominations ON denominations.id = user_assets.denomination_id").
		Joins("inner join user_addresses ON user_addresses.asset_id = user_assets.id").
		Where("address = ? && asset_symbol = ?", address, assetSymbol).
		First(model).Error; err != nil {
		return repoError(err)
	}
	return nil
}

// GetAssetByAddressMemoAndSymbol...  Get user asset matching the given condition
func (repo *UserAssetRepository) GetAssetByAddressMemoAndSymbol(address, memo, assetSymbol string, model interface{}) error {
	if err := repo.DB.Select("denominations.asset_symbol, denominations.decimal, user_addresses.v2_address, user_addresses.memo, user_assets.*").
		Joins("inner join denominations ON denominations.id = user_assets.denomination_id").
		Joins("inner join user_addresses ON user_addresses.asset_id = user_assets.id").
		Where("v2_address = ? & asset_symbol = ? & memo = ?", address, assetSymbol, memo).
		First(model).Error; err != nil {
		return repoError(err)
	}
	return nil
}
