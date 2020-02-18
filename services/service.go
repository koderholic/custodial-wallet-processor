package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	Config "wallet-adapter/config"
	"wallet-adapter/utility"
)

type Client struct {
	BaseURL    *url.URL
	UserAgent  string
	Logger     *utility.Logger
	Config     Config.Data
	httpClient *http.Client
}

func NewClient(httpClient *http.Client, logger *utility.Logger, config Config.Data, baseURL string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	c := &Client{httpClient: httpClient}
	c.Logger = logger
	c.Config = config
	c.BaseURL, _ = url.Parse(baseURL)

	return c
}

func (c *Client) NewRequest(method, path string, body interface{}) (*http.Request, error) {
	c.Logger.Info("Outgoing request to %s : %+v", c.BaseURL, body)
	rel := &url.URL{Path: path}
	u := c.BaseURL.ResolveReference(rel)
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	return req, nil
}

func (c *Client) AddHeader(req *http.Request, headers map[string]string) *http.Request {
	for header, value := range headers {
		req.Header.Set(header, value)
	}
	return req
}

func (c *Client) AddBasicAuth(req *http.Request, username, password string) *http.Request {
	req.SetBasicAuth(username, password)
	return req
}

func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}
	fmt.Println(" >>>> ", resp.StatusCode)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		fmt.Println(" !! >>>>>>>.. ", string(resBody))
		return resp, errors.New(fmt.Sprintf("%s", string(resBody)))
	}

	err = json.Unmarshal(resBody, v)
	c.Logger.Info("Incoming response from %s : %+v", c.BaseURL, v)
	return resp, err

}

//ExternalAPICall ... Makes call to an external API
func ExternalAPICall(marshaledRequest []byte, requestFlag string, responseData interface{}, config Config.Data, log *utility.Logger, authToken string) error {

	metaData := utility.GetRequestMetaData(requestFlag, config)
	log.Info("Request body sent to %s : %s", metaData.Endpoint+metaData.Action, string(marshaledRequest))

	client := &http.Client{}
	requestInstance, err := http.NewRequest(metaData.Type, fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action), bytes.NewBuffer(marshaledRequest))
	if err != nil {
		log.Error("Error From %s : %s", metaData.Endpoint, err)
		return utility.AppError{
			ErrType: utility.SYSTEM_ERR,
			Err:     err,
		}
	}

	requestInstance.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		requestInstance.Header.Set(utility.X_AUTH_TOKEN, authToken)
	}

	externalCallResponse, err := client.Do(requestInstance)
	if err != nil {
		log.Error("Error From %s : %s", metaData.Endpoint, err)
		return utility.AppError{
			ErrType: utility.SYSTEM_ERR,
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
			ErrType: utility.INPUT_ERR,
			Err:     errors.New(err),
		}
	}

	return nil
}
