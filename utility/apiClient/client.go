package apiClient

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
	"wallet-adapter/utility/logger"
)

// Client object for external API request
type Client struct {
	BaseURL    *url.URL
	UserAgent  string
	Config     Config.Data
	HttpClient *http.Client
	startTime  int64
}

func New(HttpClient *http.Client, config Config.Data, baseURL string) *Client {
	if HttpClient == nil {
		HttpClient = http.DefaultClient
	}
	c := &Client{HttpClient: HttpClient}
	c.Config = config
	c.BaseURL, _ = url.Parse(baseURL)

	return c
}

func (c *Client) NewRequest(method, path string, body interface{}) (*http.Request, error) {
	if strings.Contains(c.BaseURL.String(), "key-management/sign") {
		c.HttpClient.Timeout = 120 * time.Second
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
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		logger.Info("Response from %s : +%v", c.BaseURL, err)
		return nil, err
	}
	defer resp.Body.Close()

	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}

	if c.BaseURL.String() != "authentication/services/token" {
		duration := (time.Now().UnixNano() - c.startTime) / 1000000
		logger.Info("Response from %s : [%d] %+s Time: %d", c.BaseURL, resp.StatusCode, resBody, duration)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return resp, errors.New(fmt.Sprintf("%s", string(resBody)))
	}

	err = json.Unmarshal(resBody, v)
	return resp, err

}
