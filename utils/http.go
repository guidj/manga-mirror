package utils

import (
	"io"
	"net/http"
	"os"
	"path"
)

// HttpClient performs GET requests for crawling agents
type HttpClient struct {
	userAgent string
	client    http.Client
}

// NewHttpClient creates and returns an instance of an HttpClient that makes requsts with a specified `UserAgent` string
func NewHttpClient(userAgent string) *HttpClient {
	return &HttpClient{userAgent, http.Client{}}
}

// RetrieveContent makes an Http GET request and returns a string of the content
func (httpClient *HttpClient) RetrieveContent(url string) (content string, err error) {
	var req *http.Request
	var resp *http.Response
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return string(""), err
	}
	req.Header.Set("User-Agent", httpClient.userAgent)
	resp, err = httpClient.client.Do(req)
	if err != nil {
		return string(""), err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return string(body), err
}

// Download downloads the contets from a Http GET request to a file
func (httpClient *HttpClient) Download(url string, filePath string) error {
	var req *http.Request
	var resp *http.Response
	var err error
	baseDir := path.Dir(filePath)
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		os.MkdirAll(baseDir, 0755)
	}
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", httpClient.userAgent)
	resp, err = httpClient.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	os.WriteFile(filePath, body, 0755)
	return nil
}
