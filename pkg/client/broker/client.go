package broker

import (
	"io/ioutil"
	"net/http"
	"os"
)

const (
	APISERVER_HOST      = "http://localhost:"
	APISERVER_PORT      = "EMQX_MANAGEMENT__LISTENER__HTTP"
	AUTHORIZATION_KEY   = "EMQX_MANAGEMENT__DEFAULT_APPLICATION__ID"
	AUTHORIZATION_VALUE = "EMQX_MANAGEMENT__DEFAULT_APPLICATION__SECRET"

	HANDLER_GET string = "GET"
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
	url := APISERVER_HOST + os.Getenv(APISERVER_PORT) + uri
	req, err := http.NewRequest(action, url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(os.Getenv(AUTHORIZATION_KEY), os.Getenv(AUTHORIZATION_VALUE))
	return req, err
}

// New returns a emqx broker client
func New() Client {
	return &client{}

}

func (c *client) GetEmqxBrokerClusterStats(action, uri string) ([]byte, error) {
	req, err := newRequest(HANDLER_GET, uri)
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
