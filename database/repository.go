package database

import (
	"errors"
	"strings"
	"wallet-adapter/utility"
)

// IRepository ... Interface definition for IRepository
type IRepository interface {
	GetCount(model, count interface{}) error
	Get(id interface{}, model interface{}) error
	GetByFieldName(field interface{}, model interface{}) error
	FetchByFieldName(field interface{}, model interface{}) error
	Fetch(model interface{}) error
	Create(model interface{}) error
	Update(id interface{}, model interface{}) error
	Delete(model interface{}) error
	FindOrCreate(checkExistOrUpdate interface{}, model interface{}) error
}

// BaseRepository ... Model definition for database base repository
type BaseRepository struct {
	Database
}

// GetCount ... Get model count
func (repo *BaseRepository) GetCount(model, count interface{}) error {
	if err := repo.DB.Model(model).Count(count).Error; err != nil {
		repo.Logger.Error("Error with repository GetCount %s", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}
	return nil
}

// Get ... Retrieves a specified record from the database for a given id
func (repo *BaseRepository) Get(id interface{}, model interface{}) error {
	if repo.DB.First(model, id).RecordNotFound() {
		repo.Logger.Error("Error with repository Get %s", "Record not found")
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     errors.New("No record found for provided Id"),
		}
	}
	return nil
}

// GetByFieldName ... Retrieves a record for the specified model from the database for a given field name
func (repo *BaseRepository) GetByFieldName(field interface{}, model interface{}) error {
	if err := repo.DB.Where(field).First(model).Error; err != nil {
		repo.Logger.Error("Error with repository GetByFieldName : %+v", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}
	return nil
}

// FetchByFieldName ... Retrieves all records for the specified model from the database for a given field name
func (repo *BaseRepository) FetchByFieldName(field interface{}, model interface{}) error {
	if err := repo.DB.Where(field).Find(model).Error; err != nil {
		repo.Logger.Error("Error with repository FetchByFieldName : %s", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}
	return nil
}

// Fetch ... Retrieves all records from the database for a given models
func (repo *BaseRepository) Fetch(model interface{}) error {
	if err := repo.DB.Find(model).Error; err != nil {
		repo.Logger.Error("Error with repository Fetch : %s", err)
		return utility.AppError{
			ErrType: "SYSTEM_ERR",
			Err:     err,
		}
	}
	return nil
}

// Create ... Create a record on the database for a the given model
func (repo *BaseRepository) Create(model interface{}) error {
	if err := repo.DB.Create(model).Error; err != nil {
		repo.Logger.Error("Error with repository Create : %s", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     errors.New(strings.Join(strings.Split(err.Error(), " ")[2:], " ")),
		}
	}
	return nil
}

// Update ... Update a specified record from the database for a given id
func (repo *BaseRepository) Update(id, model interface{}) error {

	if err := repo.DB.Model(model).Update(model).Error; err != nil {
		repo.Logger.Error("Error with repository Update : %s", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     errors.New(strings.Join(strings.Split(err.Error(), " ")[2:], " ")),
		}
	}
	repo.DB.Where("id = ?", id).First(model)
	return nil
}

// Delete ... Deletes a specified record from the database for a given id
func (repo *BaseRepository) Delete(model interface{}) error {
	if err := repo.DB.Delete(model).Error; err != nil {
		repo.Logger.Error("Error (with repository Delete : %s", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}
	return nil
}

// FindOrCreate ...
func (repo *BaseRepository) FindOrCreate(checkExistOrUpdate interface{}, model interface{}) error {
	if err := repo.DB.FirstOrCreate(model, checkExistOrUpdate).Error; err != nil {
		repo.Logger.Error("Error with repository FindOrCreateUserAsset : %s", err)
		return utility.AppError{
			ErrType: "INPUT_ERR",
			Err:     err,
		}
	}
	return nil
}
