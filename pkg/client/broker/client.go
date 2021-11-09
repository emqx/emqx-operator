package broker

import (
	"io/ioutil"
	"net/http"
)

const (
	APISERVER           = "http://localhost:8081"
	AUTHORIZATION_KEY   = "authorization"
	AuTHORIZATION_VALUE = "Basic YWRtaW46cHVibGlj"

	HandlerGET string = "GET"
)

// Client defines the functions necessary to connect to emqx brokers get or set what we need
type Client interface {
	GetEmqxBrokerClusterStats(uri, action string) ([]byte, error)
}

type client struct {
	Client *http.Client
}

// new request function
func newRequest(action, uri string) (*http.Request, error) {
	url := APISERVER + uri
	req, err := http.NewRequest(action, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(AUTHORIZATION_KEY, AuTHORIZATION_VALUE)

	return req, err
}

// New returns a emqx broker client
func New() Client {
	return &client{}

}

func (c *client) GetEmqxBrokerClusterStats(action, uri string) ([]byte, error) {
	req, err := newRequest(HandlerGET, uri)
	if err != nil {
		return nil, err
	}
	res, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)
	return body, nil

}
