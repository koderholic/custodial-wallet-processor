package database

import (
	"errors"
	"strings"
	"wallet-adapter/utility"
)

type IRepository interface {
	Get(id interface{}, model interface{}) error
	Fetch(model interface{}) error
	Create(model interface{}) error
	Update(id interface{}, model interface{}) error
	Delete(model interface{}) error
}

type BaseRepository struct {
	Database
}

func (repo *BaseRepository) Get(id interface{}, model interface{}) error {
	if repo.DB.Where("id = ?", id).First(model).RecordNotFound() {
		return utility.AppError{
			ErrType: utility.INPUTERROR,
			Err:     errors.New("No record found for provided Id"),
		}
	}
	return nil
}

func (repo *BaseRepository) Fetch(model interface{}) error {
	if err := repo.DB.Find(model).Error; err != nil {
		repo.Logger.Error("Error with repository fetch %s", err)
		return utility.AppError{
			ErrType: utility.SYSTEMERROR,
			Err:     err,
		}
	}
	return nil
}

func (repo *BaseRepository) Create(model interface{}) error {
	result := repo.DB.Create(model)
	if result.Error != nil {
		repo.Logger.Error("Error with repository create %s", result.Error)
		return utility.AppError{
			ErrType: utility.INPUTERROR,
			Err:     errors.New(strings.Join(strings.Split(result.Error.Error(), " ")[2:], " ")),
		}
	}
	return nil
}

func (repo *BaseRepository) Update(id, model interface{}) error {

	if result := repo.DB.Model(model).Update(model); result.Error != nil {
		repo.Logger.Error("Error with repository update %s", result.Error)
		return utility.AppError{
			ErrType: utility.INPUTERROR,
			Err:     errors.New(strings.Join(strings.Split(result.Error.Error(), " ")[2:], " ")),
		}
	}
	repo.DB.Where("id = ?", id).First(model)
	return nil
}

func (repo *BaseRepository) Delete(model interface{}) error {
	if err := repo.DB.Delete(model).Error; err != nil {
		repo.Logger.Error("Error (with repository delete %s", err)
		return utility.AppError{
			ErrType: utility.INPUTERROR,
			Err:     err,
		}
	}
	return nil
}
