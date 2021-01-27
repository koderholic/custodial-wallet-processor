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
	"strings"
	"time"
	Config "wallet-adapter/config"
	"wallet-adapter/dto"
	"wallet-adapter/utility"
)

//Controller : Controller struct
type BaseService struct {
	Cache  *utility.MemoryCache
	Logger *utility.Logger
	Config Config.Data
	Error  *dto.ServicesRequestErr
}

type Client struct {
	BaseURL    *url.URL
	UserAgent  string
	Logger     *utility.Logger
	Config     Config.Data
	httpClient *http.Client
	startTime  int64
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

func NewService(cache *utility.MemoryCache, logger *utility.Logger, config Config.Data) *BaseService {
	baseService := BaseService{
		Logger: logger,
		Cache:  cache,
		Config: config,
	}
	return &baseService
}

func (c *Client) NewRequest(method, path string, body interface{}) (*http.Request, error) {
	metaData := utility.GetRequestMetaData("generateToken", c.Config)
	if c.BaseURL.String() != fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action) {
		c.Logger.Info("Outgoing request to %s : %+v", c.BaseURL, body)
	}
	if strings.Contains(c.BaseURL.String(), "transactions/send") {
		//We need to increase timeout in Key Management also
		c.httpClient.Timeout = 120 * time.Second
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
	c.startTime = time.Now().UnixNano()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.Logger.Info("Response from %s : +%v", c.BaseURL, err)
		return nil, err
	}
	defer resp.Body.Close()

	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}

	metaData := utility.GetRequestMetaData("generateToken", c.Config)
	if c.BaseURL.String() != fmt.Sprintf("%s%s", metaData.Endpoint, metaData.Action) {
		duration := (time.Now().UnixNano() - c.startTime) / 1000000
		c.Logger.Info("Response from %s : [%d] %+s Time: %d", c.BaseURL, resp.StatusCode, resBody, duration)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return resp, errors.New(fmt.Sprintf("%s", string(resBody)))
	}

	err = json.Unmarshal(resBody, v)
	return resp, err

}
