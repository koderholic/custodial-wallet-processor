package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	Config "wallet-adapter/config"
	"wallet-adapter/utility"
)

//ExternalAPICall ... Makes call to an external API
func ExternalAPICall(marshaledRequest []byte, requestFlag string, responseData interface{}, config Config.Data, log *utility.Logger) error {

	metaData := utility.GetRequestMetaData(requestFlag, config)
	log.Info("Request body sent to %s : %s", metaData.Endpoint+metaData.Action, string(marshaledRequest))

	client := &http.Client{}
	requestInstance, err := http.NewRequest(metaData.Type, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action), bytes.NewBuffer(marshaledRequest))
	if err != nil {
		log.Error("Error From %s : %s", metaData.Endpoint, err)
		return utility.AppError{
			ErrType: utility.SYSTEMERROR,
			Err:     err,
		}
	}

	requestInstance.Header.Set("Content-Type", "application/json")
	requestInstance.Header.Set("x-auth-token", "application/json")

	externalCallResponse, err := client.Do(requestInstance)
	if err != nil {
		log.Error("Error From %s : %s", metaData.Endpoint, err)
		return utility.AppError{
			ErrType: utility.SYSTEMERROR,
			Err:     err,
		}
	}
	defer externalCallResponse.Body.Close()

	body, _ := ioutil.ReadAll(externalCallResponse.Body)
	log.Info("Response From %s : %s", metaData.Endpoint+metaData.Action, string(body))

	json.Unmarshal(body, responseData)

	if externalCallResponse.StatusCode != http.StatusOK {
		err := "External request failed"
		return utility.AppError{
			ErrType: utility.INPUTERROR,
			Err:     errors.New(err),
		}
	}

	return nil
}
