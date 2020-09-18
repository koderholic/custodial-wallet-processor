package database

import (
	"net/http"
	"time"
	"wallet-adapter/errorcode"
	"wallet-adapter/utility"
	"wallet-adapter/utility/logger"

	uuid "github.com/satori/go.uuid"
)

// IRepository ... Interface definition for IRepository
type IRepository interface {
	Get(id interface{}, model interface{}) error
	GetByFieldName(field interface{}, model interface{}) error
	FetchByFieldName(field interface{}, model interface{}) error
	Fetch(model interface{}) error
	Create(model interface{}) error
	Update(id interface{}, model interface{}) error
	FindOrCreate(checkExistOrUpdate interface{}, model interface{}) error
	UpdateOrCreate(checkExistOrUpdate interface{}, model interface{}, update interface{}) error
	FetchTransactionsWhereIn(values []string, model interface{}) error
	FetchBatchesWithStatus(statuses []string, batches interface{}) error
	FetchByLastRunDate(assettype, lastRund string, model interface{}) error
}

// BaseRepository ... Model definition for database base repository
type BaseRepository struct {
	Database
}

func (repo *BaseRepository) FetchBatchesWithStatus(statuses []string, batches interface{}) error {
	if err := repo.DB.Where("status IN (?)", statuses).Find(batches).Error; err != nil {
		logger.Error("Error with repository FetchBatchesWithStatus %s", err)
		return repoError(err)
	}
	return nil
}

func (repo *BaseRepository) FetchTransactionsWhereIn(values []string, model interface{}) error {
	if err := repo.DB.Where("transaction_id IN (?)", values).Find(model).Error; err != nil {
		logger.Error("Error with repository FetchWhereIn %s", err)
		return repoError(err)
	}

	return nil
}

// Get ... Retrieves a specified record from the database for a given id
func (repo *BaseRepository) Get(id interface{}, model interface{}) error {
	if err := repo.DB.First(model, id).Error; err != nil {
		logger.Error("Error with repository Get : %+v", err)
		return repoError(err)
	}
	return nil
}

// GetByFieldName ... Retrieves a record for the specified model from the database for a given field name
func (repo *BaseRepository) GetByFieldName(field interface{}, model interface{}) error {
	if err := repo.DB.Where(field).First(model).Error; err != nil {
		logger.Error("Error with repository GetByFieldName : %+v", err)
		return repoError(err)
	}
	return nil
}

// FetchByFieldName ... Retrieves all records for the specified model from the database for a given field name
func (repo *BaseRepository) FetchByFieldName(field interface{}, model interface{}) error {
	if err := repo.DB.Where(field).Find(model).Error; err != nil {
		logger.Error("Error with repository FetchByFieldName : %s", err)
		return repoError(err)
	}
	return nil
}

func (repo *BaseRepository) FetchSweepCandidates(model interface{}) error {
	sweepCandidatesQuery := "SELECT * FROM transactions where (transaction_tag='DEPOSIT' and transaction_status='COMPLETED' and swept_status=0) ORDER BY asset_symbol, created_at"
	if err := repo.DB.Raw(sweepCandidatesQuery).Scan(model).Error; err != nil {
		logger.Error("Error when fetching sweep status : %s", err)
		return repoError(err)
	}
	return nil
}

// FetchByFieldName ... Retrieves all records for the specified model from the database for a given field name from a specified date
func (repo *BaseRepository) FetchByFieldNameFromDate(field interface{}, model interface{}, date *time.Time) error {
	if date == nil {
		allTransactionsMatching := repo.DB.Where(field)
		if err := allTransactionsMatching.Where("created_at < CURRENT_TIMESTAMP").Find(model).Order("created_at", true).Error; err != nil {
			logger.Error("Error with repository FetchByFieldName : %s", err)
			return repoError(err)
		}
		return nil
	}
	allTransactionsMatching := repo.DB.Where(field)
	if err := allTransactionsMatching.Where("created_at > ?", *date).Find(model).Error; err != nil {
		logger.Error("Error with repository FetchByFieldName : %s", err)
		return repoError(err)
	}
	return nil
}

// Fetch ... Retrieves all records from the database for a given models
func (repo *BaseRepository) Fetch(model interface{}) error {
	if err := repo.DB.Find(model).Error; err != nil {
		logger.Error("Error with repository Fetch : %s", err)
		return repoError(err)
	}
	return nil
}

// Create ... Create a record on the database for a the given model
func (repo *BaseRepository) Create(model interface{}) error {
	if err := repo.DB.Create(model).Error; err != nil {
		logger.Error("Error with repository Create : %s", err)
		return repoError(err)
	}
	return nil
}

// Update ... Update a specified record from the database for a given id
func (repo *BaseRepository) Update(id, model interface{}) error {

	if err := repo.DB.Model(id).Update(model).Error; err != nil {
		logger.Error("Error with repository Update : %s", err)
		return repoError(err)
	}
	repo.DB.Where(id).First(model)
	return nil
}

// FindOrCreate ...
func (repo *BaseRepository) FindOrCreate(checkExistOrUpdate interface{}, model interface{}) error {
	if err := repo.DB.FirstOrCreate(model, checkExistOrUpdate).Error; err != nil {
		logger.Error("Error with repository FindOrCreateUserAsset : %s", err)
		return repoError(err)
	}
	return nil
}

// UpdateOrCreate ...
func (repo *BaseRepository) UpdateOrCreate(checkExistOrUpdate interface{}, model interface{}, update interface{}) error {
	if err := repo.DB.Where(checkExistOrUpdate).Assign(update).FirstOrCreate(model).Error; err != nil {
		logger.Error("Error with repository UpdateOrCreate : %s", err)
		return repoError(err)
	}
	return nil
}

func (repo *BaseRepository) BulkUpdateTransactionSweptStatus(idList []uuid.UUID) error {
	if err := repo.DB.Exec("UPDATE transactions SET swept_status=true WHERE id IN (?)", idList).Error; err != nil {
		logger.Error("Error with repository bulk update transaction swept_status : %s", err)
		return repoError(err)
	}
	return nil
}

func (repo *BaseRepository) FetchByLastRunDate(assettype, lastRunDate string, model interface{}) error {
	if err := repo.DB.Raw("SELECT * FROM float_manager_variables WHERE asset_symbol = ? AND last_run_time >= ? ORDER BY last_run_time DESC", assettype, lastRunDate).Scan(model).Error; err != nil {
		logger.Error("Error with repository FetchByLastRunDate : %s", err)
		return repoError(err)
	}
	return nil
}

func repoError(err error) error {
	if err.Error() == errorcode.SQL_404 {
		return utility.AppError{
			ErrType: errorcode.SQL_ERR,
			ErrCode: http.StatusBadRequest,
			Err:     err, //errors.New(strings.Join(strings.Split(err.Error(), " ")[2:], " ")),
		}
	}
	return utility.AppError{
		ErrType: errorcode.SERVER_ERR,
		ErrCode: http.StatusInternalServerError,
		Err:     err,
	}
}
