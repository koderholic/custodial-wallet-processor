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

func NewLockerService(cache *cache.Memory, config Config.Data, repository database.IRepository) *LockerService {
	baseService := LockerService{
		Cache:      cache,
		Config:     config,
		Repository: repository,
		Error:      &dto.ExternalServicesRequestErr{},
	}
	return &baseService
}

// AcquireLock ... Calls locker service with information about the lock to lock down a transaction for processing
func (service *LockerService) acquireLock(requestData dto.LockerServiceRequest, responseData *dto.LockerServiceResponse) error {
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
func (service *LockerService) releaseLock(requestData dto.LockReleaseRequest, responseData *dto.ServicesRequestSuccess) error {
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

func (service *LockerService) AcquireLock(identifier string, ttl int64) (dto.LockerServiceResponse, error) {
	lockerServiceRequest := dto.LockerServiceRequest{
		Identifier:   fmt.Sprintf("%s%s", service.Config.LockerPrefix, identifier),
		ExpiresAfter: ttl,
	}
	lockerServiceResponse := dto.LockerServiceResponse{}
	if err := service.acquireLock(lockerServiceRequest, &lockerServiceResponse); err != nil {
		return dto.LockerServiceResponse{}, err
	}
	return lockerServiceResponse, nil
}

func (service *LockerService) ReleaseLock(identifier string, lockerserviceToken string) error {
	lockReleaseRequest := dto.LockReleaseRequest{
		Identifier: fmt.Sprintf("%s%s", service.Config.LockerPrefix, identifier),
		Token:      lockerserviceToken,
	}
	lockReleaseResponse := dto.ServicesRequestSuccess{}
	if err := service.releaseLock(lockReleaseRequest, &lockReleaseResponse); err != nil || !lockReleaseResponse.Success {
		return err
	}
	return nil
}
