package test

import (
	"time"
	Config "wallet-adapter/config"
	"wallet-adapter/dto"
	"wallet-adapter/utility"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
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
}

// BaseRepository ... Model definition for database base repository
type BaseRepository struct {
	Logger *utility.Logger
	Config Config.Data
	DB     *gorm.DB
}

// GetCount ... Get model count
func (repo *BaseRepository) GetCount(model, count interface{}) error {
	switch model.(type) {
	case dto.Asset:
		recordID := uuid.NewV4()
		dbMockedData := dto.Asset{}
		dbMockedData.ID = recordID
		dbMockedData.CreatedAt = time.Now()
		dbMockedData.UpdatedAt = time.Now()
	case float64:
	case string:
	default:
	}
	return nil
}

// Get ... Retrieves a specified record from the database for a given id
func (repo *BaseRepository) Get(id interface{}, model interface{}) error {
	switch model.(type) {
	case dto.Asset:
		dbMockedData := dto.Asset{}
		dbMockedData.ID = id.(dto.Asset).ID
		dbMockedData.CreatedAt = time.Now()
		dbMockedData.UpdatedAt = time.Now()
	case float64:
	case string:
	default:
	}
	return nil
}

// GetByFieldName ... Retrieves a record for the specified model from the database for a given field name
func (repo *BaseRepository) GetByFieldName(field interface{}, model interface{}) error {
	switch model.(type) {
	case dto.Asset:
		recordID := uuid.NewV4()
		dbMockedData := dto.Asset{}
		dbMockedData.ID = recordID
		dbMockedData.Symbol = field.(dto.Asset).Symbol
		dbMockedData.CreatedAt = time.Now()
		dbMockedData.UpdatedAt = time.Now()
	case float64:
	case string:
	default:
	}
	return nil
}

// FetchByFieldName ... Retrieves all records for the specified model from the database for a given field name
func (repo *BaseRepository) FetchByFieldName(field interface{}, model interface{}) error {
	switch model.(type) {
	case dto.Asset:
		recordID := uuid.NewV4()
		dbMockedData := dto.Asset{}
		dbMockedData.ID = recordID
		dbMockedData.CreatedAt = time.Now()
		dbMockedData.UpdatedAt = time.Now()
	case float64:
	case string:
	default:
	}
	return nil
}

// Fetch ... Retrieves all records from the database for a given models
func (repo *BaseRepository) Fetch(model interface{}) error {
	switch model.(type) {
	case dto.Asset:
		recordID := uuid.NewV4()
		dbMockedData := dto.Asset{}
		dbMockedData.ID = recordID
		dbMockedData.CreatedAt = time.Now()
		dbMockedData.UpdatedAt = time.Now()
	case float64:
	case string:
	default:
	}
	return nil
}

// Create ... Create a record on the database for a the given model
func (repo *BaseRepository) Create(model interface{}) error {
	switch model.(type) {
	case dto.Asset:
		recordID := uuid.NewV4()
		dbMockedData := dto.Asset{}
		dbMockedData.ID = recordID
		dbMockedData.CreatedAt = time.Now()
		dbMockedData.UpdatedAt = time.Now()
	case float64:
	case string:
	default:
	}
	return nil
}

// Update ... Update a specified record from the database for a given id
func (repo *BaseRepository) Update(id, model interface{}) error {
	switch model.(type) {
	case dto.Asset:
		recordID := uuid.NewV4()
		dbMockedData := dto.Asset{}
		dbMockedData.ID = recordID
		dbMockedData.CreatedAt = time.Now()
		dbMockedData.UpdatedAt = time.Now()
	case float64:
	case string:
	default:
	}
	return nil
}

// Delete ... Deletes a specified record from the database for a given id
func (repo *BaseRepository) Delete(model interface{}) error {
	switch model.(type) {
	case dto.Asset:
		recordID := uuid.NewV4()
		dbMockedData := dto.Asset{}
		dbMockedData.ID = recordID
		dbMockedData.CreatedAt = time.Now()
		dbMockedData.UpdatedAt = time.Now()
	case float64:
	case string:
	default:
	}
	return nil
}
