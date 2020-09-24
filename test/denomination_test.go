package test

// import (
// 	"bytes"
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"io/ioutil"
// 	"net/http"
// 	"net/http/httptest"
// 	"wallet-adapter/dto"

// 	_ "github.com/jinzhu/gorm/dialects/sqlite"
// 	"github.com/magiconair/properties/assert"
// 	"github.com/stretchr/testify/require"
// )

// func (s *Suite) Test_GetAddressForNonActiveAsset() {

// 	createAssetInputData := []byte(`{"assets" : ["LINK"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
// 	createAssetRequest, _ := http.NewRequest("POST", "/users/assets", bytes.NewBuffer(createAssetInputData))
// 	createAssetRequest.Header.Set("x-auth-token", authToken)

// 	response := httptest.NewRecorder()
// 	s.Router.ServeHTTP(response, createAssetRequest)

// 	resBody, err := ioutil.ReadAll(response.Body)
// 	if err != nil {
// 		require.NoError(s.T(), err)
// 	}
// 	createAssetResponse := dto.UserAssetResponse{}
// 	err = json.Unmarshal(resBody, &createAssetResponse)

// 	getNewAssetAddressRequest, _ := http.NewRequest("GET", fmt.Sprintf("/assets/%s/address", createAssetResponse.Assets[0].ID), bytes.NewBuffer([]byte("")))
// 	getNewAssetAddressRequest.Header.Set("x-auth-token", authToken)

// 	getAddressResponse := httptest.NewRecorder()
// 	s.Router.ServeHTTP(getAddressResponse, getNewAssetAddressRequest)
// 	resBody, err = ioutil.ReadAll(getAddressResponse.Body)
// 	if err != nil {
// 		require.NoError(s.T(), err)
// 	}
// 	getNewAssetAddressResponse := map[string]string{}
// 	err = json.Unmarshal(resBody, &getNewAssetAddressResponse)

// 	assert.Equal(s.T(), http.StatusBadRequest, getAddressResponse.Code, "Expected request to fail with 400 error")
// }

// func (s *Suite) Test_ExternalTransferForNonActiveAsset() {
// 	createAssetInputData := []byte(`{"assets" : ["LINK","ETH","BTC"],"userId" : "a10fce7b-7844-43af-9ed1-e130723a1ea3"}`)
// 	createAssetRequest, _ := http.NewRequest("POST", "/users/assets", bytes.NewBuffer(createAssetInputData))
// 	createAssetRequest.Header.Set("x-auth-token", authToken)
// 	createResponse := httptest.NewRecorder()
// 	s.Router.ServeHTTP(createResponse, createAssetRequest)
// 	resBody, err := ioutil.ReadAll(createResponse.Body)
// 	if err != nil {
// 		require.NoError(s.T(), err)
// 	}
// 	createAssetResponse := dto.UserAssetResponse{}
// 	err = json.Unmarshal(resBody, &createAssetResponse)
// 	if createResponse.Code != http.StatusCreated || len(createAssetResponse.Assets) < 1 {
// 		require.NoError(s.T(), errors.New("Expected asset creation to not error"))
// 	}

// 	creditAssetInputData := []byte(fmt.Sprintf(`{"assetId" : "%s","value" : 200.30,"transactionReference" : "ra29bv7y111p945e17514","memo" :"Test credit transaction"}`, createAssetResponse.Assets[0].ID))
// 	creditAssetRequest, _ := http.NewRequest("POST", "/assets/credit", bytes.NewBuffer(creditAssetInputData))
// 	creditAssetRequest.Header.Set("x-auth-token", authToken)
// 	creditAssetResponse := httptest.NewRecorder()
// 	s.Router.ServeHTTP(creditAssetResponse, creditAssetRequest)
// 	if creditAssetResponse.Code != http.StatusOK {
// 		require.NoError(s.T(), errors.New("Expected credit asset to not error"))
// 	}

// 	debitAssetInputData := []byte(fmt.Sprintf(`{"assetId" : "%s","value" : 10.30,"transactionReference" : "ra29bv7y111p945e17515","memo" :"Test credit transaction"}`, createAssetResponse.Assets[0].ID))
// 	debitAssetRequest, _ := http.NewRequest("POST", "/assets/debit", bytes.NewBuffer(debitAssetInputData))
// 	debitAssetRequest.Header.Set("x-auth-token", authToken)
// 	debitAssetResponse := httptest.NewRecorder()
// 	s.Router.ServeHTTP(debitAssetResponse, debitAssetRequest)
// 	if debitAssetResponse.Code != http.StatusOK {
// 		require.NoError(s.T(), errors.New("Expected debit asset to not error"))
// 	}

// 	externalTransferInputData := []byte(`{"recipientAddress" : "bnb1k05t5h6h7t4mq9tvafz2mx8c29jz2w4r0l0hda","value" : 10.00,"debitReference" : "ra29bv7y111p945e17515","transactionReference" : "ra29bv7y111p945e17516"}`)
// 	externalTransferRequest, _ := http.NewRequest("POST", "/assets/transfer-external", bytes.NewBuffer(externalTransferInputData))
// 	externalTransferRequest.Header.Set("x-auth-token", authToken)
// 	externalTransferResponse := httptest.NewRecorder()
// 	s.Router.ServeHTTP(externalTransferResponse, externalTransferRequest)

// 	assert.Equal(s.T(), http.StatusBadRequest, externalTransferResponse.Code, "Expected request to fail with 400 error")
// }
