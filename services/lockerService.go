package services

import (
	"encoding/json"
	"errors"
	"fmt"
	Config "wallet-adapter/config"
	"wallet-adapter/utility/apiClient"
	"wallet-adapter/utility/appError"
	"wallet-adapter/utility/cache"

	"wallet-adapter/database"
	"wallet-adapter/dto"
)

//NotificationService object
type LockerService struct {
	Cache      *cache.Memory
	Config     Config.Data
	Error      *dto.ExternalServicesRequestErr
	Repository database.IRepository
}

func NewLockerService(cache *cache.Memory, config Config.Data, repository database.IRepository, serviceErr *dto.ExternalServicesRequestErr) *LockerService {
	baseService := LockerService{
		Cache:      cache,
		Config:     config,
		Repository: repository,
		Error:      serviceErr,
	}
	return &baseService
}

// AcquireLock ... Calls locker service with information about the lock to lock down a transaction for processing
func (service *LockerService) AcquireLock(requestData dto.LockerServiceRequest, responseData *dto.LockerServiceResponse) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("acquireLock", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	if err := APIClient.Do(APIRequest, responseData); err != nil {
		appErr := err.(appError.Err)
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%s", err.Error())), service.Error); errUnmarshal != nil {
			return err
		}
		return serviceError(appErr.ErrCode, service.Error.Code, errors.New(service.Error.Message))
	}

	return nil
}

// RenewLock ... Calls locker service with information about the lock to lock down a transaction for processing
func (service *LockerService) RenewLock(requestData dto.LockerServiceRequest, responseData *dto.LockerServiceResponse) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("renewLockLease", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	if err := APIClient.Do(APIRequest, responseData); err != nil {
		appErr := err.(appError.Err)
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%s", err.Error())), service.Error); errUnmarshal != nil {
			return err
		}
		return serviceError(appErr.ErrCode, service.Error.Code, errors.New(service.Error.Message))
	}

	return nil
}

// ReleaseLock ... Calls locker service with information about the lock to lock down a transaction for processing
func (service *LockerService) ReleaseLock(requestData dto.LockReleaseRequest, responseData *dto.ServicesRequestSuccess) error {
	AuthService := NewAuthService(service.Cache, service.Config, service.Repository)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := GetRequestMetaData("releaseLock", service.Config)

	APIClient := apiClient.New(nil, service.Config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	if err := APIClient.Do(APIRequest, responseData); err != nil {
		appErr := err.(appError.Err)
		if errUnmarshal := json.Unmarshal([]byte(fmt.Sprintf("%s", err.Error())), service.Error); errUnmarshal != nil {
			return err
		}
		return serviceError(appErr.ErrCode, service.Error.Code, errors.New(service.Error.Message))
	}

	return nil
}
