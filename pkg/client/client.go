package client

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"
)

const (
	API_VERSION = "api/v4"
)

type EmqxAPI struct {
	id     string
	passwd string
	url    string
}

func New(url, id, passwd string) *EmqxAPI {
	return &EmqxAPI{
		url:    url,
		id:     id,
		passwd: passwd,
	}
}

func (e *EmqxAPI) Get(resource string, timeout time.Duration) ([]byte, error) {
	u, err := url.Parse(e.url)
	if err != nil {
		return nil, fmt.Errorf("parse error: %s", err.Error())
	}
	u.Path = path.Join(u.Path, API_VERSION, resource)

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("new request error: %s", err.Error())
	}
	req.SetBasicAuth(e.id, e.passwd)

	c := http.Client{Timeout: timeout * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http client error: %s", err.Error())
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body error: %s", err.Error())
	}
	return body, nil
}
