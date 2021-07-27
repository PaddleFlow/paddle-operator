package utils

import (
	"io"
	"net/http"
)

var DefaultClient = &HttpClient{}

type HttpClient struct {
	http.Client
}

func (c *HttpClient) Post(url, filename string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("filename", filename)
	req.Header.Set("Content-Type", "application/json")
	return c.Do(req)
}

func Post(url, filename string, body io.Reader) (resp *http.Response, err error) {
	return DefaultClient.Post(url, filename, body)
}

func Get(url string, filename string) (resp *http.Response, err error) {
	return DefaultClient.Get(url + "/" + filename)
}