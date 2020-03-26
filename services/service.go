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
	metaData := utility.GetRequestMetaData("generateToken", c.Config)
	if c.BaseURL.String() != fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action) {
		c.Logger.Info("Outgoing request to %s : %+v", c.BaseURL, body)
	}
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
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return resp, errors.New(fmt.Sprintf("%s", string(resBody)))
	}

	err = json.Unmarshal(resBody, v)
	metaData := utility.GetRequestMetaData("generateToken", c.Config)
	if c.BaseURL.String() != fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action) {
		c.Logger.Info("Incoming response from %s : %+v", c.BaseURL, v)
	}
	return resp, err

}
