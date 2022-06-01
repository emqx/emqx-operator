package client_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"

	"github.com/emqx/emqx-operator/pkg/client"
)

func TestGetBrokers(t *testing.T) {
	const (
		expected = `{"data":[{"version":"4.4.3","uptime":"15 minutes, 1 seconds","sysdescr":"EMQ X Enterprise","otp_release":"24.1.5/12.1.5","node_status":"Running","node":"emqx@10.0.29.133","datetime":"2022-05-25 07:34:13"},{"version":"4.4.3","uptime":"15 minutes, 11 seconds","sysdescr":"EMQ X Enterprise","otp_release":"24.1.5/12.1.5","node_status":"Running","node":"emqx@10.0.17.90","datetime":"2022-05-25 07:34:13"},{"version":"4.4.3","uptime":"14 minutes, 37 seconds","sysdescr":"EMQ X Enterprise","otp_release":"24.1.5/12.1.5","node_status":"Running","node":"emqx@10.0.30.111","datetime":"2022-05-25 07:34:13"}],"code":0}`
	)

	type BrokersResp struct {
		Data []interface{} `json:"data"`
		Code int           `json:"code"`
	}

	// Start a local HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Send response to be tested
		// just avoiding lint check
		_, _ = rw.Write([]byte(expected))
	}))
	// Close the server when test finishes
	defer server.Close()

	api := client.New(server.URL, "admin", "public")
	data, _ := api.Get("brokers", 10)

	expectedArr := gjson.Get(expected, "data").Array()
	actualArr := gjson.Get(string(data[:]), "data").Array()

	assert.Equal(t, len(expectedArr), len(actualArr))
}
