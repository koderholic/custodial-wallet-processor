package services

import (
	"encoding/json"
	"fmt"
	Config "wallet-adapter/config"
	"wallet-adapter/utility/apiClient"

	"wallet-adapter/dto"
	"wallet-adapter/utility"
)

//NotificationService object
type LockerService struct {
	Cache  *utility.MemoryCache
	Config Config.Data
	Error  *dto.ExternalServicesRequestErr
}

func NewLockerService(cache *utility.MemoryCache, config Config.Data) *LockerService {
	baseService := LockerService{
		Cache:  cache,
		Config: config,
	}
	return &baseService
}

// AcquireLock ... Calls locker service with information about the lock to lock down a transaction for processing
func (service *LockerService) AcquireLock(cache *utility.MemoryCache, config Config.Data, requestData dto.LockerServiceRequest, responseData *dto.LockerServiceResponse, serviceErr interface{}) error {
	AuthService := NewAuthService(service.Cache, service.Config)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("acquireLock", config)

	APIClient := apiClient.New(nil, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	_, err = APIClient.Do(APIRequest, responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), serviceErr); errUnmarshal != nil {
			return err
		}
		return err
	}

	return nil
}

// RenewLock ... Calls locker service with information about the lock to lock down a transaction for processing
func (service *LockerService) RenewLock(cache *utility.MemoryCache, config Config.Data, requestData dto.LockerServiceRequest, responseData *dto.LockerServiceResponse, serviceErr interface{}) error {
	AuthService := NewAuthService(service.Cache, service.Config)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("renewLockLease", config)

	APIClient := apiClient.New(nil, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	_, err = APIClient.Do(APIRequest, responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), serviceErr); errUnmarshal != nil {
			return err
		}
		return err
	}

	return nil
}

// ReleaseLock ... Calls locker service with information about the lock to lock down a transaction for processing
func (service *LockerService) ReleaseLock(cache *utility.MemoryCache, config Config.Data, requestData dto.LockReleaseRequest, responseData *dto.ServicesRequestSuccess, serviceErr interface{}) error {
	AuthService := NewAuthService(service.Cache, service.Config)
	authToken, err := AuthService.GetAuthToken()
	if err != nil {
		return err
	}
	metaData := utility.GetRequestMetaData("releaseLock", config)

	APIClient := apiClient.New(nil, config, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action))
	APIRequest, err := APIClient.NewRequest(metaData.Type, "", requestData)
	if err != nil {
		return err
	}
	APIClient.AddHeader(APIRequest, map[string]string{
		"x-auth-token": authToken,
	})
	_, err = APIClient.Do(APIRequest, responseData)
	if err != nil {
		if errUnmarshal := json.Unmarshal([]byte(err.Error()), serviceErr); errUnmarshal != nil {
			return err
		}
		return err
	}

	return nil
}
